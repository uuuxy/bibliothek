package jobs

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"bibliothek/internal/backupkrypto"
	"bibliothek/pkg/closeutil"
)

// entschaerfePgpass maskiert Backslash und Doppelpunkt, wie es das Format von .pgpass verlangt.
func entschaerfePgpass(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return strings.ReplaceAll(s, ":", "\\:")
}

// minBackupSchluesselLaenge ist die Mindestlänge der Passphrase für die Sicherung.
//
// Den Schlüssel leitet scrypt ab (internal/backupkrypto), speicherhart und je Datei gesalzen;
// das nimmt dem Raten an einer entwendeten Sicherung die Geschwindigkeit. Die Prüfung der Länge
// bleibt als zweite Verteidigung: scrypt erschwert das Raten, ersetzt aber keine Entropie der
// Passphrase, eine kurze bleibt kurz.
const minBackupSchluesselLaenge = 32

// IstSchluesselSchwach meldet, ob die Passphrase der Sicherung zu kurz ist. Ein leerer Schlüssel
// gilt hier nicht als schwach: Das ist der eigene, schwerere Fall „gar keine Sicherung".
func IstSchluesselSchwach(schluessel string) bool {
	return schluessel != "" && len(schluessel) < minBackupSchluesselLaenge
}

// BackupLauf ist die tägliche verschlüsselte Sicherung der Datenbank. Er ruft pg_dump auf (muss
// im PATH stehen), packt mit gzip und verschlüsselt mit AES-256-GCM, mit einem Schlüssel aus
// BACKUP_ENCRYPTION_KEY.
//
// Umgebungsvariablen:
//   - DATABASE_URL          – DSN von PostgreSQL (im Betrieb schon gesetzt)
//   - BACKUP_ENCRYPTION_KEY – Passphrase mit mindestens 32 Zeichen, aus der der Schlüssel entsteht
//   - BACKUP_DIR            – Zielverzeichnis (Vorgabe: ./backups)
type BackupLauf struct{}

// FuehreAus führt die ganze Sicherung aus: Dump, gzip, Verschlüsselung mit AES-256-GCM. Endet ctx
// vorher, bricht pg_dump ab, und es entsteht keine Datei: Der Dump liegt bis zum Schreiben im
// Speicher, und die Datei entsteht in einem Zug.
func (b *BackupLauf) FuehreAus(ctx context.Context) {
	schluessel, dsn, verzeichnis, ok := leseBackupUmgebung()
	if !ok {
		return
	}

	stempel := time.Now().UTC().Format(sicherungsStempel)
	zieldatei := filepath.Join(verzeichnis, fmt.Sprintf("backup_%s.sql.gz.enc", stempel))

	slog.Info("Backup: starting daily PostgreSQL backup", "datei", zieldatei)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	gepackt, ok := erzeugeGepacktenDump(ctx, dsn)
	if !ok {
		return
	}

	// AES-256-GCM mit einem Schlüssel aus scrypt (internal/backupkrypto).
	verschluesselt, err := backupkrypto.Verschluessele(schluessel, gepackt)
	if err != nil {
		// #nosec G706
		slog.Error("Backup: encryption failed", "fehler", err)
		return
	}

	// Die verschlüsselte Sicherung mit engen Rechten auf die Platte schreiben: nur der
	// Eigentümer liest.
	// #nosec G304 - zieldatei entsteht aus BACKUP_DIR und einem Zeitstempel
	if err := os.WriteFile(zieldatei, verschluesselt, 0600); err != nil {
		// #nosec G706
		slog.Error("Backup: writing backup file failed", "fehler", err)
		return
	}

	groesseMB := float64(len(verschluesselt)) / 1024 / 1024
	// #nosec G706
	slog.Info("Backup: completed successfully", "datei", zieldatei, "groesse_mb", math.Round(groesseMB*100)/100)

	// Kopie außer Haus nach S3, dort mit derselben Aufbewahrung (backup_auslagerung.go).
	if ziel, ok := leseS3Ziel(); ok && schreibeBackupNachS3(ctx, ziel, filepath.Base(zieldatei), verschluesselt) {
		rotiereBackupsInS3(ctx, ziel, AufbewahrteNaechte, AufbewahrteWochen)
	}

	// Aufbewahrung: die jüngsten Nächte und je Woche ein älterer Stand (backup_aufbewahrung.go).
	rotiereBackups(verzeichnis, AufbewahrteNaechte, AufbewahrteWochen)
}

