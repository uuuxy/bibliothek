package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Das Ausbuchen eines Exemplars: zwei Funde des Raster-Durchgangs 31.08.2026.
//
//  1. Der Handler verglich den Fehlertext wörtlich („Exemplar ist aktuell noch
//     verliehen!") mit einem Repo-Text in anderer Schreibung („exemplar ist aktuell noch
//     verliehen") — der Vergleich traf NIE. Ein verliehenes Buch endete als 500, und der
//     Sanitizer ersetzte die Auskunft durch „Ein interner Datenbankfehler ist
//     aufgetreten". Die Nachbar-Funktion (DeleteTitle) hatte denselben Bug schon einmal —
//     und wurde per Sentinel repariert; diese hier blieb beim fragilen Text.
//  2. Phantom-Erfolg: Bei unbekannter oder bereits ausgesonderter ID traf der UPDATE
//     null Zeilen, RowsAffected prüfte niemand — Antwort 200 samt Audit-Eintrag über
//     eine Löschung, die nie stattfand. Ein Doppelklick beschrieb zudem zustand_notiz
//     erneut mit „Systematisch gelöscht".
func TestDeleteCopy_VerliehenIst400_PhantomIst404(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	titelID := seedMonitorTitel(t, pool, "Ausbuch-Titel", "Jug Aus", false, 0)
	verliehen := exemplar(t, pool, titelID, "DEL-VERLIEHEN", true, "")
	frei := exemplar(t, pool, titelID, "DEL-FREI", true, "Eselsohr S. 12")
	sid := seedSchueler(t, pool, "S-DEL-1", "Mia", "5a")
	seedLeserAusleihe(t, pool, verliehen, sid)
	bearbeiter := seedPortalLehrkraft(t, pool, "ausbucher@test.invalid")

	loesche := func(id string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, "/api/buecher/exemplare/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: bearbeiter}))
		rec := httptest.NewRecorder()
		srv.DeleteCopyHandler(repository.NewAuditRepository(pool))(rec, req)
		return rec
	}

	// 1. Verliehen: 400 mit sprechender Auskunft — nicht 500 mit Sanitizer-Text.
	rec := loesche(verliehen)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("verliehenes Exemplar: HTTP %d, erwartet 400 — Text-Vergleich trifft nicht? (%s)",
			rec.Code, rec.Body.String())
	} else if !strings.Contains(rec.Body.String(), "verliehen") {
		t.Errorf("400 ohne Auskunft „verliehen\": %s", rec.Body.String())
	}

	// 2. Unbekannte ID: 404 und KEIN Audit-Eintrag über eine Löschung, die nie stattfand.
	phantom := "00000000-0000-0000-0000-00000000dead"
	rec = loesche(phantom)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unbekannte ID: HTTP %d, erwartet 404 (Phantom-Erfolg?): %s", rec.Code, rec.Body.String())
	}
	if n := auditEintraege(t, pool, phantom); n != 0 {
		t.Errorf("Phantom-Löschung hat %d Audit-Einträge erzeugt — die Historie behauptet eine Löschung, die nie stattfand", n)
	}

	// 3. Erfolg: 200, ausgesondert, genau ein Audit-Eintrag …
	if rec = loesche(frei); rec.Code != http.StatusOK {
		t.Fatalf("freies Exemplar: HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if n := auditEintraege(t, pool, frei); n != 1 {
		t.Errorf("nach dem Ausbuchen: %d Audit-Einträge, erwartet 1", n)
	}

	// … und der Doppelklick ist ein 404, kein zweiter „Erfolg" mit zweitem Audit-Eintrag.
	if rec = loesche(frei); rec.Code != http.StatusNotFound {
		t.Errorf("Doppelklick: HTTP %d, erwartet 404: %s", rec.Code, rec.Body.String())
	}
	if n := auditEintraege(t, pool, frei); n != 1 {
		t.Errorf("Doppelklick hat den Audit-Bestand auf %d erhöht", n)
	}
}

func auditEintraege(t *testing.T, pool *pgxpool.Pool, datensatzID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_log WHERE tabelle = 'buecher_exemplare' AND datensatz_id = $1`,
		datensatzID).Scan(&n); err != nil {
		t.Fatalf("Audit zählen: %v", err)
	}
	return n
}
