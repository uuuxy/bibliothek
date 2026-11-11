package api

import (
	"net/http"
	"strings"
	"testing"

	"bibliothek/internal/tuertest"
)

// Über der Grenze des JSON-Rumpfs (api/tuer, maxRumpfJSON) antwortet jede Tür, die ihren Rumpf
// nicht über LeseAnfrage liest, mit 413, bevor sie die Datenbank fragt. „Exemplar aussondern" und
// „Leser zusammenführen" antworteten mit ihrer eigenen 400-Meldung.
func TestRumpfUeberDerGrenzeIst413VorDerDatenbank(t *testing.T) {
	s := &Server{}
	const kennung = "11111111-1111-1111-1111-111111111111"
	zuGross := `{"x":"` + strings.Repeat("x", 6<<20) + `"}`
	faelle := []struct {
		name, methode, pfad string
		h                   http.HandlerFunc
	}{
		{"Exemplar aussondern", http.MethodDelete, "/api/buecher/exemplare/" + kennung,
			mitAnmeldung(mitKennungImPfad(s.katalog().ExemplarAussondern(nil), kennung))},
		{"Leser zusammenführen", http.MethodPost, "/api/schueler/" + kennung + "/zusammenfuehren",
			mitAnmeldung(mitKennungImPfad(s.leser().LeserZusammenfuehren(nil), kennung))},
		{"Schuljahreswechsel", http.MethodPost, "/api/schuljahreswechsel", mitAnmeldung(s.leser().Schuljahreswechsel())},
		{"Klassensätze ändern", http.MethodPut, "/api/klassensaetze", mitAnmeldung(s.katalog().KlassensaetzeAendern())},
		{"Klassensätze zuordnen", http.MethodPost, "/api/klassensaetze", mitAnmeldung(s.katalog().KlassensaetzeZuordnen())},
		{"Titel anlegen", http.MethodPost, "/api/buecher/titel", mitAnmeldung(s.katalog().TitelAnlegen())},
		{"Titel ändern", http.MethodPatch, "/api/buecher/titel/" + kennung,
			mitAnmeldung(mitKennungImPfad(s.katalog().TitelAendern(), kennung))},
		{"fremde Cover neu laden", http.MethodPost, "/api/buecher/cover-fremd", mitAnmeldung(s.katalog().FremdeCoverNeuLaden())},
		{"markierte Titel löschen", http.MethodPost, "/api/buecher/titel/loeschen", mitAnmeldung(s.katalog().MarkierteTitelLoeschen())},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if status := tuertest.StatusOhneDB(t, f.h, f.methode, f.pfad, zuGross); status != http.StatusRequestEntityTooLarge {
				t.Fatalf("Status %d, erwartet 413", status)
			}
		})
	}
}
