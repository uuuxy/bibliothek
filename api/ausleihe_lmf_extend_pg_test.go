package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/service"
	"bibliothek/internal/testdaten"
)

// TestGlobalExtendLMF_SchreibvariantenRobust sichert die zweite Hälfte von Bug #5 ab:
// Die Massen-Verlängerung erkannte LMF-Titel per `ILIKE 'LMF-%'` und übersprang damit
// still ein manuell angelegtes "LMF - Deutsch 5" (Leerzeichen nach LMF) — genau das
// Szenario aus der Meldung. Ausleihe/Frist erkannten das Buch zwar korrekt als LMF,
// die Verlängerung ließ es aber auf der alten Frist stehen → es wurde überfällig und
// landete in der Mahnung. Der Filter muss über pkg/lmf laufen, konsistent zum Rest.
func TestGlobalExtendLMF_SchreibvariantenRobust(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.ResetBestandsdaten(t, pool)

	const klasse = "5a"
	sid := testdaten.SeedSchueler(t, pool, "S-LMF-1", "Fritz", klasse)

	alteFrist := time.Date(2025, 9, 1, 23, 59, 59, 0, time.UTC)

	// Drei LMF-Schreibvarianten (alle müssen verlängert werden) + ein Freihand-Titel
	// (darf NICHT angefasst werden).
	bindestrich := testdaten.SeedAusleihe(t, pool, sid, "LMF-Mathe 5", alteFrist)
	leerBindestrich := testdaten.SeedAusleihe(t, pool, sid, "LMF - Deutsch 5", alteFrist)
	leer := testdaten.SeedAusleihe(t, pool, sid, "LMF Bio 5", alteFrist)
	freihand := testdaten.SeedAusleihe(t, pool, sid, "Der Hobbit", alteFrist)

	srv := &Server{Basis: tuer.Basis{DB: &db.Database{Pool: pool}}}
	req := httptest.NewRequest(http.MethodPost, "/api/ausleihen/lmf/global-extend",
		strings.NewReader(`{"klasse":"05A","neues_rueckgabe_datum":"2026-07-31"}`))
	rec := httptest.NewRecorder()
	srv.GlobalExtendLMFHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	// Berliner Tagesende zum Datum (Sommer/CEST = 21:59:59 UTC), wie der Handler es
	// seit dem Zeit-Sweep normalisiert — nicht mehr roh 23:59:59 UTC.
	neu := service.TagesEndeInSchulzeitzone(time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC))
	for _, tc := range []struct {
		name       string
		id         string
		verlängert bool
	}{
		{"LMF-Mathe 5 (Bindestrich)", bindestrich, true},
		{"LMF - Deutsch 5 (Leer-Bindestrich)", leerBindestrich, true},
		{"LMF Bio 5 (Leerzeichen)", leer, true},
		{"Der Hobbit (Freihand)", freihand, false},
	} {
		frist := testdaten.FristVon(t, pool, tc.id)
		if tc.verlängert && !frist.Equal(neu) {
			t.Errorf("%s: Frist nicht verlängert — erwartet %s, war %s (LMF-Schreibvariante übersehen)",
				tc.name, neu, frist)
		}
		if !tc.verlängert && !frist.Equal(alteFrist) {
			t.Errorf("%s: Freihand-Buch fälschlich verlängert — war %s", tc.name, frist)
		}
	}
}
