package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

func TestMahnUebersicht_AggregiertKorrekt(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	leser := testdaten.LegeSchuelerAn(t, pool, "M-LESER", "Mahn", "1A")

	now := time.Now()
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 1", now.AddDate(0, 0, -5))
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 2", now.AddDate(0, 0, -10))
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 3", now.AddDate(0, 0, -20))
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 4", now.AddDate(0, 0, -45))
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 5", now.AddDate(0, 0, -70))
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 6", now.AddDate(0, 0, -80))

	// Nicht überfällig
	testdaten.LegeAusleiheMitFristAn(t, pool, leser, "Buch 7", now.AddDate(0, 0, 5))

	rec := httptest.NewRecorder()
	srv.MahnUebersicht().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body.String())
	}

	var antwort struct {
		Ueberfaellig        int `json:"total_overdue"`
		MaxTageUeberfaellig int `json:"max_tage_overdue"`
		Zeitspannen         []struct {
			Bezeichnung string `json:"label"`
			Anzahl      int    `json:"count"`
		} `json:"overdue_buckets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}

	if antwort.Ueberfaellig != 6 {
		t.Errorf("Ueberfaellig: %d, erwartet 6", antwort.Ueberfaellig)
	}

	if antwort.MaxTageUeberfaellig < 79 || antwort.MaxTageUeberfaellig > 81 {
		t.Errorf("MaxTageUeberfaellig: %d, erwartet ca. 80", antwort.MaxTageUeberfaellig)
	}

	if len(antwort.Zeitspannen) != 4 {
		t.Fatalf("Zeitspannen: %d, erwartet 4", len(antwort.Zeitspannen))
	}

	for _, z := range antwort.Zeitspannen {
		switch z.Bezeichnung {
		case "1–14 Tage":
			if z.Anzahl != 2 {
				t.Errorf("1-14 Tage: %d, erwartet 2", z.Anzahl)
			}
		case "15–30 Tage":
			if z.Anzahl != 1 {
				t.Errorf("15-30 Tage: %d, erwartet 1", z.Anzahl)
			}
		case "31–60 Tage":
			if z.Anzahl != 1 {
				t.Errorf("31-60 Tage: %d, erwartet 1", z.Anzahl)
			}
		case "über 60 Tage":
			if z.Anzahl != 2 {
				t.Errorf("über 60 Tage: %d, erwartet 2", z.Anzahl)
			}
		default:
			t.Errorf("Unerwartete Zeitspanne: %s", z.Bezeichnung)
		}
	}
}
