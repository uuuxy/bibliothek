package tuer

import (
	"net/http"
	"strconv"

	"bibliothek/pkg/httpresp"
)

// AntworteJSON schreibt die Antwort mit dem Status als JSON.
func AntworteJSON(w http.ResponseWriter, status int, antwort any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	httpresp.SchreibeJSON(w, antwort)
}

// HeaderGrenze nennt in der Antwort einer Liste, die der Server kappt, die Grenze, die für diese
// Antwort gilt. Daran sagt die Oberfläche, dass es hinter der letzten Zeile weitergeht
// (frontend/src/lib/utils/listenGrenze.js); die Zahl steht so nur im Server.
const HeaderGrenze = "X-Grenze"

// AntworteListe schreibt eine Liste als JSON und nennt im Kopf HeaderGrenze die Grenze, bei der
// der Server sie kappt; 0 heißt, für diese Antwort gilt keine.
func AntworteListe(w http.ResponseWriter, liste any, grenze int) {
	if grenze > 0 {
		w.Header().Set(HeaderGrenze, strconv.Itoa(grenze))
	}
	AntworteJSON(w, http.StatusOK, liste)
}

// AntworteErfolg antwortet mit {"status": "success"}, wo eine Tür nichts anderes zurückgibt.
func AntworteErfolg(w http.ResponseWriter) {
	AntworteJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Namen der Kopffelder einer HTTP-Antwort. Sie stehen hier statt als Text an jeder Tür, weil
// SonarQube wiederholte Zeichenketten meldet (go:S1192).
const (
	HeaderContentType        = "Content-Type"
	HeaderContentDisposition = "Content-Disposition"
	HeaderContentLength      = "Content-Length"
	HeaderCacheControl       = "Cache-Control"
)

// Werte des Kopffelds Content-Type.
const (
	ContentTypeJSON = "application/json"
	ContentTypePDF  = "application/pdf"
)

// Meldungen, die mehrere Türen wörtlich gleich ausgeben.
const (
	MeldungIDLeer                   = "Die Kennung fehlt"
	MeldungEinstellungenUnlesbar    = "Einstellungen konnten nicht gelesen werden"
	MeldungTitelNichtGefunden       = "Titel nicht gefunden"
	MeldungSchuelerKennungUngueltig = "Die Kennung des Schülers ist ungültig"
	MeldungAnfangKeinDatum          = "Der Anfang des Zeitraums muss ein Datum der Form JJJJ-MM-TT sein"
	MeldungEndeKeinDatum            = "Das Ende des Zeitraums muss ein Datum der Form JJJJ-MM-TT sein"
)
