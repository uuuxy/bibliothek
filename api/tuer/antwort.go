package tuer

import (
	"errors"
	"net/http"

	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// AntworteJSON schreibt die Antwort mit dem Status als JSON.
func AntworteJSON(w http.ResponseWriter, status int, antwort any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	httpresp.SchreibeJSON(w, antwort)
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
	MeldungIDLeer                = "Die Kennung fehlt"
	MeldungEinstellungenUnlesbar = "Einstellungen konnten nicht gelesen werden"
	MeldungTitelNichtGefunden    = "Titel nicht gefunden"
	MeldungAnfangKeinDatum       = "Der Anfang des Zeitraums muss ein Datum der Form JJJJ-MM-TT sein"
	MeldungEndeKeinDatum         = "Das Ende des Zeitraums muss ein Datum der Form JJJJ-MM-TT sein"
)

// GrundAus liefert den Grund einer Ablehnung aus repository (repository.RegelFehler), sonst "".
func GrundAus(err error) string {
	var regel *repository.RegelFehler
	if errors.As(err, &regel) {
		return regel.Grund
	}
	return ""
}

// MitGrund setzt den Grund einer Ablehnung aus repository hinter den Satz der Tür; ohne Grund
// bleibt der Satz allein.
func MitGrund(satz string, err error) string {
	if grund := GrundAus(err); grund != "" {
		return satz + ": " + grund
	}
	return satz
}
