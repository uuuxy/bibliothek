package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func mahnstufeVon(t *testing.T, pool *pgxpool.Pool, ausleiheID string) int {
	t.Helper()
	var m int
	if err := pool.QueryRow(context.Background(),
		`SELECT mahnstufe FROM ausleihen WHERE id = $1`, ausleiheID).Scan(&m); err != nil {
		t.Fatalf("mahnstufe lesen: %v", err)
	}
	return m
}

// markiereGemahntUndUeberfaellig setzt eine Ausleihe auf Mahnstufe `stufe` und eine bereits
// abgelaufene Frist — der Ausgangszustand für die Eskalations-Tests.
func markiereGemahntUndUeberfaellig(t *testing.T, pool *pgxpool.Pool, ausleiheID string, stufe int, frist time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE ausleihen SET mahnstufe = $2, letztes_mahndatum = now(), rueckgabe_frist = $3 WHERE id = $1`,
		ausleiheID, stufe, frist); err != nil {
		t.Fatalf("mahnstufe/frist setzen: %v", err)
	}
}

// TestGlobalExtendLMF_ResetMahnstufe sichert Audit-Bug #1 ab: Schiebt eine Verlängerung die
// Frist wieder in die Zukunft, wird die Mahn-Eskalation zurückgesetzt (mahnstufe = 0) — sonst
// übersprang ein nach der 1. Mahnung verlängertes und erneut überzogenes Buch die 1. Stufe und
// eskalierte sofort zur Rechnung. Bleibt die Frist trotz "Verlängerung" in der Vergangenheit,
// bleibt die Mahnstufe erhalten (Eskalation berechtigt).
func TestGlobalExtendLMF_ResetMahnstufe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	vergangenheit := time.Now().AddDate(0, 0, -60)

	// Klasse 9z: wird in die Zukunft verlängert -> Reset erwartet.
	sidZ := seedSchueler(t, pool, "S-MST-Z", "Mona", "9z")
	loanReset := seedAusleihe(t, pool, sidZ, "LMF-Mathe 9", vergangenheit)
	markiereGemahntUndUeberfaellig(t, pool, loanReset, 2, vergangenheit)

	// Klasse 9y: "Verlängerung" auf ein vergangenes Datum -> Mahnstufe bleibt.
	sidY := seedSchueler(t, pool, "S-MST-Y", "Bodo", "9y")
	loanKeep := seedAusleihe(t, pool, sidY, "LMF-Deutsch 9", vergangenheit)
	markiereGemahntUndUeberfaellig(t, pool, loanKeep, 2, vergangenheit)

	srv := &Server{DB: &db.Database{Pool: pool}}

	// (a) Zukunft -> Reset.
	zukunft := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if rec := globalExtend(t, srv, "9z", zukunft); rec.Code != http.StatusOK {
		t.Fatalf("Extend Zukunft: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if m := mahnstufeVon(t, pool, loanReset); m != 0 {
		t.Errorf("Verlängerung in die Zukunft: Mahnstufe erwartet 0, war %d (Eskalations-Falle)", m)
	}

	// (b) Vergangenheit -> Mahnstufe bleibt erhalten.
	vergangen := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	if rec := globalExtend(t, srv, "9y", vergangen); rec.Code != http.StatusOK {
		t.Fatalf("Extend Vergangenheit: erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if m := mahnstufeVon(t, pool, loanKeep); m != 2 {
		t.Errorf("weiterhin überfällig: Mahnstufe erwartet 2 (unverändert), war %d", m)
	}
}

func globalExtend(t *testing.T, srv *Server, klasse, datum string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"klasse":%q,"neues_rueckgabe_datum":%q}`, klasse, datum)
	req := httptest.NewRequest(http.MethodPost, "/api/ausleihen/lmf/global-extend", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.GlobalExtendLMFHandler()(rec, req)
	return rec
}

// TestLaufzettel_NurAbgaengerMitBuechern sichert Audit-Bug #2 ab: Der Laufzettel-Massendruck
// darf NUR Abgänger mit noch offenen Büchern enthalten. Vorher (LEFT JOIN) erschien jeder
// Abgänger — 150 Abgänger, davon 140 ohne Buch, ergaben 140 leere Laufzettel.
func TestLaufzettel_NurAbgaengerMitBuechern(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Abgänger MIT offenem Buch.
	mitBuch := seedAbgaengerRet(t, pool, "AB-1", "Anna", "10a")
	seedAusleihe(t, pool, mitBuch, "Physikbuch 10", time.Now().AddDate(0, 0, -3))
	// Abgänger OHNE Buch — darf keinen (leeren) Laufzettel bekommen.
	seedAbgaenger(t, pool, "AB-2", "Bodo", "10a", true)
	// Nicht-Abgänger mit Buch — darf nicht auftauchen.
	kein := seedSchueler(t, pool, "AB-3", "Cleo", "8a")
	seedAusleihe(t, pool, kein, "Chemiebuch 8", time.Now().AddDate(0, 0, -3))

	srv := &Server{DB: &db.Database{Pool: pool}}
	students, err := srv.queryLaufzettelStudents(ctx)
	if err != nil {
		t.Fatalf("queryLaufzettelStudents: %v", err)
	}

	if len(students) != 1 {
		t.Fatalf("erwartet genau 1 Laufzettel (nur Abgänger mit Buch), waren %d", len(students))
	}
	if students[0].Vorname != "Anna" || len(students[0].Ausleihen) != 1 {
		t.Errorf("falscher/leerer Laufzettel: %+v", students[0])
	}
}

