package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestGlobalExtendLMF_SchreibvariantenRobust sichert die zweite Hälfte von Bug #5 ab:
// Die Massen-Verlängerung erkannte LMF-Titel per `ILIKE 'LMF-%'` und übersprang damit
// still ein manuell angelegtes "LMF - Deutsch 5" (Leerzeichen nach LMF) — genau das
// Szenario aus der Meldung. Ausleihe/Frist erkannten das Buch zwar korrekt als LMF,
// die Verlängerung ließ es aber auf der alten Frist stehen → es wurde überfällig und
// landete in der Mahnung. Der Filter muss über pkg/lmf laufen, konsistent zum Rest.
func TestGlobalExtendLMF_SchreibvariantenRobust(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	const klasse = "5a"
	sid := seedSchueler(t, pool, "S-LMF-1", "Fritz", klasse)

	alteFrist := time.Date(2025, 9, 1, 23, 59, 59, 0, time.UTC)

	// Drei LMF-Schreibvarianten (alle müssen verlängert werden) + ein Freihand-Titel
	// (darf NICHT angefasst werden).
	bindestrich := seedAusleihe(t, pool, sid, "LMF-Mathe 5", alteFrist)
	leerBindestrich := seedAusleihe(t, pool, sid, "LMF - Deutsch 5", alteFrist)
	leer := seedAusleihe(t, pool, sid, "LMF Bio 5", alteFrist)
	freihand := seedAusleihe(t, pool, sid, "Der Hobbit", alteFrist)

	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPost, "/api/ausleihen/lmf/global-extend",
		strings.NewReader(`{"klasse":"5a","neues_rueckgabe_datum":"2026-07-31"}`))
	rec := httptest.NewRecorder()
	srv.GlobalExtendLMFHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
	}

	neu := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)
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
		frist := fristVon(t, pool, tc.id)
		if tc.verlängert && !frist.Equal(neu) {
			t.Errorf("%s: Frist nicht verlängert — erwartet %s, war %s (LMF-Schreibvariante übersehen)",
				tc.name, neu, frist)
		}
		if !tc.verlängert && !frist.Equal(alteFrist) {
			t.Errorf("%s: Freihand-Buch fälschlich verlängert — war %s", tc.name, frist)
		}
	}
}

func seedSchueler(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ($1, $2, 'Test', $3, 2030) RETURNING id`,
		barcode, vorname, klasse).Scan(&id); err != nil {
		t.Fatalf("Schüler %q anlegen: %v", vorname, err)
	}
	return id
}

// seedAusleihe legt Titel + Exemplar + offene Ausleihe an und liefert die Ausleih-ID.
func seedAusleihe(t *testing.T, pool *pgxpool.Pool, schuelerID, titel string, frist time.Time) string {
	t.Helper()
	ctx := context.Background()
	tid := titelMitMeldebestand(t, pool, titel, 1)
	eid := exemplar(t, pool, tid, "BC-"+titel, true, "")
	var aid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, $3) RETURNING id`,
		eid, schuelerID, frist).Scan(&aid); err != nil {
		t.Fatalf("Ausleihe für %q anlegen: %v", titel, err)
	}
	return aid
}

func fristVon(t *testing.T, pool *pgxpool.Pool, ausleiheID string) time.Time {
	t.Helper()
	var frist time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT rueckgabe_frist FROM ausleihen WHERE id = $1`, ausleiheID).Scan(&frist); err != nil {
		t.Fatalf("Frist lesen: %v", err)
	}
	return frist.UTC()
}
