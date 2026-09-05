package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestGDPRDeleteAbgaenger_HarteLoeschungGegenEchtesPostgres schließt die Gate-Lücke, die
// den GDPR-Anonymisierungs-Bug so lange verborgen hielt (Cron-Matrix-Sweep 20.08.2026):
// Die harte DSGVO-Löschung fälliger Abgänger (RunGDPRDeleteAbgaenger → PurgeAbgaenger →
// entferneSchuelerPIIUndLoesche) war NUR mit pgxmock getestet — ein Schema-/Constraint-/
// FK-Fehler (etwa ein ON DELETE RESTRICT auf einer Nebentabelle) würde still scheitern und
// nur geloggt, die Löschung fände nie statt. Dieser Test fährt die volle Kette gegen echtes
// Postgres mit allen verknüpften Zeilen und prüft, dass der Schüler wirklich weg ist und
// die Nebentabellen korrekt behandelt wurden.
func TestGDPRDeleteAbgaenger_HarteLoeschungGegenEchtesPostgres(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdprdel")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Bearbeiter (FK-Ziel für Ausleihe/Audit) + Titel + Exemplar.
	var bearbeiterID, titelID, exemplarID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('DEL-B', 'Del', 'Kraft', 'del@example.org', 'admin', true) RETURNING id`).Scan(&bearbeiterID); err != nil {
		t.Fatalf("Benutzer: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel) VALUES ('DSGVO-Buch') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'DEL-EX-1') RETURNING id`, titelID).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}

	// Fälliger Abgänger: ist_abgaenger, altes Abgangsjahr, NICHT im Papierkorb — und schon
	// anonymisiert: Seit 05.09.2026 löscht der Job nur, was die Karenz durchlaufen hat
	// (cron_dsgvo_loeschung_karenz_pg_test.go). Hier geht es um die Mechanik der harten
	// Löschung selbst (Ausleihhistorie anonymisiert, Zeile weg), nicht um die Fälligkeit.
	var sid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, anonymized_at)
		VALUES ('DEL-S-1', 'Alt', 'Abgang', 'ABG', 2020, true, NOW() - interval '1 day') RETURNING id`).Scan(&sid); err != nil {
		t.Fatalf("Schüler: %v", err)
	}

	// Verknüpfte Zeilen, die die Löschkette anfassen MUSS: eine ZURÜCKGEGEBENE Ausleihe
	// (schueler_id → NULL), ein BEZAHLTER Schadensfall (gelöscht — unbezahlte würden die
	// Löschung sperren), ein Audit-Eintrag (anonymisiert), eine Vormerkung + ein Foto
	// (beide per ON DELETE CASCADE). Nur so schlägt ein RESTRICT-FK hier wirklich an.
	if _, err := pool.Exec(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, $3, now() - interval '40 days', now() - interval '30 days', now() - interval '20 days')`,
		exemplarID, sid, bearbeiterID); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		VALUES ($1, $2, 'bezahlt, aber Freitext', 5.00, true)`, exemplarID, sid); err != nil {
		t.Fatalf("Schadensfall: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, details)
		VALUES ('schueler', 'DELETE_STUDENT', $1, $2, jsonb_build_object('nachname','Abgang'))`,
		sid, bearbeiterID); err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`, titelID, sid); err != nil {
		t.Fatalf("Vormerkung: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO schueler_fotos (schueler_id, foto_encrypted) VALUES ($1, '\x00'::bytea)`, sid); err != nil {
		t.Fatalf("Foto: %v", err)
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRDeleteAbgaenger()

	zaehle := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, query, args...).Scan(&n); err != nil {
			t.Fatalf("Zählen (%s): %v", query, err)
		}
		return n
	}

	if n := zaehle(`SELECT count(*) FROM schueler WHERE id = $1`, sid); n != 0 {
		t.Errorf("Abgänger wurde NICHT hart gelöscht (noch %d Zeile) — Löschkette still gescheitert?", n)
	}
	if n := zaehle(`SELECT count(*) FROM schadensfaelle WHERE schueler_id = $1`, sid); n != 0 {
		t.Errorf("Schadensfälle des Abgängers blieben stehen: %d", n)
	}
	if n := zaehle(`SELECT count(*) FROM vormerkungen WHERE schueler_id = $1`, sid); n != 0 {
		t.Errorf("Vormerkungen wurden nicht per CASCADE entfernt: %d", n)
	}
	if n := zaehle(`SELECT count(*) FROM schueler_fotos WHERE schueler_id = $1`, sid); n != 0 {
		t.Errorf("Foto wurde nicht per CASCADE entfernt: %d", n)
	}
	// Die Ausleihe bleibt als Bestandshistorie, aber ohne Personenbezug.
	if n := zaehle(`SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND schueler_id IS NULL`, exemplarID); n != 1 {
		t.Errorf("Ausleihe-Historie muss erhalten, aber schueler_id genullt sein (gefunden: %d)", n)
	}
	// Der ursprüngliche Audit-Eintrag darf keinen Klarnamen mehr führen.
	if n := zaehle(`SELECT count(*) FROM audit_log WHERE datensatz_id = $1 AND details->>'nachname' IS NOT NULL`, sid); n != 0 {
		t.Errorf("Klarname überlebt im Audit-Log nach der Löschung: %d", n)
	}
}

// TestGDPRAnonymizeLoans_GegenEchtesPostgres deckt den letzten nur mit pgxmock getesteten
// DSGVO-Mutator ab: RunGDPRAnonymizeLoans nullt die Bearbeiter-IDs abgeschlossener
// Ausleihen, die älter als 14 Tage sind (Datensparsamkeit der Operator-Identität). Ein
// Schema-/Constraint-Fehler wäre auch hier still geblieben.
func TestGDPRAnonymizeLoans_GegenEchtesPostgres(t *testing.T) {
	adminDSN := os.Getenv(drillEnvVar)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", drillEnvVar)
	}
	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdprloans")
	befuelleQuelle(t, dsn)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	var bearbeiterID, titelID, exemplarID, sid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('ANL-B', 'Anon', 'Kraft', 'anl@example.org', 'admin', true) RETURNING id`).Scan(&bearbeiterID); err != nil {
		t.Fatalf("Benutzer: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel) VALUES ('Anon-Buch') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ANL-EX') RETURNING id`, titelID).Scan(&exemplarID); err != nil {
		t.Fatalf("Exemplar: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('ANL-S', 'A', 'B', '7a', 2030) RETURNING id`).Scan(&sid); err != nil {
		t.Fatalf("Schüler: %v", err)
	}
	// Abgeschlossene Ausleihe, vor 20 Tagen zurückgegeben, mit gesetzten Bearbeitern.
	var loanID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, rueckgabe_bearbeiter_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, $3, $3, now() - interval '40 days', now() - interval '30 days', now() - interval '20 days') RETURNING id`,
		exemplarID, sid, bearbeiterID).Scan(&loanID); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}

	s := NewScheduler(pool, repository.NewAuditRepository(pool))
	s.RunGDPRAnonymizeLoans()

	var beide bool
	if err := pool.QueryRow(ctx,
		`SELECT bearbeiter_id IS NULL AND rueckgabe_bearbeiter_id IS NULL FROM ausleihen WHERE id = $1`, loanID).Scan(&beide); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	if !beide {
		t.Error("Operator-IDs der alten Ausleihe wurden NICHT anonymisiert")
	}
}