// leseBackupUmgebung liest die Umgebungsvariablen der Sicherung und legt bei Bedarf das
// Zielverzeichnis an. ok=false heißt Abbruch, der Grund steht im Log (Schlüssel oder DSN fehlt,
// Verzeichnis lässt sich nicht anlegen).
func leseBackupUmgebung() (schluessel, dsn, verzeichnis string, ok bool) {
	schluessel = os.Getenv("BACKUP_ENCRYPTION_KEY")
	if schluessel == "" {
		slog.Error("Backup: BACKUP_ENCRYPTION_KEY not set – skipping encrypted backup")
		return "", "", "", false
	}
	// Warnen, aber weitermachen: Eine Sicherung mit kurzer Passphrase ist viel besser als gar
	// keine. Sichtbar wird der Zustand über /api/admin/system/backup-status in der Verwaltung; eine
	// Logzeile allein liest niemand.
	if IstSchluesselSchwach(schluessel) {
		slog.Warn("Backup: BACKUP_ENCRYPTION_KEY ist kürzer als empfohlen – scrypt erschwert das Raten, ersetzt aber keine Entropie; kurze Passphrasen bleiben angreifbar", "laenge", len(schluessel), "empfohlen", minBackupSchluesselLaenge)
	}

	dsn = os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("Backup: DATABASE_URL not set – skipping backup")
		return "", "", "", false
	}

	verzeichnis = os.Getenv("BACKUP_DIR")
	if verzeichnis == "" {
		verzeichnis = "./backups"
	}
	if err := os.MkdirAll(verzeichnis, 0750); err != nil {
		// #nosec G706
		slog.Error("Backup: cannot create backup directory", "fehler", err)
		return "", "", "", false
	}
	return schluessel, dsn, verzeichnis, true
}

// erzeugePgpassDatei schreibt die Zugangsdaten für pg_dump und psql in eine vorübergehende
// .pgpass-Datei und liefert ihren Pfad; der Aufrufer löscht sie.
func erzeugePgpassDatei(konfig *pgconn.Config) (string, error) {
	datei, err := os.CreateTemp("", "pgpass-*")
	if err != nil {
		return "", fmt.Errorf("konnte pgpass-Datei nicht erstellen: %w", err)
	}
	defer func() { _ = datei.Close() }() //nolint:errcheck

	port := fmt.Sprintf("%d", konfig.Port)
	if port == "0" {
		port = "5432"
	}

	zeile := fmt.Sprintf("%s:%s:%s:%s:%s\n",
		entschaerfePgpass(konfig.Host),
		entschaerfePgpass(port),
		entschaerfePgpass(konfig.Database),
		entschaerfePgpass(konfig.User),
		entschaerfePgpass(konfig.Password),
	)
	if _, err := datei.WriteString(zeile); err != nil {
		return "", fmt.Errorf("konnte in pgpass-Datei nicht schreiben: %w", err)
	}

	return datei.Name(), nil
}

