package api

import (
	"net/http"
	"testing"
)

// Ein Inventurstart ohne brauchbaren Bereich wird abgewiesen, bevor eine Inventur
// entsteht. Der Server hat hier keine Datenbank; erreicht der Ablauf sie doch, meldet
// antwortOhneDB das als Fehler.
//
// Die leere Signatur ist der Fall, um den es geht: Sie ist der Anfang jeder Signatur, und
// aus der Inventur eines Regals würde eine über den ganzen Bestand, deren Abschluss alles
// nicht Gescannte als Verlust bucht.
func TestInventurStart_UnbrauchbarerBereichIst400VorDerDatenbank(t *testing.T) {
	s := &Server{}
	faelle := map[string]string{
		"Art fehlt":                   `{}`,
		"unbekannte Art":              `{"type":"regal"}`,
		"Signatur fehlt":              `{"type":"signature"}`,
		"Signatur leer":               `{"type":"signature","signatur":""}`,
		"Signatur nur Leerzeichen":    `{"type":"signature","signatur":"   "}`,
		"Filter ohne Fach und Klasse": `{"type":"filter"}`,
		"kein JSON":                   `type`,
	}
	for name, rumpf := range faelle {
		t.Run(name, func(t *testing.T) {
			if status := antwortOhneDB(t, s.InventurStartHandler(), http.MethodPost, "/api/inventur/start", rumpf); status != http.StatusBadRequest {
				t.Fatalf("Status %d, erwartet 400", status)
			}
		})
	}
}
