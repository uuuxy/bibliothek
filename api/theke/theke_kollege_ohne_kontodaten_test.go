package theke

import (
	"encoding/json"
	"strings"
	"testing"

	"bibliothek/internal/service"
	"bibliothek/repository"
)

// Die Theken-Antwort trägt von einem Kollegen nur Identität, keine Anschrift, keine
// Kontaktdaten, kein Geburtsdatum. Gemessen an der serialisierten Antwort, nicht am Typ:
// Genau dieses JSON bekommt der Helfer, denn /api/action hängt am Recht perform_actions, das
// auch die Helfer-Rolle trägt. Ein Kollege ist ein Leser (Migration 125) und läuft durch
// dieselbe Theken-Sicht wie ein Schüler (thekenLeser); das volle Konto mit E-Mail, Rolle und
// Anlagedatum gehört nicht hinein.
func TestEingabeAntwort_KollegeOhneKontodaten(t *testing.T) {
	gebdatum := "1980-04-01"
	voll := &repository.Leser{
		ID: "leser-1", BarcodeID: "L-0001", Vorname: "Frieda", Nachname: "Fachlehrerin",
		Art:          "lehrkraft",
		Geburtsdatum: &gebdatum,
		Strasse:      "Schulstraße", Hausnummer: "7", Plz: "61250", Ort: "Usingen",
		ElternEmail: "frieda.fachlehrerin@philipp-reis-schule.de",
	}
	resp := zurEingabeAntwort(&service.OmniboxErgebnis{
		Art: "student", Leser: voll, Vorbesitzer: voll,
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