func seedAbgaengerRet(t *testing.T, pool *pgxpool.Pool, barcode, vorname, klasse string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
		 VALUES ($1, $2, 'Test', $3, 2030, true) RETURNING id`,
		barcode, vorname, klasse).Scan(&id); err != nil {
		t.Fatalf("Abgänger %q anlegen: %v", vorname, err)
	}
	return id
}

// TestKlassensatzReservierung_Bestandsdeckelung sichert Audit-Bug #3 ab: Es dürfen nicht mehr
// Exemplare reserviert werden, als die Bibliothek physisch besitzt — sonst bleibt die
// Reservierung als unlösbare Aufgabe im Dashboard hängen.
func TestKlassensatzReservierung_Bestandsdeckelung(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	tid := titelMitMeldebestand(t, pool, "Klassenlektuere", 0)
	for i := 0; i < 5; i++ {
		exemplar(t, pool, tid, fmt.Sprintf("BC-KL-%d", i), true, "")
	}
	var uid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		 VALUES ('RES-B', 'Res', 'Kraft', 'res@example.org', 'mitarbeiter', true) RETURNING id`).Scan(&uid); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}

	// 150 Exemplare bei 5 im Bestand -> 400 (Überbuchung abgewiesen).
	if rec := reservierung(t, srv, uid, tid, 150); rec.Code != http.StatusBadRequest {
		t.Fatalf("Überbuchung (150 von 5): erwartet 400, war %d: %s", rec.Code, rec.Body.String())
	}
	// 5 Exemplare -> 201 (erfüllbar).
	if rec := reservierung(t, srv, uid, tid, 5); rec.Code != http.StatusCreated {
		t.Fatalf("gültige Reservierung (5 von 5): erwartet 201, war %d: %s", rec.Code, rec.Body.String())
	}
}

func reservierung(t *testing.T, srv *Server, userID, titelID string, anzahl int) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"titel_id":%q,"klasse":"5a","anzahl":%d}`, titelID, anzahl)
	req := httptest.NewRequest(http.MethodPost, "/api/reservierungen/klassensatz", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: userID, Rolle: auth.RoleMitarbeiter}))
	rec := httptest.NewRecorder()
	srv.CreateKlassensatzReservierungHandler()(rec, req)
	return rec
}

// TestShelfWarmers_NeuzugangKeinLadenhueter sichert Audit-Bug #4 ab: Ein frisch gekaufter,
// noch nie ausgeliehener Titel darf NICHT sofort als Ladenhüter/Aussonderungskandidat gelten.
// Nur ein alter (>2 Jahre im Bestand), nie ausgeliehener Titel ist ein echter Ladenhüter.
func TestShelfWarmers_NeuzugangKeinLadenhueter(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	altTid := titelMitMeldebestand(t, pool, "Uralt nie ausgeliehen", 0)
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, erstellt_am)
		 VALUES ($1, 'BC-OLD', true, now() - INTERVAL '3 years')`, altTid); err != nil {
		t.Fatalf("altes Exemplar: %v", err)
	}
	neuTid := titelMitMeldebestand(t, pool, "Frischer Neuzugang", 0)
	if _, err := pool.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, erstellt_am)
		 VALUES ($1, 'BC-NEW', true, now())`, neuTid); err != nil {
		t.Fatalf("neues Exemplar: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	warmers := srv.queryShelfWarmers(ctx, "", 100)

	titel := map[string]bool{}
	for _, w := range warmers {
		titel[w.Titel] = true
	}
	if !titel["Uralt nie ausgeliehen"] {
		t.Error("alter, nie ausgeliehener Titel fehlt in der Ladenhüter-Liste")
	}
	if titel["Frischer Neuzugang"] {
		t.Error("frischer Neuzugang landet fälschlich auf der Ladenhüter/Aussonderungs-Liste")
	}
}

// TestMahnwesen_KeineZombieMahnungFuerGeloeschte sichert Audit-Bug #5 ab: Für einen weich
// (Papierkorb) gelöschten Schüler dürfen keine Mahn-Daten mehr verarbeitet werden — sonst
// gingen "Zombie-Mahnungen" an den ehemaligen Klassenlehrer (DSGVO-Verstoß).
func TestMahnwesen_KeineZombieMahnungFuerGeloeschte(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	pastFrist := time.Now().AddDate(0, 0, -10)
	aktiv := seedSchueler(t, pool, "MZ-1", "Nils", "7a")
	seedAusleihe(t, pool, aktiv, "Mahnbuch A", pastFrist)
	geloescht := seedSchueler(t, pool, "MZ-2", "Greta", "7a")
	seedAusleihe(t, pool, geloescht, "Mahnbuch B", pastFrist)
	if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = now() WHERE id = $1`, geloescht); err != nil {
		t.Fatalf("Schüler löschen: %v", err)
	}

	repo := repository.NewMahnwesenRepository(pool)
	klassen, err := repo.QueryUeberfaelligeNachKlasse(ctx, "")
	if err != nil {
		t.Fatalf("QueryUeberfaelligeNachKlasse: %v", err)
	}

	ids := map[string]bool{}
	for _, k := range klassen {
		for _, sch := range k.Schueler {
			ids[sch.SchuelerID] = true
		}
	}
	if !ids[aktiv] {
		t.Error("aktiver überfälliger Schüler fehlt in der Mahnliste")
	}
	if ids[geloescht] {
		t.Error("gelöschter Schüler erscheint in der Mahnliste (Zombie-Mahnung / DSGVO-Verstoß)")
	}
}
