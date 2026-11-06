package tuer

import (
	"mime"
	"net/http"
	"strconv"

	"bibliothek/pkg/httpresp"
)

// Auslieferung sagt dem Browser, ob er eine Datei im Tab zeigt oder in den Download-Ordner legt.
// Dokumente mit Personendaten gehen im Tab auf (ARCHITEKTUR.md 8.9, api/anhang_ratsche_test.go).
type Auslieferung string

const (
	// Zeigen öffnet die Datei im Tab (Content-Disposition: inline).
	Zeigen Auslieferung = "inline"
	// Speichern legt die Datei in den Download-Ordner (Content-Disposition: attachment).
	Speichern Auslieferung = "attachment"
)

// SetzeDateiKopf setzt Typ und Namen einer Datei, deren Inhalt die Tür danach selbst schreibt.
// mime.FormatMediaType kodiert den Namen nach RFC 6266: Ein Anführungszeichen wird maskiert,
// ein Umlaut steht in filename*.
func SetzeDateiKopf(w http.ResponseWriter, typ string, art Auslieferung, dateiname string) {
	w.Header().Set(HeaderContentType, typ)
	w.Header().Set(HeaderContentDisposition,
		mime.FormatMediaType(string(art), map[string]string{"filename": dateiname}))
}

// AntworteDatei schickt eine fertige Datei mit Typ, Namen und Länge.
func AntworteDatei(w http.ResponseWriter, typ string, art Auslieferung, dateiname string, inhalt []byte) {
	SetzeDateiKopf(w, typ, art, dateiname)
	w.Header().Set(HeaderContentLength, strconv.Itoa(len(inhalt)))
	httpresp.Schreibe(w, inhalt)
}
