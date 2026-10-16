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

// Die echte Restore-Probe: pg_dump → gzip → AES-256-GCM → entschlüsseln → psql → Abgleich der
// Daten, gegen zwei echte Datenbanken.
//
// internal/backupkrypto/krypto_test.go prüft die Verschlüsselung an einem erfundenen SQL-Text
// ("-- pg_dump\nCREATE TABLE schueler …"). Das belegt, dass AES-GCM wirkt, nicht, dass sich aus
// einer echten Sicherung je wieder eine Datenbank herstellen lässt. Dazwischen liegt alles, was
// im Ernstfall schiefgeht: der Aufruf von pg_dump mit .pgpass, die Leitung mit gzip und ihren
// zwei Goroutinen und die Frage, ob psql den Dump fehlerfrei einspielt. Der Ernstfall ist der
// schlechteste Zeitpunkt, das zum ersten Mal zu erfahren.
//
// Voraussetzungen: TEST_DATABASE_URL sowie pg_dump und psql im PATH. Fehlt eines, wird der Test
// übersprungen, mit Angabe, was fehlt, damit ein stummes Überspringen nicht als bestandene Probe
// durchgeht.

const datenbankVariable = "TEST_DATABASE_URL"

// TestBackupRestoreDrill stellt den ganzen Weg der Wiederherstellung nach.
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

	// ── Schritt 1: die echte Sicherung erzeugen ───────────────────────────────────
	verzeichnis := t.TempDir()
	t.Setenv("BACKUP_ENCRYPTION_KEY", "probe-passphrase-mit-mehr-als-32-zeichen")
	t.Setenv("DATABASE_URL", quellDSN)
	t.Setenv("BACKUP_DIR", verzeichnis)
	t.Setenv("S3_ENDPOINT", "") // kein Offsite-Upload in der Probe

	(&backupLauf{}).fuehreAus()

	rohBackup, err := os.ReadFile(einzigeSicherung(t, verzeichnis)) // #nosec G304 - Pfad aus t.TempDir()
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

	// ── Schritt 3: in die leere Zieldatenbank einspielen ──────────────────────
	spieleInZielEin(t, zielDSN, sqlText)

	// ── Schritt 4: Gegenprobe an den Daten ────────────────────────────────────
	tatsaechlich := zaehleAlleTabellen(t, zielDSN)
	vergleicheBestand(t, erwartet, tatsaechlich)

	var titel, barcode string
	verbindung := verbinde(t, zielDSN)
	if err := verbindung.QueryRow(context.Background(),
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

// spieleInZielEin spielt den Dump mit psql in die leere Zieldatenbank ein. ON_ERROR_STOP=1 ist
// Pflicht: Ohne den Schalter arbeitet psql nach einer gescheiterten Anweisung weiter und endet
// mit 0; eine Wiederherstellung, die zur Hälfte misslingt, sähe gelungen aus.
func spieleInZielEin(t *testing.T, zielDSN, sqlText string) {
	t.Helper()

	sqlPfad := filepath.Join(t.TempDir(), "restore.sql")
	if err := os.WriteFile(sqlPfad, []byte(sqlText), 0o600); err != nil {
		t.Fatalf("SQL konnte nicht abgelegt werden: %v", err)
	}
	befehl := exec.Command("psql", "--dbname="+zielDSN, "--file="+sqlPfad,
		"--quiet", "--no-psqlrc", "-v", "ON_ERROR_STOP=1")
	var stderr bytes.Buffer
	befehl.Stderr = &stderr
	if err := befehl.Run(); err != nil {
		t.Fatalf("psql-Wiederherstellung fehlgeschlagen: %v\n%s", err, stderr.String())
	}
}

// pruefeVoraussetzungen überspringt den Test mit klarer Begründung statt stumm.
func pruefeVoraussetzungen(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv(datenbankVariable)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — Restore-Probe übersprungen", datenbankVariable)
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

	verbindung := verbinde(t, adminDSN)
	ctx := context.Background()

	var aktuell string
	if err := verbindung.QueryRow(ctx, `SELECT current_database()`).Scan(&aktuell); err != nil {
		t.Fatalf("Datenbanknamen nicht lesbar: %v", err)
	}
	// Dieselbe Notbremse wie in den übrigen Tests an Postgres: Diese Probe legt Datenbanken an
	// und verwirft sie wieder. Zeigt TEST_DATABASE_URL auf etwas Echtes, ist Schluss.
	if !strings.Contains(strings.ToLower(aktuell), "test") {
		t.Fatalf("Sicherheitsabbruch: %q enthält nicht \"test\" — %s darf nur auf eine "+
			"Wegwerf-Datenbank zeigen", aktuell, datenbankVariable)
	}

	name = fmt.Sprintf("drill_%s_%d_test", rolle, os.Getpid())
	// Die Bezeichner stehen hier als Text (CREATE DATABASE kennt keine Parameter); der Name
	// stammt nur aus der Kennung des Prozesses und einer Konstante.
	if _, err := verbindung.Exec(ctx, `DROP DATABASE IF EXISTS `+pgBezeichner(name)); err != nil {
		t.Fatalf("Aufräumen von %s fehlgeschlagen: %v", name, err)
	}
	if _, err := verbindung.Exec(ctx, `CREATE DATABASE `+pgBezeichner(name)); err != nil {
		t.Fatalf("Anlegen von %s fehlgeschlagen: %v", name, err)
	}
	t.Cleanup(func() {
		c := verbinde(t, adminDSN)
		if _, err := c.Exec(context.Background(), `DROP DATABASE IF EXISTS `+pgBezeichner(name)+` WITH (FORCE)`); err != nil {
			t.Logf("Probe-Datenbank %s blieb liegen: %v", name, err)
		}
	})

	return name, tauscheDatenbank(adminDSN, name)
}

// pgBezeichner setzt einen Bezeichner in Anführungszeichen, wo Postgres keine Parameter erlaubt.
func pgBezeichner(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// tauscheDatenbank ersetzt den Datenbanknamen im DSN.
func tauscheDatenbank(dsn, neu string) string {
	konfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return dsn
	}
	sslmode := "disable"
	if v := konfig.RuntimeParams["sslmode"]; v != "" {
		sslmode = v
	}
	zugang := konfig.User
	if konfig.Password != "" {
		zugang += ":" + konfig.Password
	}
	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", zugang, konfig.Host, konfig.Port, neu, sslmode)
}

func verbinde(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	verbindung, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Verbindung zu %s fehlgeschlagen: %v", dsn, err)
	}
	t.Cleanup(func() { _ = verbindung.Close(context.Background()) }) //nolint:errcheck
	return verbindung
}

