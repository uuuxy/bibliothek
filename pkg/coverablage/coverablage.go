// Package coverablage kennt den Ort der lokal gespeicherten Cover: Es prüft den Pfad einer
// Cover-URL, öffnet das Verzeichnis und entfernt eine Cover-Datei.
//
// Ohne Bildbibliothek. pkg/coverdatei wandelt Cover für die PDF-Erzeuger und zieht dafür die
// WebP-Bibliothek (cgo) mit; das Löschen eines Titels braucht nur den Pfad, und über
// repository binden es auch die Werkzeuge des Images ein, die ohne cgo gebaut werden.
package coverablage

import (
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
	defer closeutil.LogClose(wurzel, "coverablage wurzel")

	// Lstat: Eine Verknüpfung oder ein Ordner unter diesem Namen ist kein Cover.
	if info, err := wurzel.Lstat(rel); err != nil || !info.Mode().IsRegular() {
		return nil
	}
	if err := wurzel.Remove(rel); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
