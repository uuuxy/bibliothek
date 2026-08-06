package inventur

import (
	"fmt"
	"os"

	"bibliothek/pkg/closeutil"
)

// Dateizugriffe unter uploads/ laufen über os.OpenRoot statt über einen Präfixvergleich.
//
// Vorher stand an drei Stellen dasselbe Muster: filepath.Join, dann filepath.Clean, dann
// strings.HasPrefix gegen "uploads/". Das hält den geradlinigen Fall auf, ist aber eine
// Prüfung im Programm über einen Pfad, der danach trotzdem ans Betriebssystem geht —
// zwischen Prüfung und Zugriff kann sich der Pfad ändern (ein Symlink in uploads/ genügt,
// und HasPrefix sieht Symlinks nicht). os.OpenRoot bindet die Auflösung OS-seitig an das
// Verzeichnis: Ein Name, der herausführt, scheitert am Kernel, nicht an einer
// Zeichenkettenprüfung.
//
// Der Rest des Projekts arbeitet bereits so (api/image_caching.go, api/mahnwesen_pdf.go,
// api/router.go, cmd/migrate-fotos). Die Inventur war die letzte Ausnahme.

const uploadsVerzeichnis = "uploads"

// mitUploadsRoot legt uploads/ an, öffnet es als Wurzel und übergibt sie an fn. Die
// Wurzel wird in jedem Fall geschlossen.
func mitUploadsRoot(fn func(*os.Root) error) error {
	if err := os.MkdirAll(uploadsVerzeichnis, 0750); err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht angelegt werden: %w", err)
	}
	root, err := os.OpenRoot(uploadsVerzeichnis)
	if err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht geöffnet werden: %w", err)
	}
	defer closeutil.LogClose(root, "uploads dir")
	return fn(root)
}

// schreibeUploadDatei legt inhalt unter uploads/<name> ab. name darf kein Verzeichnis
// enthalten; jeder Versuch, das Verzeichnis zu verlassen, scheitert am Betriebssystem.
func schreibeUploadDatei(name string, inhalt []byte) error {
	return mitUploadsRoot(func(root *os.Root) error {
		datei, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("datei %q konnte nicht angelegt werden: %w", name, err)
		}
		defer closeutil.LogClose(datei, "upload file")

		if _, err := datei.Write(inhalt); err != nil {
			return fmt.Errorf("datei %q konnte nicht geschrieben werden: %w", name, err)
		}
		return nil
	})
}

// loescheUploadDatei entfernt uploads/<name>. Eine nicht vorhandene Datei ist kein
// Fehler — der Aufrufer räumt nur auf.
func loescheUploadDatei(name string) error {
	return mitUploadsRoot(func(root *os.Root) error {
		if err := root.Remove(name); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("datei %q konnte nicht gelöscht werden: %w", name, err)
		}
		return nil
	})
}
