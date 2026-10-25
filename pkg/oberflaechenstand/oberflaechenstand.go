// Package oberflaechenstand liest, welchen Stand der Oberfläche der Server ausliefert.
//
// Der Stand ist der Name der Programmdatei, mit der index.html die Oberfläche lädt
// (/assets/index-….js). Der Name trägt einen Prüfwert und wechselt mit jeder geänderten
// Oberfläche. Er steht ohnehin in der Startseite, anders als der Commit im Image (GIT_COMMIT),
// der verriete, welcher Stand des öffentlichen Repositorys läuft. Der Server nennt ihn in jeder
// Antwort; ein Fenster, das mit einem anderen läuft, bittet ums Neuladen
// (frontend/src/lib/stores/programmstand.svelte.js).
package oberflaechenstand

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"regexp"
)

// Kopf ist das Kopffeld, in dem jede Antwort den Stand nennt.
const Kopf = "X-Oberflaechenstand"

var programmdateiInIndex = regexp.MustCompile(`<script\b[^>]*\btype="module"[^>]*\bsrc="([^"]+)"`)

// Lese liest den Stand aus index.html der gebauten Oberfläche; leer ohne sie.
func Lese(oberflaeche *os.Root) string {
	if oberflaeche == nil {
		return ""
	}
	inhalt, err := fs.ReadFile(oberflaeche.FS(), "index.html")
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		slog.Warn("oberflaechenstand: index.html unlesbar, offene Fenster merken kein Update", "fehler", err)
		return ""
	}
	treffer := programmdateiInIndex.FindSubmatch(inhalt)
	if treffer == nil {
		slog.Warn("oberflaechenstand: index.html nennt keine Programmdatei, offene Fenster merken kein Update")
		return ""
	}
	return string(treffer[1])
}
