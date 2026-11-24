package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Am Tag der Bücherausgabe zeigt der Monitor die Zeilen des veröffentlichten Plans für diesen
// Tag (docs/FACHKONZEPT.md §16.2). Gefragt wird die echte Tür mit gestellter Uhr; der Tag ist
// der Tag der Schule, auch kurz nach Mitternacht, wenn UTC noch beim Vortag ist.
func TestMonitorFolien_LMFHeute(t *testing.T) {
	pool := pgtest.Pool(t)
	ausgabe := legeLMFPlanAn(t, pool, "ausgabe", "2099-08-01", true)
	legeLMFTerminAn(t, pool, ausgabe, 1, "2099-08-18", 1, "", "05A", "05B")
	legeLMFTerminAn(t, pool, ausgabe, 2, "2099-08-18", 2, "Nachzügler")
	legeLMFTerminAn(t, pool, ausgabe, 3, "2099-08-19", 1, "", "07A")
	entwurf := legeLMFPlanAn(t, pool, "rueckgabe", "2098-08-01", false)
	legeLMFTerminAn(t, pool, entwurf, 1, "2099-06-24", 1, "", "06A")

	heute := &repository.MonitorLMFTag{Art: "ausgabe", Datum: "2099-08-18", Termine: []repository.MonitorLMFTermin{
		{Stunde: 1, Klassen: []string{"05A", "05B"}, Vermerk: ""},
		{Stunde: 2, Klassen: []string{}, Vermerk: "Nachzügler"},
	}}
	faelle := []struct {
		name     string
		jetzt    time.Time
		erwartet *repository.MonitorLMFTag
	}{
		{"am Tag die Zeilen dieses Tages", time.Date(2099, 8, 18, 10, 0, 0, 0, schulzeit.Zone()), heute},
		{"kurz nach Mitternacht schon der neue Tag", time.Date(2099, 8, 18, 0, 30, 0, 0, schulzeit.Zone()), heute},
		{"am Tag davor keine Folie", time.Date(2099, 8, 17, 23, 30, 0, 0, schulzeit.Zone()), nil},
		{"ein Entwurf bringt keine Folie", time.Date(2099, 6, 24, 10, 0, 0, 0, schulzeit.Zone()), nil},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Uhr: func() time.Time { return f.jetzt }}}
			rec := httptest.NewRecorder()
			srv.MonitorFolien()(rec, httptest.NewRequest(http.MethodGet, "/api/monitor/slides", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET /api/monitor/slides = %d: %s", rec.Code, rec.Body.String())
			}
			var antwort repository.MonitorFolien
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v", err)
			}
			if diff := cmp.Diff(f.erwartet, antwort.LMFHeute); diff != "" {
				t.Errorf("lmf_heute (-erwartet +bekommen):\n%s", diff)
			}
		})
	}
}

// legeLMFPlanAn legt einen Plan der Art an, veröffentlicht oder als Entwurf. Der Rahmen ist
// beliebig: Die Zeilen tragen Datum und Stunde selbst.
func legeLMFPlanAn(t *testing.T, pool *pgxpool.Pool, art, schuljahrBeginn string, veroeffentlicht bool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO lmf_plaene (art, schuljahr_beginn, erster_tag, veroeffentlicht_am, letzter_tag, letzte_stunde)
		VALUES ($1, $2::date, $2::date,
		        CASE WHEN $3 THEN CURRENT_TIMESTAMP END,
		        CASE WHEN $1 = 'rueckgabe' THEN $2::date + 300 END,
		        CASE WHEN $1 = 'rueckgabe' THEN 4 END)
		RETURNING id`, art, schuljahrBeginn, veroeffentlicht).Scan(&id); err != nil {
		t.Fatalf("Plan %s anlegen: %v", art, err)
	}
	return id
}

// legeLMFTerminAn legt eine Zeile des Plans an, mit oder ohne Klassen.
func legeLMFTerminAn(t *testing.T, pool *pgxpool.Pool, planID string, position int, datum string, stunde int, vermerk string, klassen ...string) {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO lmf_termine (plan_id, position, datum, stunde, art, vermerk)
		SELECT $1, $2, $3::date, $4, art, $5 FROM lmf_plaene WHERE id = $1
		RETURNING id`, planID, position, datum, stunde, vermerk).Scan(&id); err != nil {
		t.Fatalf("Zeile %d anlegen: %v", position, err)
	}
	for _, k := range klassen {
		if _, err := pool.Exec(ctx, `INSERT INTO lmf_termin_klassen (termin_id, klasse) VALUES ($1, $2)`, id, k); err != nil {
			t.Fatalf("Klasse %s anlegen: %v", k, err)
		}
	}
}
