package pdf

import (
	"strings"
	"testing"
)

// Auf einem Blatt mit genau einem Buch stand ein Satz, der nur für mehrere gebaut war: „Bitte
// gib die folgenden 1 Medium … ab", „Summe …: 1 Exemplare", „1 weitere Exemplare sind
// ausgesondert", „(1 Bestellungen, …)". Die Mahnliste bekommt das Kind in die Hand, den
// Bestellbericht legt die Schule vor.
func TestAnzahlText_EinzahlUndMehrzahl(t *testing.T) {
	for n, soll := range map[int]string{0: "0 Exemplare", 1: "1 Exemplar", 2: "2 Exemplare", 30: "30 Exemplare"} {
		if ist := exemplarText(n); ist != soll {
			t.Errorf("exemplarText(%d) = %q, erwartet %q", n, ist, soll)
		}
	}
	if ist := anzahlText(1, "Bestellung", "Bestellungen"); ist != "1 Bestellung" {
		t.Errorf("anzahlText(1, Bestellung) = %q", ist)
	}
}

func TestSaetzeFuerEinEinzelnesBuch(t *testing.T) {
	for name, f := range map[string]struct {
		satz      func(int) string
		eins      string
		mehrere   string
		nieInEins []string
	}{
		"Mahnliste": {aufforderungZurAbgabe,
			"Bitte gib das folgende Medium umgehend in der Schulbibliothek ab.",
			"Bitte gib die folgenden 3 Medien umgehend in der Schulbibliothek ab.",
			[]string{"folgenden", "Medien"}},
		"Abgangsbuch ohne Zeitpunkt": {hinweisOhneZeitpunkt,
			"Hinweis: 1 weiteres Exemplar ist ausgesondert",
			"Hinweis: 3 weitere Exemplare sind ausgesondert",
			[]string{"Exemplare", "Sie wurden", "können", "sie stehen"}},
		"Abgangsbuch aus dem Katalog gelöscht": {hinweisAusKatalogGeloescht,
			"Hinweis: 1 Exemplar wurde in diesem Zeitraum aus dem Katalog gelöscht",
			"Hinweis: 3 Exemplare wurden in diesem Zeitraum aus dem Katalog gelöscht",
			[]string{"Exemplare", "mit ihnen", "sie stehen", "Ihre Nummern"}},
	} {
		if ist := f.satz(1); !strings.HasPrefix(ist, f.eins) {
			t.Errorf("%s für ein Buch: %q, erwartet den Anfang %q", name, ist, f.eins)
		}
		for _, wort := range f.nieInEins {
			if strings.Contains(f.satz(1), wort) {
				t.Errorf("%s für ein Buch enthält %q: %q", name, wort, f.satz(1))
			}
		}
		if ist := f.satz(3); !strings.HasPrefix(ist, f.mehrere) {
			t.Errorf("%s für drei Bücher: %q, erwartet den Anfang %q", name, ist, f.mehrere)
		}
	}
}
