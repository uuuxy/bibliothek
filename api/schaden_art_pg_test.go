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
)

// Die Fallgruppe einer Forderung („nicht zurückgegeben" oder „beschädigt") entscheidet,
// welches Kästchen der Schadensersatz-Bescheid ankreuzt und ob er die Rückgabe verlangt.
//
// Bis zum 10.09.2026 schrieb kein Produktivpfad die Spalte schadensfaelle.art: Der Dialog
// „Verlust/Schaden melden" hatte nur einen Freitext (Vorgabe „Verloren"), und jede neue
// Forderung bekam den DEFAULT 'beschaedigt'. Der Bescheid behauptete dann gegenüber den
// Eltern, das Kind habe ein verlorenes Buch „so stark beschädigt zurückgegeben"
// (Bestands-Durchgang, Bugklasse „Zustand ohne Eingang").
func TestSchadenMelden_ArtKommtAusDerMeldung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	admin := adminFuerAudit(t, pool)

	melde := func(barcode, rumpfZusatz string) (*httptest.ResponseRecorder, string) {
		t.Helper()
		sid := seedSchueler(t, pool, "S-ART-"+barcode, "Artkind"+barcode, "07A")
		ex := exemplar(t, pool, titelMitMeldebestand(t, pool, "Arttitel-"+barcode, 1), "EX-ART-"+barcode, true, "")
		var loan string
		if err := pool.QueryRow(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			VALUES ($1, $2, now() + interval '14 days') RETURNING id`, ex, sid).Scan(&loan); err != nil {
			t.Fatal(err)
		}
		rumpf := `{"loan_id":"` + loan + `","schueler_id":"` + sid + `","copy_id":"` + ex +
			`","beschreibung":"Verloren","betrag":15` + rumpfZusatz + `}`
		req := httptest.NewRequest(http.MethodPost, "/api/damage/report", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.ReportDamageHandler(repository.NewDamageRepository(pool))(rec, req)
		return rec, loan
	}

	for _, art := range []string{"nicht_zurueckgegeben", "beschaedigt"} {
		rec, loan := melde(art, `,"art":"`+art+`"`)
		if rec.Code != http.StatusOK {
			t.Fatalf("art=%s: Status %d: %s", art, rec.Code, rec.Body.String())
		}
		var gespeichert string
		if err := pool.QueryRow(ctx, `SELECT art FROM schadensfaelle WHERE ausleihe_id = $1`, loan).Scan(&gespeichert); err != nil {
			t.Fatal(err)
		}
		if gespeichert != art {
			t.Errorf("gemeldet %q, gespeichert %q", art, gespeichert)
		}
	}

	// Ohne Angabe kein stiller Vorgabewert: Die Fallgruppe steht im Brief an die Eltern.
	if rec, _ := melde("ohne", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("Meldung ohne art: Status %d, want 400: %s", rec.Code, rec.Body.String())
	}
}
