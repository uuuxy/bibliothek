package jobs

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"bibliothek/internal/backupkrypto"
)

// Die echte Restore-Probe: pg_dump → gzip → AES-256-GCM → entschlüsseln → psql →
// Datenabgleich, gegen zwei echte Datenbanken.
//
// Warum das nötig war: internal/backupkrypto/krypto_test.go prüft die Verschlüsselung gegen einen
// ERFUNDENEN SQL-String ("-- pg_dump\nCREATE TABLE schueler …"). Damit ist bewiesen, dass
// AES-GCM funktioniert — nicht, dass sich aus einem echten Backup jemals wieder eine
// Datenbank herstellen lässt. Zwischen beidem liegt alles, was im Ernstfall schiefgeht:
// der pg_dump-Aufruf samt .pgpass, die Gzip-Pipeline mit ihren zwei Goroutinen, und die
// Frage, ob psql den Dump überhaupt fehlerfrei einspielt.
//
// Der Ernstfall ist der schlechteste Zeitpunkt, das zum ersten Mal zu erfahren.
//
// Voraussetzungen: TEST_DATABASE_URL sowie pg_dump und psql im PATH. Fehlt eines,
// wird übersprungen — mit Angabe, was fehlt, damit ein stiller Skip nicht als
// bestandene Probe durchgeht.

const drillEnvVar = "TEST_DATABASE_URL"

// TestBackupRestoreDrill stellt den kompletten Wiederherstellungsweg nach.
func TestBackupRestoreDrill(t *testing.T) {
	adminDSN := pruefeVoraussetzungen(t)

	quelle, quellDSN := legeProbeDatenbankAn(t, adminDSN, "src")
	ziel, zielDSN := legeProbeDatenbankAn(t, adminDSN, "dst")
	t.Logf("Probe-Datenbanken: Quelle=%s Ziel=%s", quelle, ziel)

	befuelleQuelle(t, quellDSN)
	erwartet := zaehleAlleTabellen(t, quellDSN)
	if len(erwartet) < 10 {
		t.Fatalf("Quelle sieht nicht nach dem echten Schema aus (nur %d Tabellen)", len(erwartet))
	}

	// ── Schritt 1: das echte Backup erzeugen ──────────────────────────────────
	backupDir := t.TempDir()
	t.Setenv("BACKUP_ENCRYPTION_KEY", "probe-passphrase-mit-mehr-als-32-zeichen")
	t.Setenv("DATABASE_URL", quellDSN)
	t.Setenv("BACKUP_DIR", backupDir)
	t.Setenv("S3_ENDPOINT", "") // kein Offsite-Upload in der Probe

	(&BackupJob{}).RunDatabaseBackup()

	treffer, err := filepath.Glob(filepath.Join(backupDir, "backup_*.sql.gz.enc"))
	if err != nil || len(treffer) != 1 {
		t.Fatalf("genau eine Backup-Datei erwartet, gefunden: %v (err=%v)", treffer, err)
	}
	rohBackup, err := os.ReadFile(treffer[0]) // #nosec G304 - Pfad aus t.TempDir()
	if err != nil {
		t.Fatalf("Backup-Datei nicht lesbar: %v", err)
	}
	if len(rohBackup) < 512 {
		t.Fatalf("Backup ist nur %d Bytes groß — das kann kein vollständiger Dump sein", len(rohBackup))
	}

	// ── Schritt 2: entschlüsseln und auspacken (der Weg von cmd/restore-backup) ─
	sqlText := entschluesseleUndPacke(t, os.Getenv("BACKUP_ENCRYPTION_KEY"), rohBackup)
	if !strings.Contains(sqlText, "CREATE TABLE") {
		t.Fatalf("wiederhergestelltes SQL enthält kein CREATE TABLE — Dump unbrauchbar")
	}

	sqlPfad := filepath.Join(backupDir, "restore.sql")
	if err := os.WriteFile(sqlPfad, []byte(sqlText), 0o600); err != nil {
		t.Fatalf("SQL konnte nicht abgelegt werden: %v", err)
	}

	// ── Schritt 3: in die leere Zieldatenbank einspielen ──────────────────────
	// ON_ERROR_STOP=1 ist hier nicht Kosmetik: ohne den Schalter arbeitet psql nach
	// einem fehlgeschlagenen Statement einfach weiter und endet mit Rückgabewert 0.
	// Eine Wiederherstellung, die zur Hälfte misslingt, sähe damit erfolgreich aus.
	cmd := exec.Command("psql", "--dbname="+zielDSN, "--file="+sqlPfad,
		"--quiet", "--no-psqlrc", "-v", "ON_ERROR_STOP=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("psql-Wiederherstellung fehlgeschlagen: %v\n%s", err, stderr.String())
	}

	// ── Schritt 4: Gegenprobe an den Daten ────────────────────────────────────
	tatsaechlich := zaehleAlleTabellen(t, zielDSN)
	vergleicheBestand(t, erwartet, tatsaechlich)

	var titel, barcode string
	conn := verbinde(t, zielDSN)
	if err := conn.QueryRow(context.Background(),
		`SELECT t.titel, e.barcode_id FROM buecher_titel t
		   JOIN buecher_exemplare e ON e.titel_id = t.id
		  WHERE t.titel = 'Probe-Titel für die Restore-Probe'`).Scan(&titel, &barcode); err != nil {
		t.Fatalf("Inhalt kam nicht mit: %v", err)
	}
	if barcode != "B-DRILL-1" {
		t.Errorf("Barcode B-DRILL-1 erwartet, wiederhergestellt: %q", barcode)
	}
	t.Logf("Wiederherstellung bestätigt: %d Tabellen, Beispielsatz %q/%s", len(tatsaechlich), titel, barcode)
}

