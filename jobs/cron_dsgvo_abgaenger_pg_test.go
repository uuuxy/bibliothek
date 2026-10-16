package jobs

import (
	"context"
	"os"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die endgültige Löschung fälliger Abgänger (FuehreAbgaengerLoeschungAus →
// LoescheAbgaengerEndgueltig → tilgeSpurenUndLoescheLeser) an Postgres, nicht nur mit pgxmock:
// Ein Fehler an Schema, Constraint oder Fremdschlüssel (etwa ON DELETE RESTRICT an einer
// anderen Tabelle) scheiterte sonst stumm und stünde nur im Log, die Löschung fände nie statt.
// Der Test fährt die ganze Kette mit allen verknüpften Zeilen und prüft, dass der Schüler wirklich
// weg ist und die übrigen Tabellen richtig behandelt wurden.
func TestAbgaengerLoeschung_HarteLoeschungGegenEchtesPostgres(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
	}

	_, dsn := legeProbeDatenbankAn(t, adminDSN, "gdprdel")
	befuelleQuelle(t, dsn)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Bearbeiter (Ziel der Fremdschlüssel von Ausleihe und Protokoll), Titel und Exemplar.
	var bearbeiterID, titelID, exemplarID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Del', 'Kraft', 'del@example.org', 'admin', true) RETURNING id`).Scan(&bearbeiterID); err != nil {
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

	// Fälliger Abgänger: ist_abgaenger, altes Abgangsjahr, nicht im Papierkorb und schon
	// anonymisiert, denn der Lauf löscht nur, was die Karenz durchlaufen hat
	// (cron_dsgvo_loeschung_karenz_pg_test.go). Hier geht es um die Mechanik der endgültigen
	// Löschung selbst (Ausleihen anonymisiert, Zeile weg), nicht um die Fälligkeit.
	var sid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger, anonymized_at)
		VALUES ('DEL-S-1', 'Alt', 'Abgang', 'ABG', 2020, true, NOW() - interval '1 day') RETURNING id`).Scan(&sid); err != nil {
		t.Fatalf("Schüler: %v", err)
	}

	// Verknüpfte Zeilen, die die Kette der Löschung anfassen muss: eine zurückgegebene Ausleihe
	// (schueler_id → NULL), ein bezahlter Schadensfall (gelöscht; unbezahlte sperrten die Löschung),
	// ein Eintrag im Protokoll (anonymisiert), eine Vormerkung und ein Foto (beide über ON DELETE
	// CASCADE). Nur so schlägt ein Fremdschlüssel mit RESTRICT hier wirklich an.
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

	z := NewZeitplan(pool, repository.NewProtokollAbfragen(pool))
	z.FuehreAbgaengerLoeschungAus(ctx)

	zaehle := func(abfrage string, argumente ...any) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, abfrage, argumente...).Scan(&n); err != nil {
			t.Fatalf("Zählen (%s): %v", abfrage, err)
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
	// Der Eintrag im Protokoll darf keinen Klarnamen mehr führen.
	if n := zaehle(`SELECT count(*) FROM audit_log WHERE datensatz_id = $1 AND details->>'nachname' IS NOT NULL`, sid); n != 0 {
		t.Errorf("Klarname überlebt im Audit-Log nach der Löschung: %d", n)
	}
}

// FuehreBearbeiterAnonymisierungAus an Postgres: Sie leert die Kennungen der Bearbeiter an
// abgeschlossenen Ausleihen, die älter als 14 Tage sind (Datensparsamkeit für die Person an der
// Theke). Ein Fehler an Schema oder Constraint bliebe mit einer Attrappe auch hier unbemerkt.
func TestBearbeiterAnonymisierung_GegenEchtesPostgres(t *testing.T) {
	adminDSN := os.Getenv(datenbankVariable)
	if adminDSN == "" {
		t.Skipf("%s nicht gesetzt — Test übersprungen", datenbankVariable)
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
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Anon', 'Kraft', 'anl@example.org', 'admin', true) RETURNING id`).Scan(&bearbeiterID); err != nil {
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
	var ausleiheID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, rueckgabe_bearbeiter_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, $3, $3, now() - interval '40 days', now() - interval '30 days', now() - interval '20 days') RETURNING id`,
		exemplarID, sid, bearbeiterID).Scan(&ausleiheID); err != nil {
		t.Fatalf("Ausleihe: %v", err)
	}

	// Die Grenzen der Bedingung, je eine Ausleihe: Nur einer der zwei Bearbeiter steht da; die
	// Rückgabe liegt noch keine 14 Tage zurück; die Ausleihe läuft noch.
	leihe := func(was, barcode, ausgabe, rueckgabe, werte string) string {
		t.Helper()
		var exemplar, id string
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titelID, barcode).Scan(&exemplar); err != nil {
			t.Fatalf("Exemplar %s: %v", barcode, err)
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, rueckgabe_bearbeiter_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, $2, `+ausgabe+`, `+rueckgabe+`, `+werte+`) RETURNING id`,
			exemplar, sid, bearbeiterID).Scan(&id); err != nil {
			t.Fatalf("%s: %v", was, err)
		}
		return id
	}
	zurueckVor20Tagen := `now() - interval '40 days', now() - interval '30 days', now() - interval '20 days'`
	nurAusgabe := leihe("nur Bearbeiter der Ausgabe", "ANL-EX2", "$3", "NULL", zurueckVor20Tagen)
	nurRueckgabe := leihe("nur Bearbeiter der Rückgabe", "ANL-EX3", "NULL", "$3", zurueckVor20Tagen)
	frisch := leihe("vor zehn Tagen zurück", "ANL-EX4", "$3", "$3",
		`now() - interval '40 days', now() - interval '30 days', now() - interval '10 days'`)
	laufend := leihe("läuft noch", "ANL-EX5", "$3", "NULL::uuid",
		`now() - interval '60 days', now() + interval '10 days', NULL`)

	z := NewZeitplan(pool, repository.NewProtokollAbfragen(pool))
	z.FuehreBearbeiterAnonymisierungAus(ctx)

	ohneBearbeiter := func(id string) bool {
		t.Helper()
		var beide bool
		if err := pool.QueryRow(ctx,
			`SELECT bearbeiter_id IS NULL AND rueckgabe_bearbeiter_id IS NULL FROM ausleihen WHERE id = $1`, id).Scan(&beide); err != nil {
			t.Fatalf("Ausleihe lesen: %v", err)
		}
		return beide
	}
	if !ohneBearbeiter(ausleiheID) {
		t.Error("Operator-IDs der alten Ausleihe wurden NICHT anonymisiert")
	}
	if !ohneBearbeiter(nurAusgabe) {
		t.Error("Der Bearbeiter der Ausgabe steht noch, obwohl die Rückgabe 20 Tage zurückliegt")
	}
	if !ohneBearbeiter(nurRueckgabe) {
		t.Error("Der Bearbeiter der Rückgabe steht noch, obwohl die Rückgabe 20 Tage zurückliegt")
	}
	if ohneBearbeiter(frisch) {
		t.Error("Eine vor zehn Tagen zurückgegebene Ausleihe hat ihre Bearbeiter schon verloren")
	}
	if ohneBearbeiter(laufend) {
		t.Error("Eine laufende Ausleihe hat ihren Bearbeiter verloren")
	}
}
