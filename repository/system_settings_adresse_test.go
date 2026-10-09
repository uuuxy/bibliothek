package repository

import "testing"

// Die öffentliche Adresse gilt nur, wenn außer Leerraum etwas dasteht; zurück kommt sie ohne
// den Leerraum am Rand.
func TestAdresseFuerLinks(t *testing.T) {
	zeiger := func(s string) *string { return &s }
	faelle := []struct {
		name    string
		adresse *string
		will    string
	}{
		{"nicht hinterlegt", nil, ""},
		{"geleert", zeiger(""), ""},
		{"nur Leerzeichen", zeiger("   "), ""},
		{"Tabulator und Zeilenende", zeiger("\t\n"), ""},
		{"Leerraum am Rand", zeiger("  https://bib.schule.de \n"), "https://bib.schule.de"},
		{"hinterlegt", zeiger("https://bib.schule.de"), "https://bib.schule.de"},
	}
	for _, f := range faelle {
		einstellungen := &SystemEinstellungen{OeffentlicheAdresse: f.adresse}
		if ist := einstellungen.AdresseFuerLinks(); ist != f.will {
			t.Errorf("%s: %q, erwartet %q", f.name, ist, f.will)
		}
	}
	var keine *SystemEinstellungen
	if ist := keine.AdresseFuerLinks(); ist != "" {
		t.Errorf("ohne Einstellungen: %q, erwartet keine Adresse", ist)
	}
}
