package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Befund F3, Kern des Jahreswechsel-Problems: Der Versetzungslauf zählte nur
// schueler.klasse hoch — die Klassenlehrer-Zuordnung blieb auf den alten Namen
// stehen, und die Mahnliste der neuen 6a ging an niemanden. Seit 18.08.2026
// wandert die Zuordnung in derselben Transaktion mit. Die Klassen-Bücherlisten
// bleiben BEWUSST stehen (Stufen-Semantik, siehe student_promotion.go).

func versetzungAusfuehren(t *testing.T, pool *pgxpool.Pool, dryRun bool) PromoteStudentsResponse {
	t.Helper()
	// Der Audit-Eintrag des Laufs verlangt einen echten Bearbeiter (FK auf benutzer).
	var adminID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('F3', 'Admin', 'f3-admin@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	srv := &Server{DB: &db.Database{Pool: pool}}
	body := `{"confirm": true}`
	if dryRun {
		body = `{"dry_run": true}`
	}
	req := httptest.NewRequest("POST", "/api/students/promote", strings.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
	w := httptest.NewRecorder()
	srv.PromoteStudentsHandler()(w, req)
	if w.Code != 200 {
		t.Fatalf("Versetzung: HTTP %d: %s", w.Code, w.Body.String())
	}
	var resp PromoteStudentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}
	return resp
}

func TestVersetzungNimmtKlassenlehrerMit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)
	if _, err := pool.Exec(ctx, `DELETE FROM klassen_lehrer_mapping`); err != nil {
		t.Fatalf("Mapping leeren: %v", err)
	}

	// Ein Schüler, damit der Lauf etwas zu versetzen hat.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('F3-S-1', 'Vera', 'Versetzt', '05A', 2033)`); err != nil {
		t.Fatalf("Schüler: %v", err)
	}
	// Kette 7a→8a→9a beweist die absteigende Reihenfolge; 9h ist Abschluss.
	for klasse, mail := range map[string]string{
		"05A": "a@schule.example", "07A": "b@schule.example",
		"08A": "c@schule.example", "09H": "d@schule.example",
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ($1, $2)`, klasse, mail); err != nil {
			t.Fatalf("Mapping %s: %v", klasse, err)
		}
	}

	// Erst der Dry-Run: exakte Vorschau, aber NICHTS geschrieben.
	vorschau := versetzungAusfuehren(t, pool, true)
	if vorschau.MappingVersetzt != 3 || vorschau.MappingEntfernt != 1 {
		t.Fatalf("Vorschau falsch: %+v", vorschau)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM klassen_lehrer_mapping WHERE klasse = '05A'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("Dry-Run hat geschrieben (5a weg, n=%d)", n)
	}

	echt := versetzungAusfuehren(t, pool, false)
	if echt.MappingVersetzt != 3 || echt.MappingEntfernt != 1 || len(echt.MappingKonflikte) != 0 {
		t.Fatalf("Lauf falsch: %+v", echt)
	}

	erwartet := map[string]string{
		"06A": "a@schule.example", "08A": "b@schule.example", "09A": "c@schule.example",
	}
	for klasse, mail := range erwartet {
		var got string
		if err := pool.QueryRow(ctx, `SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = $1`, klasse).Scan(&got); err != nil || got != mail {
			t.Errorf("Zuordnung %s: erwartet %s, got %q (err=%v)", klasse, mail, got, err)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM klassen_lehrer_mapping`).Scan(&n); err != nil || n != 3 {
		t.Errorf("Abschlussklasse 9h muss entfernt sein, Zeilen=%d", n)
	}
	// Und der Schüler trägt den Namen, auf den die Zuordnung jetzt zeigt.
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE barcode_id = 'F3-S-1'`).Scan(&klasse); err != nil || klasse != "06A" {
		t.Errorf("Schüler-Klasse: erwartet 6a, got %q", klasse)
	}
}

// TestVersetzungNamenskonflikteStrukturellEliminiert: Der klassische Konfliktfall —
// '09A' UND '09A' existieren nebeneinander und laufen beide auf '10A' — ist seit dem
// Klassen-Vokabular (Migration 079) UNMÖGLICH: Der Kanonisierungs-Trigger zieht '09A'
// beim Schreiben auf die registrierte Form '09A', und der Primärschlüssel weist die
// Dublette ab. Der Konflikt-Zweig in versetzeKlassenlehrerZuordnung bleibt als
// Rückfallebene bestehen; der Lauf selbst ist konfliktfrei.
func TestVersetzungNamenskonflikteStrukturellEliminiert(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)
	if _, err := pool.Exec(ctx, `DELETE FROM klassen_lehrer_mapping`); err != nil {
		t.Fatalf("Mapping leeren: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('09A', 'g@schule.example')`); err != nil {
		t.Fatalf("Mapping 9a: %v", err)
	}
	// Die Schreibvariante wird kanonisiert und prallt am Primärschlüssel ab — genau
	// die Vorbedingung des alten Konfliktfalls kann nicht mehr entstehen.
	if _, err := pool.Exec(ctx, `
		INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('9a', 'h@schule.example')`); err == nil {
		t.Fatal("'9a' neben '09A' darf nicht mehr existieren (Kanonisierung + PK), wurde aber angelegt")
	}

	resp := versetzungAusfuehren(t, pool, false)
	if resp.MappingVersetzt != 1 || len(resp.MappingKonflikte) != 0 {
		t.Fatalf("Lauf falsch: %+v", resp)
	}
	var mail string
	if err := pool.QueryRow(ctx, `SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = '10A'`).Scan(&mail); err != nil || mail != "g@schule.example" {
		t.Errorf("'09A' muss zu '10A' geworden sein, got %q", mail)
	}
}

