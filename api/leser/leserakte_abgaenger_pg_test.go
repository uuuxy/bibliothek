package leser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// Der Dialog „Zusammenführen" liest am offenen Datensatz ist_abgaenger aus der Akte. Fehlt das
// Feld in der Antwort, steht die Marke „Abgänger" nur am Treffer der Suche. Geprüft am rohen
// JSON: Der Name des Felds ist die Zusage an die Oberfläche.
func TestLeserakteNenntAbgaenger(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	akte := func(t *testing.T, id string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
			&auth.Claims{KontoID: id, Rolle: auth.RolleAdmin}))
		rec := httptest.NewRecorder()
		srv.Leserakte(repository.NewLeserAbfragen(pool))(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Akte %s: %d %s", id, rec.Code, rec.Body.String())
		}
		var antwort map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Akte lesen: %v", err)
		}
		return antwort
	}

	faelle := []struct {
		name string
		id   string
		soll bool
	}{
		{"Abgänger", testdaten.LegeAbgaengerAn(t, pool, "A-AKTE-ABG", "Abgang", "ABG"), true},
		{"Schüler der Schule", testdaten.LegeSchuelerAn(t, pool, "A-AKTE-AKTIV", "Aktiv", "07A"), false},
	}
	for _, fall := range faelle {
		t.Run(fall.name, func(t *testing.T) {
			if ist := akte(t, fall.id)["ist_abgaenger"]; ist != fall.soll {
				t.Errorf("ist_abgaenger = %v, erwartet %v", ist, fall.soll)
			}
		})
	}
}
