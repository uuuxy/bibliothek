package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/mahnwesen"
	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
)

// Paar-Gate: Die Abgängerliste und die Versetzung müssen dieselbe Menge sehen. Die Liste
// zeigt im Juni, wen man einsammelt; die Versetzung markiert im Sommer, wer geht. Weichen
// sie ab, bekommt eine Klasse Kontoauszüge und bleibt — oder geht ohne Kontoauszug.
// Beide lesen repository.AbschlussklasseSQL; dieser Test hält fest, dass das so bleibt.
func TestAbgaengerliste_UndVersetzungSehenDieselbeMenge(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	for i, k := range []string{"9H1", "09h2", "10H1", "10R1", "10R2", "13", "10G1", "8H1", "9R1", "5F1", "12"} {
		testdaten.LegeSchuelerMitAusleiheAn(t, pool, "P-"+string(rune('A'+i)), "Kind"+k, k)
	}

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}, Uhr: testdaten.SaisonUhr}}
	rec := httptest.NewRecorder()
	srv.mahnwesen().Abgaenger()(rec, httptest.NewRequest(http.MethodGet, "/api/abgaenger", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Abgängerliste: Status %d — %s", rec.Code, rec.Body.String())
	}
	var liste mahnwesen.AbgaengerAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
		t.Fatalf("Abgängerliste unlesbar: %v", err)
	}
	ausListe := map[string]bool{}
	for _, z := range liste.Abgaenger {
		ausListe[z.Vorname] = true
	}

	admin := testdaten.LegeAdminKontoAn(t, pool, "paar-versetzung@test.invalid", "Paar", "Versetzung")
	if rec := tuertest.RufeAlsAdmin(t, admin, srv.leser().Schuljahreswechsel(), http.MethodPost,
		"/api/students/promote", `{"confirm": true}`, nil); rec.Code != http.StatusOK {
		t.Fatalf("Versetzung: Status %d — %s", rec.Code, rec.Body.String())
	}
	rows, err := pool.Query(ctx, `SELECT vorname FROM schueler WHERE ist_abgaenger = true`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	ausVersetzung := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		ausVersetzung[v] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(ausListe) == 0 {
		t.Fatal("die Liste ist leer — das Gate vergliche zwei Nullmengen")
	}
	for name := range ausListe {
		if !ausVersetzung[name] {
			t.Errorf("%s steht in der Abgängerliste, die Versetzung lässt ihn aber bleiben", name)
		}
	}
	for name := range ausVersetzung {
		if !ausListe[name] {
			t.Errorf("die Versetzung schickt %s von der Schule, die Abgängerliste hat ihn nie gezeigt", name)
		}
	}
}
