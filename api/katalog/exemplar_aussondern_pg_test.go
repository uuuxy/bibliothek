package katalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Aussondern eines Exemplars über DELETE, mit zwei Fehlern, die dieser Test festhält:
//
//  1. Ein verliehenes Buch endet als 400 mit Auskunft. Erkennte die Tür den Fehler am Text
//     der Meldung statt am Fehlerwert, träfe der Vergleich bei jeder anderen Schreibung nie,
//     das Buch endete als 500, und die Bereinigung der Fehler ersetzte die Auskunft durch
//     „Ein interner Datenbankfehler ist aufgetreten".
//  2. Eine unbekannte oder schon ausgesonderte Kennung ist 404. Prüfte niemand die Zahl der
//     getroffenen Zeilen, antwortete die Tür 200 samt Eintrag im Protokoll über eine
//     Aussonderung, die nie stattfand, und ein Doppelklick beschriebe zustand_notiz erneut.
func TestExemplarAussondern_VerliehenIst400_PhantomIst404(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	titelID := testdaten.LegeTitelMitCoverUndAlterAn(t, pool, "Ausbuch-Titel", "Jug Aus", false, 0)
	verliehen := testdaten.LegeExemplarAn(t, pool, titelID, "DEL-VERLIEHEN", true, "")
	frei := testdaten.LegeExemplarAn(t, pool, titelID, "DEL-FREI", true, "Eselsohr S. 12")
	sid := testdaten.LegeSchuelerAn(t, pool, "S-DEL-1", "Mia", "5a")
	testdaten.LegeAusleiheVorZweiTagenAn(t, pool, verliehen, sid)
	bearbeiter := testdaten.LegeKollegiumskontoAn(t, pool, "ausbucher@test.invalid")

	loesche := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/buecher/exemplare/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{KontoID: bearbeiter}))
		rec := httptest.NewRecorder()
		srv.ExemplarAussondern(repository.NewProtokollAbfragen(pool))(rec, req)
		return rec
	}

	// 1. Verliehen: 400 mit sprechender Auskunft, nicht 500 mit dem Ersatztext aus apierrors.
	rec := loesche(verliehen)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("verliehenes Exemplar: HTTP %d, erwartet 400 — Text-Vergleich trifft nicht? (%s)",
			rec.Code, rec.Body.String())
	} else if !strings.Contains(rec.Body.String(), "verliehen") {
		t.Errorf("400 ohne Auskunft „verliehen\": %s", rec.Body.String())
	}

	// 2. Unbekannte Kennung: 404 und kein Eintrag im Protokoll über eine Aussonderung, die
	// nie stattfand.
	phantom := "00000000-0000-0000-0000-00000000dead"
	rec = loesche(phantom)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unbekannte ID: HTTP %d, erwartet 404 (Phantom-Erfolg?): %s", rec.Code, rec.Body.String())
	}
	if n := protokollEintraege(t, pool, phantom); n != 0 {
		t.Errorf("Phantom-Löschung hat %d Audit-Einträge erzeugt — die Historie behauptet eine Löschung, die nie stattfand", n)
	}

	// 3. Erfolg: 200, ausgesondert, genau ein Eintrag im Protokoll …
	if rec = loesche(frei); rec.Code != http.StatusOK {
		t.Fatalf("freies Exemplar: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if n := protokollEintraege(t, pool, frei); n != 1 {
		t.Errorf("nach dem Ausbuchen: %d Audit-Einträge, erwartet 1", n)
	}

	// … und der Doppelklick ist ein 404, kein zweiter „Erfolg" mit zweitem Eintrag.
	if rec = loesche(frei); rec.Code != http.StatusNotFound {
		t.Errorf("Doppelklick: HTTP %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if n := protokollEintraege(t, pool, frei); n != 1 {
		t.Errorf("Doppelklick hat den Audit-Bestand auf %d erhöht", n)
	}
}

func protokollEintraege(t *testing.T, pool *pgxpool.Pool, datensatzID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_log WHERE tabelle = 'buecher_exemplare' AND datensatz_id = $1`,
		datensatzID).Scan(&n); err != nil {
		t.Fatalf("Audit zählen: %v", err)
	}
	return n
}
