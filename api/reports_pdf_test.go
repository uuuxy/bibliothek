package api

// Tests am fertigen PDF-Inhaltsstrom (Technik wie die Etiketten-Gates): Der Mahnbrief ist
// ein Brief nach DIN 5008 für das Fensterkuvert. Den Weg über die Tür prüft
// mahnwesen_bulk_frist_pg_test.go; hier stehen die Fälle des Blatts selbst.

import (
	"strings"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

func renderMahnbrief(t *testing.T, e repository.MahnbriefEmpfaenger, betreff, text string) string {
	t.Helper()
	roh, err := erzeugeMahnbriefe([]repository.MahnbriefEmpfaenger{e},
		mahnbriefVorlage{Betreff: betreff, Text: text, Absender: "Testschule"})
	if err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	return pdfText(t, roh)
}

func renderElternMahnbrief(t *testing.T, e repository.MahnbriefEmpfaenger) string {
	t.Helper()
	return renderMahnbrief(t, e, "Mahnung", "Liebe Eltern von {{.Vorname}} {{.Nachname}},\nbitte zurückgeben:\n{{.BuchListe}}")
}

func testMahnbriefEmpfaenger() repository.MahnbriefEmpfaenger {
	return repository.MahnbriefEmpfaenger{
		Vorname: "Mia", Nachname: "Musterkind",
		Buecher: []repository.MahnbriefBuch{{
			Titel: "Testband", Barcode: "BC-1",
			AusgeliehenAm:    time.Now().AddDate(0, -2, 0),
			Frist:            time.Now().AddDate(0, -1, 0),
			TageUeberfaellig: 30,
		}},
	}
}

func TestElternMahnbriefDrucktAnschriftInsFensterfeld(t *testing.T) {
	e := testMahnbriefEmpfaenger()
	e.Strasse, e.Hausnummer = "Blumenweg", "7"
	e.PLZ, e.Ort = "61169", "Friedberg"

	text := renderElternMahnbrief(t, e)
	for _, soll := range []string{"Blumenweg 7", "61169 Friedberg", "Eltern von Mia Musterkind"} {
		if !strings.Contains(text, soll) {
			t.Errorf("Brief ohne %q — das Fensterkuvert bliebe leer", soll)
		}
	}
	if strings.Contains(text, "Adresse unbekannt") || strings.Contains(text, "keine Adresse hinterlegt") {
		t.Error("Brief trägt trotz vollständiger Anschrift einen Fehlt-Vermerk")
	}
}

// {{.Frist}} trägt die älteste Rückgabefrist der gemahnten Bücher, nicht das Druckdatum:
// Der Text der Vorlage nennt sie über einer Tabelle mit den Tagen über der Frist.
func TestElternMahnbriefFristIstDieAeltesteRueckgabefrist(t *testing.T) {
	e := testMahnbriefEmpfaenger()
	aeltere := time.Now().AddDate(0, 0, -40)
	e.Buecher = append(e.Buecher, repository.MahnbriefBuch{
		Titel: "Zweitband", Barcode: "BC-2",
		AusgeliehenAm:    time.Now().AddDate(0, -3, 0),
		Frist:            aeltere,
		TageUeberfaellig: 40,
	})

	text := renderMahnbrief(t, e, "Mahnung", "Frist war {{.Frist}} Ende\n{{.BuchListe}}")

	// Der Brief nennt den Kalendertag der Schule.
	if soll := "Frist war " + aeltere.In(schulzeit.Zone()).Format(dateFormatDE) + " Ende"; !strings.Contains(text, soll) {
		t.Errorf("Brief nennt nicht die älteste Rückgabefrist: %q fehlt", soll)
	}
	if falsch := "Frist war " + schulzeit.Jetzt().Format(dateFormatDE); strings.Contains(text, falsch) {
		t.Errorf("Brief füllt {{.Frist}} mit dem Druckdatum (%q)", falsch)
	}
}

// Die Vorlage ist freier Text der Schule. {{.BuchListe}} im Betreff stünde wörtlich in der
// Betreffzeile, und bei zwei Vorkommen im Text verschwände alles nach dem zweiten samt
// Grußformel.
func TestElternMahnbriefUeberlebtSchiefePlatzhalter(t *testing.T) {
	text := renderMahnbrief(t, testMahnbriefEmpfaenger(),
		"Mahnung {{.BuchListe}}",
		"Anfang {{.BuchListe}} Mitte {{.BuchListe}} Grussformel-Ende")

	// Klammern stehen im PDF-Strom escaped — auf den Kern ohne Klammern prüfen.
	if strings.Contains(text, "{.BuchListe}") {
		t.Error("{{.BuchListe}} steht wörtlich im Brief (Betreff oder zweites Vorkommen)")
	}
	if !strings.Contains(text, "Grussformel-Ende") {
		t.Error("Text nach dem zweiten {{.BuchListe}} wurde verschluckt — die Grußformel fehlt")
	}
	if !strings.Contains(text, "Mitte") {
		t.Error("Text zwischen den Vorkommen fehlt")
	}
}

func TestElternMahnbriefOhneAnschriftSagtEsAusdruecklich(t *testing.T) {
	text := renderElternMahnbrief(t, testMahnbriefEmpfaenger())
	// Ohne Klammern gesucht: Im PDF-Inhaltsstrom stehen Klammern escaped
	// (`\(…\)`), der Wortlaut dazwischen bleibt unverändert.
	if !strings.Contains(text, "keine Adresse hinterlegt") {
		t.Error("Brief ohne Anschrift muss '(keine Adresse hinterlegt)' ins Fensterfeld drucken — " +
			"eine leere Zeile ist von einem Druckfehler nicht zu unterscheiden")
	}
}