// pruefeVoraussetzungen überspringt den Test mit klarer Begründung statt still.
func pruefeVoraussetzungen(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv(drillEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — Restore-Probe übersprungen", drillEnvVar)
	}
	for _, werkzeug := range []string{"pg_dump", "psql"} {
		if _, err := exec.LookPath(werkzeug); err != nil {
			t.Skipf("%s nicht im PATH — Restore-Probe übersprungen (postgresql-client installieren)", werkzeug)
		}
	}
	return dsn
}

// legeProbeDatenbankAn erzeugt eine Wegwerf-Datenbank und räumt sie hinterher weg.
func legeProbeDatenbankAn(t *testing.T, adminDSN, rolle string) (name, dsn string) {
	t.Helper()

	conn := verbinde(t, adminDSN)
	ctx := context.Background()

	var aktuell string
	if err := conn.QueryRow(ctx, `SELECT current_database()`).Scan(&aktuell); err != nil {
		t.Fatalf("Datenbanknamen nicht lesbar: %v", err)
	}
	// Dieselbe Notbremse wie in den übrigen PG-Tests: Diese Probe legt Datenbanken an
	// und verwirft sie wieder. Zeigt TEST_DATABASE_URL auf etwas Echtes, ist Schluss.
	if !strings.Contains(strings.ToLower(aktuell), "test") {
		t.Fatalf("Sicherheitsabbruch: %q enthält nicht \"test\" — %s darf nur auf eine "+
			"Wegwerf-Datenbank zeigen", aktuell, drillEnvVar)
	}

	name = fmt.Sprintf("drill_%s_%d_test", rolle, os.Getpid())
	// Bezeichner sind hier zwingend Literale (CREATE DATABASE kennt keine Parameter);
	// der Name stammt ausschließlich aus Prozess-ID und Konstante.
	if _, err := conn.Exec(ctx, `DROP DATABASE IF EXISTS `+pgIdent(name)); err != nil {
		t.Fatalf("Aufräumen von %s fehlgeschlagen: %v", name, err)
	}
	if _, err := conn.Exec(ctx, `CREATE DATABASE `+pgIdent(name)); err != nil {
		t.Fatalf("Anlegen von %s fehlgeschlagen: %v", name, err)
	}
	t.Cleanup(func() {
		c := verbinde(t, adminDSN)
		if _, err := c.Exec(context.Background(), `DROP DATABASE IF EXISTS `+pgIdent(name)+` WITH (FORCE)`); err != nil {
			t.Logf("Probe-Datenbank %s blieb liegen: %v", name, err)
		}
	})

	return name, tauscheDatenbank(adminDSN, name)
}

// pgIdent quotet einen Bezeichner für Stellen, an denen Postgres keine Parameter erlaubt.
func pgIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// tauscheDatenbank ersetzt den Datenbanknamen im DSN.
func tauscheDatenbank(dsn, neu string) string {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return dsn
	}
	sslmode := "disable"
	if v := cfg.RuntimeParams["sslmode"]; v != "" {
		sslmode = v
	}
	auth := cfg.User
	if cfg.Password != "" {
		auth += ":" + cfg.Password
	}
	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", auth, cfg.Host, cfg.Port, neu, sslmode)
}

