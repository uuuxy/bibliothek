package schaden

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
)

// Die Fallgruppe einer Forderung („nicht zurückgegeben" oder „beschädigt") entscheidet,
// welches Kästchen der Schadensersatz-Bescheid ankreuzt und ob er die Rückgabe verlangt.
//
// Die Meldung muss sie schreiben: Mit der Vorgabe 'beschaedigt' der Spalte behauptete der
// Bescheid gegenüber den Eltern, das Kind habe ein verlorenes Buch „so stark beschädigt
// zurückgegeben".
func TestSchadenMelden_ArtKommtAusDerMeldung(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	admin := testdaten.LegeBearbeiterAn(t, pool)

	melde := func(barcode, rumpfZusatz string) (*httptest.ResponseRecorder, string) {
		t.Helper()
		sid := testdaten.LegeSchuelerAn(t, pool, "S-ART-"+barcode, "Artkind"+barcode, "07A")
		ex := testdaten.LegeExemplarAn(t, pool, testdaten.LegeTitelAn(t, pool, "Arttitel-"+barcode, 1), "EX-ART-"+barcode, true, "")
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
		srv.SchadenMelden(repository.NewSchadensAbfragen(pool))(rec, req)
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
