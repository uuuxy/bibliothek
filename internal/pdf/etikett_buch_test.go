package pdf

import (
	"bytes"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
)

func TestErzeugeBuchEtiketten(t *testing.T) {
	etiketten := []BuchEtikett{
		{Schulname: "Testbibliothek", BarcodeID: "B-10001", Titel: "Test Buch 1", Signatur: "LMF-Deutsch 5", Eigentumsvermerk: "Eigentum des Landes Hessen"},
		{Schulname: "Testbibliothek", BarcodeID: "B-10002", Titel: "Test Buch 2", Eigentumsvermerk: "Eigentum des Landes Hessen"},
	}

	dokument, err := ErzeugeBuchEtiketten("zweckform_l4760", 1, false, etiketten)
	if err != nil {
		t.Fatalf("Failed to generate label PDF: %v", err)
	}
	var buf bytes.Buffer
	if err := dokument.Output(&buf); err != nil {
		t.Fatalf("Failed to output label PDF: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("Generated label PDF is empty")
	}
}

// Lieferanten, die selbst etikettieren, bekommen zum kleinen das große Lernmittel-Etikett
// mitgeschickt und wählen selbst, welches sie drucken.
func TestErzeugeLernmittelEtiketten(t *testing.T) {
	etiketten := []BuchEtikett{
		{Schulname: "Testbibliothek", BarcodeID: "B-10001", Titel: "Test Buch 1", AnschaffungsJahr: "2024", Signatur: "LMF-Deutsch 5", Eigentumsvermerk: "Eigentum des Landes Hessen"},
		{Schulname: "Testbibliothek", BarcodeID: "B-10002", Titel: "Test Buch 2", Eigentumsvermerk: "Eigentum des Landes Hessen"},
	}

	dokument, err := ErzeugeLernmittelEtiketten(etiketten)
	if err != nil {
		t.Fatalf("Failed to generate Lernmittel-Etikett PDF: %v", err)
	}
	if len(dokument) == 0 {
		t.Error("Generated Lernmittel-Etikett PDF is empty")
	}
}

func TestZweiteZeile(t *testing.T) {
	faelle := []struct {
		jahr, signatur, want string
	}{
		{"2016", "LMF-Deutsch 5", "Ansch.J. 2016 · LMF-Deutsch 5"},
		{"2016", "", "Ansch.J. 2016"},
		{"", "LMF-Deutsch 5", "LMF-Deutsch 5"},
		{"", "", ""},
	}
	for _, c := range faelle {
		if got := zweiteZeile(c.jahr, c.signatur); got != c.want {
			t.Errorf("zweiteZeile(%q, %q) = %q, erwartet %q", c.jahr, c.signatur, got, c.want)
		}
	}
}

// gedruckteZeilen liefert die gedruckten Textstücke, die das Wort tragen, in der Reihenfolge
// des Blatts.
func gedruckteZeilen(t *testing.T, roh []byte, wort string) []string {
	t.Helper()
	var zeilen []string
	for _, text := range pdftest.TexteInReihenfolge(t, roh) {
		if strings.Contains(text, wort) {
			zeilen = append(zeilen, text)
		}
	}
	return zeilen
}

// Der Erzeuger kennt weder Einstellungen noch den Topf: Er druckt Schulname und Vermerk, die
// das Etikett trägt, und ohne Vermerk keinen, mit Strichcode wie mit QR-Code. Auf dem kleinen
// Format unter 30 mm hat die Zeile des Vermerks keinen Platz.
func TestBuchEtikett_DrucktSchulnameUndVermerkDesEtiketts(t *testing.T) {
	etiketten := []BuchEtikett{
		{Schulname: "Schule Eins", BarcodeID: "B-1", Titel: "Eins", Eigentumsvermerk: "Vermerk Eins"},
		{Schulname: "Schule Zwei", BarcodeID: "B-2", Titel: "Zwei", Eigentumsvermerk: "Vermerk Zwei"},
		{Schulname: "Schule Drei", BarcodeID: "B-3", Titel: "Drei"},
	}
	gross, err := ErzeugeLernmittelEtiketten(etiketten)
	if err != nil {
		t.Fatalf("großes Etikett: %v", err)
	}

	schulen := []string{"Schule Eins", "Schule Zwei", "Schule Drei"}
	vermerke := []string{"Vermerk Eins", "Vermerk Zwei"}
	bogen := map[string][]byte{
		"Bogen ab 30 mm mit Strichcode": buchBogen(t, "zweckform_l4760", false, etiketten...),
		"Bogen ab 30 mm mit QR-Code":    buchBogen(t, "zweckform_l4760", true, etiketten...),
		"großes Etikett":                gross,
	}
	for name, roh := range bogen {
		if ist := gedruckteZeilen(t, roh, "Schule"); strings.Join(ist, "|") != strings.Join(schulen, "|") {
			t.Errorf("%s: gedruckte Schulnamen %q, erwartet %q", name, ist, schulen)
		}
		if ist := gedruckteZeilen(t, roh, "Vermerk"); strings.Join(ist, "|") != strings.Join(vermerke, "|") {
			t.Errorf("%s: gedruckte Vermerke %q, erwartet %q", name, ist, vermerke)
		}
	}
	for _, alsQR := range []bool{false, true} {
		klein := buchBogen(t, "standard_52", alsQR, etiketten...)
		if ist := gedruckteZeilen(t, klein, "Schule"); strings.Join(ist, "|") != strings.Join(schulen, "|") {
			t.Errorf("kleines Format (QR %v): gedruckte Schulnamen %q, erwartet %q", alsQR, ist, schulen)
		}
		if ist := gedruckteZeilen(t, klein, "Vermerk"); len(ist) != 0 {
			t.Errorf("kleines Format (QR %v): gedruckte Vermerke %q, erwartet keinen", alsQR, ist)
		}
	}
}
