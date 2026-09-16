package api

import (
	"encoding/json"
	"strings"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Die Theken-Antwort trägt von einem Kollegen nur Identität — keine Anschrift, keine
// Kontaktdaten, kein Geburtsdatum. Gemessen an der SERIALISIERTEN Antwort, nicht am Typ:
// Genau dieses JSON bekommt der Helfer, denn /api/action hängt am Recht perform_actions,
// das auch die Helfer-Rolle trägt.
//
// Bis zum 07.09.2026 lief hier das volle Konto durch — mit E-Mail, Rolle und Anlagedatum.
// Seit Migration 125 ist ein Kollege ein LESER und läuft durch dieselbe Theken-Sicht wie
// ein Schüler (SchuelerKiosk); dieser Test hält fest, dass die Beschränkung dabei nicht
// verloren gegangen ist.
func TestActionResponse_KollegeOhneKontodaten(t *testing.T) {
	gebdatum := "1980-04-01"
	voll := &repository.Student{
		ID: "leser-1", BarcodeID: "L-0001", Vorname: "Frieda", Nachname: "Fachlehrerin",
		Art:          "lehrkraft",
		Geburtsdatum: &gebdatum,
		Strasse:      "Schulstraße", Hausnummer: "7", Plz: "61250", Ort: "Usingen",
		ElternEmail: "frieda.fachlehrerin@philipp-reis-schule.de",
	}
	resp := mapOmniboxResultToActionResponse(&service.OmniboxResult{
		Type: "student", Student: voll, Vorbesitzer: voll,
	})
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, verboten := range []string{
		"philipp-reis-schule.de", "eltern_email", "strasse", "Schulstraße", "hausnummer",
		"plz", "Usingen", "geburtsdatum", "1980-04-01",
	} {
		if strings.Contains(body, verboten) {
			t.Errorf("Theken-Antwort trägt Personendaten (%q): %s", verboten, body)
		}
	}
	for _, pflicht := range []string{
		`"vorname":"Frieda"`, `"nachname":"Fachlehrerin"`, `"barcode_id":"L-0001"`,
		`"id":"leser-1"`, `"art":"lehrkraft"`,
	} {
		if !strings.Contains(body, pflicht) {
			t.Errorf("Theken-Antwort verliert die Identität (%s): %s", pflicht, body)
		}
	}
}
