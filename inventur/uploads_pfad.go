package inventur

import (
	"fmt"
	"os"

	"bibliothek/pkg/closeutil"
)

// Zugriffe auf Dateien unter uploads/ laufen über os.OpenRoot statt über einen Vergleich des
// Präfixes.
//
// filepath.Join, dann filepath.Clean, dann strings.HasPrefix gegen "uploads/" hält den
// geradlinigen Fall auf, ist aber eine Prüfung im Programm über einen Pfad, der danach trotzdem
// ans Betriebssystem geht: Zwischen Prüfung und Zugriff kann sich der Pfad ändern (ein Symlink in
// uploads/ genügt, und HasPrefix sieht Symlinks nicht). os.OpenRoot bindet die Auflösung im
// Betriebssystem an das Verzeichnis: Ein Name, der herausführt, scheitert am Kernel, nicht an
// einer Prüfung der Zeichenkette. So arbeitet auch der Rest des Projekts (api/katalog/cover.go,
// pkg/coverdatei, api/router.go, cmd/migrate-fotos).

const uploadsVerzeichnis = "uploads"

// mitUploadsRoot legt uploads/ an, öffnet es als Wurzel und übergibt sie an fn. Die
// Wurzel wird in jedem Fall geschlossen.
func mitUploadsRoot(fn func(*os.Root) error) error {
	if err := os.MkdirAll(uploadsVerzeichnis, 0750); err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht angelegt werden: %w", err)
	}
	wurzel, err := os.OpenRoot(uploadsVerzeichnis)
	if err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht geöffnet werden: %w", err)
	}
	defer closeutil.Schliesse(wurzel, "uploads dir")
	return fn(wurzel)
}

// schreibeUploadDatei legt inhalt unter uploads/<name> ab. name darf kein Verzeichnis
// enthalten; jeder Versuch, das Verzeichnis zu verlassen, scheitert am Betriebssystem.
func schreibeUploadDatei(name string, inhalt []byte) error {
	return mitUploadsRoot(func(wurzel *os.Root) error {
		datei, err := wurzel.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("datei %q konnte nicht angelegt werden: %w", name, err)
		}
		defer closeutil.Schliesse(datei, "upload file")

		if _, err := datei.Write(inhalt); err != nil {
			return fmt.Errorf("datei %q konnte nicht geschrieben werden: %w", name, err)
		}
		return nil
	})
}

// uploadDateiVorhanden sagt, ob uploads/<name> schon als Datei da ist.
func uploadDateiVorhanden(name string) bool {
	vorhanden := false
	_ = mitUploadsRoot(func(wurzel *os.Root) error { //nolint:errcheck // ohne Verzeichnis gibt es die Datei nicht
		info, err := wurzel.Stat(name)
		vorhanden = err == nil && info.Mode().IsRegular()
		return nil
	})
	return vorhanden
}

// loescheUploadDatei entfernt uploads/<name>. Eine nicht vorhandene Datei ist kein
// Fehler — der Aufrufer räumt nur auf.
func loescheUploadDatei(name string) error {
	return mitUploadsRoot(func(wurzel *os.Root) error {
		if err := wurzel.Remove(name); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("datei %q konnte nicht gelöscht werden: %w", name, err)
		}
		return nil
	})
}
