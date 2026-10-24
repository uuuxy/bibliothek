// Package coverablage kennt den Ort der lokal gespeicherten Cover: Es prüft den Pfad einer
// Cover-URL, öffnet das Verzeichnis, legt eine Cover-Datei ab und entfernt sie.
//
// Ohne Bildbibliothek. pkg/coverdatei wandelt Cover für die PDF-Erzeuger und zieht dafür die
// WebP-Bibliothek (cgo) mit; das Löschen eines Titels braucht nur den Pfad, und über
// repository binden es auch die Werkzeuge des Images ein, die ohne cgo gebaut werden.
package coverablage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bibliothek/pkg/closeutil"
)

// Wurzel ist das Verzeichnis aller lokal gespeicherten Bilder, relativ zum
// Arbeitsverzeichnis des Servers.
const Wurzel = "uploads"

// Zerlege prüft die Cover-URL und liefert den Dateipfad zweimal: einmal relativ zur Wurzel
// (so verlangt es os.Root) und einmal vollständig, weil die PDF-Erzeuger ihn als Namen ihres
// Bildspeichers führen. ok=false heißt: nicht verwendbar.
//
// Die Herkunftsprüfung ist kein Schutz — das ist os.Root beim Zugriff (OeffneWurzel). Sie
// steht hier als Vorfilter: Die meisten Titel haben kein Cover, und für die soll kein
// Verzeichnis geöffnet werden.
func Zerlege(coverURL string) (rel, pfad string, ok bool) {
	if !strings.HasPrefix(coverURL, "/"+Wurzel+"/") {
		return "", "", false
	}
	pfad = filepath.Clean(strings.TrimPrefix(coverURL, "/"))
	rel, err := filepath.Rel(Wurzel, pfad)
	if err != nil {
		return "", "", false
	}
	return rel, pfad, true
}

// OeffneWurzel öffnet das Upload-Verzeichnis als os.Root. Erst diese Klammer hält auch den
// Fall auf, den die Textprüfung nicht sehen kann: eine Verknüpfung innerhalb von uploads,
// die nach draußen zeigt. os.Stat würde ihr folgen, root.Stat verweigert sie.
func OeffneWurzel() (*os.Root, bool) {
	wurzel, err := os.OpenRoot(Wurzel)
	if err != nil {
		return nil, false
	}
	return wurzel, true
}

// Loesche entfernt die Datei eines lokal gespeicherten Covers, etwa nachdem sein Titel
// gelöscht ist. Nur Dateien direkt in der Wurzel: In ihren Unterordnern liegen andere Bilder
// (Ausweisfotos, der Zwischenspeicher des Cover-Abrufs), und die Cover-URL eines Titels kommt
// aus einer Maske. Eine fehlende Datei und eine Adresse außerhalb sind kein Fehler; der
// Aufrufer räumt nur auf.
func Loesche(coverURL string) error {
	rel, _, ok := Zerlege(coverURL)
	if !ok || rel == "." || rel != filepath.Base(rel) {
		return nil
	}
	wurzel, ok := OeffneWurzel()
	if !ok {
		return nil
	}
	defer closeutil.Schliesse(wurzel, "coverablage wurzel")

	// Lstat: Eine Verknüpfung oder ein Ordner unter diesem Namen ist kein Cover.
	if info, err := wurzel.Lstat(rel); err != nil || !info.Mode().IsRegular() {
		return nil
	}
	if err := wurzel.Remove(rel); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Schreibe legt inhalt als Datei name direkt in der Wurzel ab und legt die Wurzel an, wenn es sie
// noch nicht gibt. Ein Name, der aus der Wurzel herausführt, scheitert am Betriebssystem
// (os.Root), nicht an einer Prüfung der Zeichenkette: Zwischen Prüfung und Zugriff könnte sich
// der Pfad ändern, eine Verknüpfung in uploads genügt.
func Schreibe(name string, inhalt []byte) error {
	return mitAngelegterWurzel(func(wurzel *os.Root) error {
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

// IstVorhanden sagt, ob name schon als Datei direkt in der Wurzel liegt.
func IstVorhanden(name string) bool {
	vorhanden := false
	_ = mitAngelegterWurzel(func(wurzel *os.Root) error { //nolint:errcheck // ohne Verzeichnis gibt es die Datei nicht
		info, err := wurzel.Stat(name)
		vorhanden = err == nil && info.Mode().IsRegular()
		return nil
	})
	return vorhanden
}

// mitAngelegterWurzel legt die Wurzel an, öffnet sie und übergibt sie an fn. Die Wurzel wird in
// jedem Fall geschlossen.
func mitAngelegterWurzel(fn func(*os.Root) error) error {
	if err := os.MkdirAll(Wurzel, 0750); err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht angelegt werden: %w", err)
	}
	wurzel, err := os.OpenRoot(Wurzel)
	if err != nil {
		return fmt.Errorf("uploads-Verzeichnis konnte nicht geöffnet werden: %w", err)
	}
	defer closeutil.Schliesse(wurzel, "uploads dir")
	return fn(wurzel)
}
