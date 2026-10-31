package theke

import (
	"testing"

	"bibliothek/internal/service"
)

// Nur eine gebuchte Fremdrückgabe mit gültiger Ausleih-Kennung lässt den Eintrag „ausleihen" noch
// zu. Jede andere Buchung gilt als vollständig; der Eintrag buchte das Buch sonst ein zweites Mal.
func TestNurFremdrueckgabeVorAusleihe(t *testing.T) {
	gueltig := "5e1d0c8a-5b0e-4c55-9d0f-3a8b1c2d4e01"
	falsch := "urn:uuid:5e1d0c8a-5b0e-4c55-9d0f-3a8b1c2d4e01"
	faelle := []struct {
		name    string
		gebucht eingabeAntwort
		absicht string
		soll    bool
	}{
		{"Fremdrückgabe, Eintrag will ausleihen", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true, AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, true},
		{"gewöhnliche Rückgabe", eingabeAntwort{Art: "rueckgabe", AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, false},
		{"online schon ausgeliehen", eingabeAntwort{Art: "ausleihe", AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, false},
		{"Ausleihe mit Merkmal der Fremdrückgabe", eingabeAntwort{Art: "ausleihe", Fremdrueckgabe: true, AusleiheID: &gueltig}, service.NachbuchAbsichtAusleihe, false},
		{"Eintrag will zurückgeben", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true, AusleiheID: &gueltig}, "rueckgabe", false},
		{"ohne Kennung", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true}, service.NachbuchAbsichtAusleihe, false},
		{"Kennung nicht in strenger Form", eingabeAntwort{Art: "rueckgabe", Fremdrueckgabe: true, AusleiheID: &falsch}, service.NachbuchAbsichtAusleihe, false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if ist := nurFremdrueckgabeVorAusleihe(f.gebucht, f.absicht); ist != f.soll {
				t.Errorf("= %v, erwartet %v", ist, f.soll)
			}
		})
	}
}
