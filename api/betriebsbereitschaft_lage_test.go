package api

import (
	"testing"

	"bibliothek/internal/bereitschaft"
	"bibliothek/repository"
)

// Die Selbstprüfung bekommt die öffentliche Adresse so, wie die Links sie benutzen: ohne
// Leerraum am Rand, und leer, wenn nur Leerraum dasteht.
func TestEinstellungenInDieLage_OeffentlicheAdresse(t *testing.T) {
	zeiger := func(s string) *string { return &s }
	faelle := []struct {
		name    string
		adresse *string
		will    string
	}{
		{"nicht hinterlegt", nil, ""},
		{"nur Leerzeichen", zeiger("   "), ""},
		{"Leerraum am Rand", zeiger(" https://bib.schule.de "), "https://bib.schule.de"},
	}
	for _, f := range faelle {
		var lage bereitschaft.Lage
		einstellungenInDieLage(&lage, &repository.SystemEinstellungen{OeffentlicheAdresse: f.adresse})
		if lage.OeffentlicheAdresse != f.will {
			t.Errorf("%s: die Lage trägt %q, erwartet %q", f.name, lage.OeffentlicheAdresse, f.will)
		}
	}
}
