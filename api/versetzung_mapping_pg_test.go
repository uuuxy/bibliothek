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
		VALUES ('F3-S-1', 'Vera', 'Versetzt', '05a', 2033)`); err != nil {
		t.Fatalf("Schüler: %v", err)
	}
	// Kette 07a→08a→09a beweist die absteigende Reihenfolge; 09h ist Abschluss.
	for klasse, mail := range map[string]string{
		"05a": "a@schule.example", "07a": "b@schule.example",
		"08a": "c@schule.example", "09h": "d@schule.example",
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
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM klassen_lehrer_mapping WHERE klasse = '05a'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("Dry-Run hat geschrieben (05a weg, n=%d)", n)
	}

	echt := versetzungAusfuehren(t, pool, false)
	if echt.MappingVersetzt != 3 || echt.MappingEntfernt != 1 || len(echt.MappingKonflikte) != 0 {
		t.Fatalf("Lauf falsch: %+v", echt)
	}

	erwartet := map[string]string{
		"06a": "a@schule.example", "08a": "b@schule.example", "09a": "c@schule.example",
	}
	for klasse, mail := range erwartet {
		var got string
		if err := pool.QueryRow(ctx, `SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = $1`, klasse).Scan(&got); err != nil || got != mail {
			t.Errorf("Zuordnung %s: erwartet %s, got %q (err=%v)", klasse, mail, got, err)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM klassen_lehrer_mapping`).Scan(&n); err != nil || n != 3 {
		t.Errorf("Abschlussklasse 09h muss entfernt sein, Zeilen=%d", n)
	}
	// Und der Schüler trägt den Namen, auf den die Zuordnung jetzt zeigt.
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM schueler WHERE barcode_id = 'F3-S-1'`).Scan(&klasse); err != nil || klasse != "06a" {
		t.Errorf("Schüler-Klasse: erwartet 06a, got %q", klasse)
	}
}

func TestVersetzungMeldetNamenskonflikte(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	resetBestandsdaten(t, pool)
	if _, err := pool.Exec(ctx, `DELETE FROM klassen_lehrer_mapping`); err != nil {
		t.Fatalf("Mapping leeren: %v", err)
	}
	// '9a' und '09a' laufen beide auf '10a' — die zweite bleibt stehen und wird gemeldet.
	for klasse, mail := range map[string]string{"9a": "g@schule.example", "09a": "h@schule.example"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ($1, $2)`, klasse, mail); err != nil {
			t.Fatalf("Mapping %s: %v", klasse, err)
		}
	}

	resp := versetzungAusfuehren(t, pool, false)
	if resp.MappingVersetzt != 1 || len(resp.MappingKonflikte) != 1 {
		t.Fatalf("Konfliktlauf falsch: %+v", resp)
	}
	var mail string
	if err := pool.QueryRow(ctx, `SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = '10a'`).Scan(&mail); err != nil || mail != "g@schule.example" {
		t.Errorf("'9a' muss zu '10a' geworden sein, got %q", mail)
	}
	if err := pool.QueryRow(ctx, `SELECT lehrer_email FROM klassen_lehrer_mapping WHERE klasse = '09a'`).Scan(&mail); err != nil || mail != "h@schule.example" {
		t.Errorf("'09a' muss als Konflikt stehen bleiben, got %q", mail)
	}
}
