package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/google/go-cmp/cmp"
)

// Die Mahnübersicht zählt die überfälligen Ausleihen je Zeitspanne und nennt die längste
// Überfälligkeit (repository.LadeMahnKennzahlen). Nicht mit zählen eine Dauerleihe, die nie
// überfällig wird, eine zurückgegebene Ausleihe und eine, deren Frist noch läuft. Jede Frist
// liegt einen halben Tag neben der vollen Zahl, damit eine Abweichung zwischen der Uhr der
// Datenbank und der des Tests keine Grenze verschiebt.
func TestMahnUebersicht_ZaehltUeberfaelligeJeZeitspanne(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var jetzt time.Time
	if err := pool.QueryRow(ctx, `SELECT now()`).Scan(&jetzt); err != nil {
		t.Fatalf("Uhr der Datenbank: %v", err)
	}
	vorTagen := func(tage int) time.Time {
		return jetzt.Add(-time.Duration(tage)*24*time.Hour - 12*time.Hour)
	}
	leser := testdaten.LegeSchuelerAn(t, pool, "M-UEBERSICHT", "Mahn", "07A")
	for titel, tage := range map[string]int{
		"Mahn 5": 5, "Mahn 10": 10, "Mahn 20": 20, "Mahn 45": 45, "Mahn 70": 70, "Mahn 80": 80,
	} {
		testdaten.LegeAusleiheMitFristAn(t, pool, leser, titel, vorTagen(tage))
	}
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Mahn offen", jetzt.Add(5*24*time.Hour))
	dauerleihe := testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Mahn Dauerleihe", vorTagen(90))
	zurueck := testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Mahn zurück", vorTagen(90))
	for anweisung, id := range map[string]string{
		`UPDATE ausleihen SET ist_handapparat = true WHERE id = $1`: dauerleihe,
		`UPDATE ausleihen SET rueckgabe_am = now() WHERE id = $1`:   zurueck,
	} {
		if _, err := pool.Exec(ctx, anweisung, id); err != nil {
			t.Fatalf("%s: %v", anweisung, err)
		}
	}

	rec := httptest.NewRecorder()
	srv.MahnUebersicht().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/dashboard/summary = %d: %s", rec.Code, rec.Body.String())
	}
	var ist mahnUebersicht
	if err := json.Unmarshal(rec.Body.Bytes(), &ist); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}

	erwartet := mahnUebersicht{
		Ueberfaellig:        6,
		MaxTageUeberfaellig: 80,
		Zeitspannen: []ueberfaelligZeitspanne{
			{Bezeichnung: "1–14 Tage", Anzahl: 2},
			{Bezeichnung: "15–30 Tage", Anzahl: 1},
			{Bezeichnung: "31–60 Tage", Anzahl: 1},
			{Bezeichnung: "über 60 Tage", Anzahl: 2},
		},
	}
	if diff := cmp.Diff(erwartet, ist); diff != "" {
		t.Errorf("Mahnübersicht (-erwartet +ist):\n%s", diff)
	}
}