func verbinde(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Verbindung zu %s fehlgeschlagen: %v", dsn, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) }) //nolint:errcheck
	return conn
}

// befuelleQuelle spielt das echte schema.sql ein und legt einen wiedererkennbaren
// Datensatz an. Ein Backup gegen ein Spielzeugschema bewiese nichts.
func befuelleQuelle(t *testing.T, dsn string) {
	t.Helper()

	_, dieseDatei, _, _ := runtime.Caller(0)
	schema, err := os.ReadFile(filepath.Join(filepath.Dir(dieseDatei), "..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql nicht lesbar: %v", err)
	}

	conn := verbinde(t, dsn)
	ctx := context.Background()
	if _, err := conn.Exec(ctx, string(schema)); err != nil {
		t.Fatalf("schema.sql konnte nicht eingespielt werden: %v", err)
	}

	var titelID string
	if err := conn.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, isbn) VALUES ($1,$2) RETURNING id`,
		"Probe-Titel für die Restore-Probe", "9783161484100").Scan(&titelID); err != nil {
		t.Fatalf("Probedaten (Titel): %v", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1,$2)`,
		titelID, "B-DRILL-1"); err != nil {
		t.Fatalf("Probedaten (Exemplar): %v", err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('S-DRILL-1','Probe','Schüler','7a',2030)`); err != nil {
		t.Fatalf("Probedaten (Schüler): %v", err)
	}
}

// zaehleAlleTabellen liefert für JEDE Tabelle des Schemas die Zeilenzahl. Der Vergleich
// über alle Tabellen ist die aussagekräftige Gegenprobe: Er bemerkt auch eine Tabelle,
// an die beim Schreiben des Tests niemand gedacht hat.
func zaehleAlleTabellen(t *testing.T, dsn string) map[string]int {
	t.Helper()

	conn := verbinde(t, dsn)
	ctx := context.Background()

	rows, err := conn.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		t.Fatalf("Tabellenliste: %v", err)
	}
	var namen []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			t.Fatalf("Tabellenname: %v", err)
		}
		namen = append(namen, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("Tabellenliste: %v", err)
	}

	bestand := make(map[string]int, len(namen))
	for _, n := range namen {
		var anzahl int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM `+pgIdent(n)).Scan(&anzahl); err != nil {
			t.Fatalf("Zählung %s: %v", n, err)
		}
		bestand[n] = anzahl
	}
	return bestand
}

func vergleicheBestand(t *testing.T, erwartet, tatsaechlich map[string]int) {
	t.Helper()

	var fehlend, abweichend []string
	for name, soll := range erwartet {
		ist, da := tatsaechlich[name]
		switch {
		case !da:
			fehlend = append(fehlend, name)
		case ist != soll:
			abweichend = append(abweichend, fmt.Sprintf("%s: %d statt %d", name, ist, soll))
		}
	}
	sort.Strings(fehlend)
	sort.Strings(abweichend)

	if len(fehlend) > 0 {
		t.Errorf("nach der Wiederherstellung fehlen %d Tabellen: %s", len(fehlend), strings.Join(fehlend, ", "))
	}
	if len(abweichend) > 0 {
		t.Errorf("abweichende Zeilenzahlen: %s", strings.Join(abweichend, "; "))
	}
	if ueber := len(tatsaechlich) - len(erwartet); ueber != 0 {
		t.Errorf("Zieldatenbank hat %d Tabellen mehr als die Quelle", ueber)
	}
}

// entschluesseleUndPacke bildet exakt den Weg von cmd/restore-backup nach:
// backupkrypto.EntschluesseleBackup, dann gunzip.
func entschluesseleUndPacke(t *testing.T, key string, roh []byte) string {
	t.Helper()

	komprimiert, err := backupkrypto.EntschluesseleBackup(key, roh)
	if err != nil {
		t.Fatalf("Entschlüsselung fehlgeschlagen: %v", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(komprimiert))
	if err != nil {
		t.Fatalf("gzip-Header ungültig: %v", err)
	}
	defer func() { _ = gz.Close() }() //nolint:errcheck

	sqlText, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("Dekomprimierung fehlgeschlagen: %v", err)
	}
	return string(sqlText)
}
