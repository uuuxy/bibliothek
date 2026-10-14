package auskunft

import (
	"strings"
	"testing"
)

// Die Speicherdauer nennt die eingestellte Karenzzeit, keine feste Zahl: Der Löschjob rechnet
// mit abgaenger_karenz_tage, und die Auskunft nennt dieselbe Frist.
func TestDSGVOVerarbeitungsangaben_KarenzAusEinstellung(t *testing.T) {
	va := dsgvoVerarbeitungsangaben(90, 730, 5, 24)
	if !strings.Contains(va.Speicherdauer, "Karenzzeit von 5 Tagen") {
		t.Errorf("Speicherdauer nennt die Karenz nicht: %q", va.Speicherdauer)
	}
	if strings.Contains(va.Speicherdauer, "360") {
		t.Errorf("Speicherdauer trägt noch die alte feste Frist: %q", va.Speicherdauer)
	}
	if va := dsgvoVerarbeitungsangaben(90, 730, 0, 24); !strings.Contains(va.Speicherdauer, "sofort nach dem letzten Vorgang") {
		t.Errorf("Karenz 0 muss sofort heißen: %q", va.Speicherdauer)
	}
}

// Die Herkunft nennt jeden Weg, auf dem Stammdaten in die Leserdatei kommen, auch die
// Übernahme aus dem bisherigen Bibliotheksprogramm (internal/littera/schreiber_personen.go
// legt Schüler und Lehrkräfte an).
func TestDSGVOHerkunft_NenntDieUebernahme(t *testing.T) {
	for art, va := range map[string]Verarbeitungsangaben{
		"schueler":  dsgvoVerarbeitungsangaben(90, 730, 90, 24),
		"lehrkraft": dsgvoVerarbeitungsangabenKollegium(FristWerte{LesehistorieTage: 90, LernmittelTage: 730, KarenzTage: 90, ProtokollMonate: 24, AnliegenTage: 365}),
	} {
		if !strings.Contains(va.Herkunft, "Übernahme aus dem bisherigen Bibliotheksprogramm") {
			t.Errorf("%s: Herkunft nennt die Übernahme nicht: %q", art, va.Herkunft)
		}
	}
}
