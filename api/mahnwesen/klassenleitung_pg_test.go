package mahnwesen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

// Eine gültige Adresse wird gespeichert, eine mit Namen davor als blanke Adresse: An sie gehen
// die Mahnlisten, und der Versand nimmt nur die Adresse.
func TestKlassenleitungSetzen_SpeichertDieBlankeAdresse(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM klassen_lehrer_mapping WHERE klasse = 'PRB1'`); err != nil {
			t.Errorf("Zuordnung entfernen: %v", err)
		}
	})

	for _, fall := range []struct{ eingabe, gespeichert string }{
		{"mueller@schule.example", "mueller@schule.example"},
		{"Herr Müller <meier@schule.example>", "meier@schule.example"},
	} {
		rumpf := `{"klasse":"PRB1","lehrer_email":"` + fall.eingabe + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/klassen-mapping", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.KlassenleitungSetzen()(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%q = %d %s, erwartet 200", fall.eingabe, rec.Code, rec.Body.String())
		}
		var adresse string
		if err := pool.QueryRow(ctx, `SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = 'PRB1'`).Scan(&adresse); err != nil {
			t.Fatalf("Zuordnung lesen: %v", err)
		}
		if adresse != fall.gespeichert {
			t.Errorf("%q gespeichert als %q, erwartet %q", fall.eingabe, adresse, fall.gespeichert)
		}
	}
}
