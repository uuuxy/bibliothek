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

// Die Kachel „Überfällige Ausleihen" teilt die Zahl durch die laufenden Ausleihen aus
// /api/statistiken, und die folgen dem Bestandsfilter. Ohne denselben Filter stand eine Zahl
// der ganzen Schule über einer aus Freihand oder LMF („≈ 240 % der laufenden Ausleihen"). Ein
// Gerät hat kein Exemplar: Ohne Filter zählt es mit, mit Filter gehört es zu keinem Bestand.
func TestMahnUebersicht_FolgtDemBestandsfilter(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	var gestern time.Time
	if err := pool.QueryRow(ctx, `SELECT now() - interval '1 day'`).Scan(&gestern); err != nil {
		t.Fatalf("Uhr der Datenbank: %v", err)
	}
	leser := testdaten.LegeSchuelerAn(t, pool, "M-FILTER", "Filter", "07A")
	schulbuch := testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Filter Schulbuch", gestern)
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Filter Roman 1", gestern)
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Filter Roman 2", gestern)
	if _, err := pool.Exec(ctx, `
		UPDATE buecher_titel SET ist_lernmittel = true
		WHERE id = (SELECT e.titel_id FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id WHERE a.id = $1)`,
		schulbuch); err != nil {
		t.Fatalf("Schulbuch kennzeichnen: %v", err)
	}
	var geraet string
	if err := pool.QueryRow(ctx, `INSERT INTO geraete (modellname, barcode_id) VALUES ('Filter Tablet', 'G-FILTER') RETURNING id::text`).Scan(&geraet); err != nil {
		t.Fatalf("Gerät anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist) VALUES ($1, $2, $3)`,
		geraet, leser, gestern); err != nil {
		t.Fatalf("Geräteausleihe anlegen: %v", err)
	}

	for adresse, erwartet := range map[string]int{
		"/api/dashboard/summary":               4,
		"/api/dashboard/summary?type=lmf":      1,
		"/api/dashboard/summary?type=freihand": 2,
	} {
		rec := httptest.NewRecorder()
		srv.MahnUebersicht().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, adresse, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", adresse, rec.Code, rec.Body.String())
		}
		var ist mahnUebersicht
		if err := json.Unmarshal(rec.Body.Bytes(), &ist); err != nil {
			t.Fatalf("Antwort unlesbar: %v", err)
		}
		if ist.Ueberfaellig != erwartet {
			t.Errorf("GET %s: %d überfällig, erwartet %d", adresse, ist.Ueberfaellig, erwartet)
		}
	}
}