// TestVersetzungDoppellaufSerialisiert belegt den Advisory-Lock (Nebenläufigkeits-
// Audit 19.08.2026): Zwei GLEICHZEITIGE Versetzungsläufe dürfen die Schule nicht +2
// befördern. Der reine COUNT-Check auf audit_logs ist TOCTOU; der pg_advisory_xact_lock
// serialisiert hart — genau einer gewinnt (200), der andere sieht danach den
// Audit-Eintrag und bekommt 409. Netto: jede Klasse steigt um GENAU eine Stufe.
func TestVersetzungDoppellaufSerialisiert(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('DL', 'Admin', 'dl-admin@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	// Ein Schüler in 5a — die Kohorte, die genau einmal auf 6a steigen darf.
	if _, err := pool.Exec(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum)
		VALUES ('Doppel','Lauf','05A','S-DL-1',2032,'2015-01-01')`); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM schueler WHERE barcode_id='S-DL-1'`); err != nil {
			t.Logf("Aufräumen Schüler: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM audit_logs WHERE aktion='SCHULJAHRESWECHSEL' AND admin_id=$1`, adminID); err != nil {
			t.Logf("Aufräumen Audit: %v", err)
		}
	})

	lauf := func() int {
		srv := &Server{DB: &db.Database{Pool: pool}}
		req := httptest.NewRequest("POST", "/api/students/promote", strings.NewReader(`{"confirm": true}`))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
		w := httptest.NewRecorder()
		srv.PromoteStudentsHandler()(w, req)
		return w.Code
	}

	codes := make(chan int, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			codes <- lauf()
		}()
	}
	close(start)
	c1, c2 := <-codes, <-codes

	// Genau ein 200 und ein 409 (Reihenfolge egal).
	ok, konflikt := 0, 0
	for _, c := range []int{c1, c2} {
		switch c {
		case 200:
			ok++
		case 409:
			konflikt++
		default:
			t.Fatalf("unerwarteter Status %d", c)
		}
	}
	if ok != 1 || konflikt != 1 {
		t.Fatalf("erwartet genau 1×200 + 1×409, war %d×200 / %d×409", ok, konflikt)
	}

	// Der Schüler steht in 6a — NICHT 7a.
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE barcode_id='S-DL-1'`).Scan(&klasse); err != nil {
		t.Fatalf("Klasse lesen: %v", err)
	}
	if klasse != "06A" {
		t.Errorf("Schule wurde doppelt versetzt: Klasse %q statt 6a", klasse)
	}
}