// befuelleQuelle spielt das echte schema.sql ein und legt einen wiedererkennbaren Datensatz an.
// Eine Sicherung eines Spielzeugschemas bewiese nichts.
func befuelleQuelle(t *testing.T, dsn string) {
	t.Helper()

	_, dieseDatei, _, _ := runtime.Caller(0)
	schema, err := os.ReadFile(filepath.Join(filepath.Dir(dieseDatei), "..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql nicht lesbar: %v", err)
	}

	verbindung := verbinde(t, dsn)
	ctx := context.Background()
	if _, err := verbindung.Exec(ctx, string(schema)); err != nil {
		t.Fatalf("schema.sql konnte nicht eingespielt werden: %v", err)
	}

	var titelID string
	if err := verbindung.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, isbn) VALUES ($1,$2) RETURNING id`,
		"Probe-Titel für die Restore-Probe", "9783161484100").Scan(&titelID); err != nil {
		t.Fatalf("Probedaten (Titel): %v", err)
	}
	if _, err := verbindung.Exec(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1,$2)`,
		titelID, "B-DRILL-1"); err != nil {
		t.Fatalf("Probedaten (Exemplar): %v", err)
	}
	if _, err := verbindung.Exec(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('S-DRILL-1','Probe','Schüler','7a',2030)`); err != nil {
		t.Fatalf("Probedaten (Schüler): %v", err)
	}
}

// zaehleAlleTabellen liefert für jede Tabelle des Schemas die Zahl der Zeilen. Der Vergleich
// über alle Tabellen ist die aussagekräftige Gegenprobe: Er bemerkt auch eine Tabelle, an die
// beim Schreiben des Tests niemand gedacht hat.
func zaehleAlleTabellen(t *testing.T, dsn string) map[string]int {
	t.Helper()

	verbindung := verbinde(t, dsn)
	ctx := context.Background()

	zeilen, err := verbindung.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		t.Fatalf("Tabellenliste: %v", err)
	}
	var namen []string
	for zeilen.Next() {
		var n string
		if err := zeilen.Scan(&n); err != nil {
			zeilen.Close()
			t.Fatalf("Tabellenname: %v", err)
		}
		namen = append(namen, n)
	}
	zeilen.Close()
	if err := zeilen.Err(); err != nil {
		t.Fatalf("Tabellenliste: %v", err)
	}

	bestand := make(map[string]int, len(namen))
	for _, n := range namen {
		var anzahl int
		if err := verbindung.QueryRow(ctx, `SELECT count(*) FROM `+pgBezeichner(n)).Scan(&anzahl); err != nil {
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

// entschluesseleUndPacke bildet genau den Weg von cmd/restore-backup nach:
// backupkrypto.Entschluessele, dann gunzip.
func entschluesseleUndPacke(t *testing.T, schluessel string, roh []byte) string {
	t.Helper()

	komprimiert, err := backupkrypto.Entschluessele(schluessel, roh)
	if err != nil {
		t.Fatalf("Entschlüsselung fehlgeschlagen: %v", err)
	}
	entpacker, err := gzip.NewReader(bytes.NewReader(komprimiert))
	if err != nil {
		t.Fatalf("gzip-Header ungültig: %v", err)
	}
	defer func() { _ = entpacker.Close() }() //nolint:errcheck

	sqlText, err := io.ReadAll(entpacker)
	if err != nil {
		t.Fatalf("Dekomprimierung fehlgeschlagen: %v", err)
	}
	return string(sqlText)
}