// erzeugeGepacktenDump ruft pg_dump auf (die Zugangsdaten stehen in einer vorübergehenden
// .pgpass-Datei) und liefert den mit gzip gepackten SQL-Dump. Jeder Fehler steht mit seiner
// Ursache im Log; ok=false heißt Abbruch.
func erzeugeGepacktenDump(ctx context.Context, dsn string) (gepackt []byte, ok bool) {
	// Den DSN lesen, um die Angaben der Verbindung für pg_dump zu gewinnen.
	konfig, err := pgconn.ParseConfig(dsn)
	if err != nil {
		slog.Error("Backup: failed to parse DSN", "fehler", err)
		return nil, false
	}

	pgpassDatei, err := erzeugePgpassDatei(konfig)
	if err != nil {
		slog.Error("Backup: pgpass-Datei nicht angelegt", "fehler", err)
		return nil, false
	}
	defer func() { _ = os.Remove(pgpassDatei) }() //nolint:errcheck

	port := fmt.Sprintf("%d", konfig.Port)
	if port == "0" {
		port = "5432"
	}

	// #nosec G204 - die Argumente stammen aus dem gelesenen DSN der eigenen Konfiguration
	pgDump := exec.CommandContext(ctx, "pg_dump",
		"--host="+konfig.Host,
		"--port="+port,
		"--username="+konfig.User,
		"--dbname="+konfig.Database,
		"--no-password",
		"--format=plain",
		"--encoding=UTF8",
		"--verbose",
	)
	pgDump.Env = append(os.Environ(), "PGPASSFILE="+pgpassDatei)
	sqlLeser, sqlSchreiber := io.Pipe()
	pgDump.Stdout = sqlSchreiber
	pgDump.Stderr = os.Stderr

	if err := pgDump.Start(); err != nil {
		// #nosec G706
		slog.Error("Backup: pg_dump start failed", "fehler", err)
		return nil, false
	}

	// Wait in einer Goroutine: Endet pg_dump, schließt sie das Schreibende der Leitung.
	endeFehler := make(chan error, 1)
	go func() {
		err := pgDump.Wait()
		_ = sqlSchreiber.Close() //nolint:errcheck
		endeFehler <- err
	}()

	// Eine zweite Leitung trägt den mit gzip gepackten Dump.
	gepacktLeser, gepacktSchreiber := io.Pipe()

	// Die Goroutine packt die Ausgabe von pg_dump.
	go func() {
		packer := gzip.NewWriter(gepacktSchreiber)
		if _, err := io.Copy(packer, sqlLeser); err != nil {
			gepacktSchreiber.CloseWithError(err)
			return
		}
		// Close schreibt den Abschluss von gzip; scheitert es, erfährt es die lesende Seite,
		// damit sie kein abgeschnittenes Archiv übernimmt.
		if err := packer.Close(); err != nil {
			gepacktSchreiber.CloseWithError(err)
			return
		}
		gepacktSchreiber.CloseWithError(nil) // sauberes Ende für die lesende Seite
	}()

	// Den ganzen gepackten Dump lesen.
	inhalt, err := io.ReadAll(gepacktLeser)
	if err != nil {
		// #nosec G706
		slog.Error("Backup: compression failed", "fehler", err)
		return nil, false
	}

	if err := <-endeFehler; err != nil {
		// #nosec G706
		slog.Error("Backup: pg_dump finished with error", "fehler", err)
		return nil, false
	}

	return inhalt, true
}

// rotiereBackups löscht, was die Aufbewahrung nicht hält: Die naechte jüngsten Sicherungen
// bleiben, von den älteren je Kalenderwoche die jüngste für wochen Wochen (zuLoeschen). Die
// Namen tragen den Zeitstempel, die Reihenfolge der Liste ist die zeitliche (backup_dateien.go).
// Gelöscht wird über die Wurzel: Ein Symlink verschwindet als Link, sein Ziel bleibt.
func rotiereBackups(verzeichnis string, naechte, wochen int) {
	wurzel, err := os.OpenRoot(verzeichnis)
	if err != nil {
		return
	}
	defer closeutil.Schliesse(wurzel, "backup dir")
	dateien, err := leseBackups(wurzel)
	if err != nil {
		// Nicht stumm aufhören: Ohne Aufbewahrung wächst das Verzeichnis, bis die Platte voll ist,
		// und genau dann entsteht auch keine neue Sicherung mehr. Der Fehler darf deshalb nicht im
		// gewöhnlichen Fall „noch nicht genug Sicherungen" verschwinden.
		slog.Warn("Backup rotation: Verzeichnis nicht lesbar, es wird nichts rotiert", "verzeichnis", verzeichnis, "fehler", err)
		return
	}
	for _, d := range zuLoeschen(dateien, naechte, wochen) {
		if err := wurzel.Remove(d.Name); err != nil {
			// #nosec G706
			slog.Warn("Backup rotation: failed to delete", "datei", d.Name, "fehler", err)
		} else {
			// #nosec G706
			slog.Info("Backup rotation: deleted old backup", "datei", d.Name)
		}
	}
}
