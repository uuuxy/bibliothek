package pdf

// Der Mahnbrief, geprüft am gedruckten Blatt: ein Brief nach DIN 5008 für das Fensterkuvert.
// Der Erzeuger kennt weder Abfragen noch Einstellungen. Den Weg über die Tür prüft
// api/mahnwesen_bulk_frist_pg_test.go, das Füllen der Eingabe api/mahnwesen_bulk_test.go.

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/schulzeit"
)

func renderMahnbrief(t *testing.T, e MahnbriefEmpfaenger, betreff, text string) string {
	t.Helper()
	roh, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{e},
		MahnbriefVorlage{Betreff: betreff, Text: text, Absender: "Testschule"})
	if err != nil {
		t.Fatalf("PDF erzeugen: %v", err)
	}
	return blattText(t, roh)
}

func renderElternMahnbrief(t *testing.T, e MahnbriefEmpfaenger) string {
	t.Helper()
	return renderMahnbrief(t, e, "Mahnung", "Liebe Eltern von {{.Vorname}} {{.Nachname}},\nbitte zurückgeben:\n{{.BuchListe}}")
}

func testMahnbriefEmpfaenger() MahnbriefEmpfaenger {
	return MahnbriefEmpfaenger{
		Vorname: "Mia", Nachname: "Musterkind",
		Buecher: []MahnbriefBuch{{
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
	e.Buecher = append(e.Buecher, MahnbriefBuch{
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
	if !strings.Contains(text, "(keine Adresse hinterlegt)") {
		t.Error("Brief ohne Anschrift muss '(keine Adresse hinterlegt)' ins Fensterfeld drucken — " +
			"eine leere Zeile ist von einem Druckfehler nicht zu unterscheiden")
	}
}

// Was der Brief setzt, in der Folge von oben nach unten: Absenderzeile, Anschrift, Datum,
// Betreff und Text mit den Namen aus der Vorlage, die Tabelle mit ihren Spalten, der Rest des
// Texts. Die Tage der Tabelle sind Kalendertage der Schule: 23:30 Uhr UTC ist dort der Folgetag.
func TestMahnbrief_SetztVorlageAnschriftUndTabelleInIhrerFolge(t *testing.T) {
	roh, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{{
		Vorname: "Mia", Nachname: "Musterkind", Strasse: "Blumenweg", Hausnummer: "7", PLZ: "61169", Ort: "Friedberg",
		Buecher: []MahnbriefBuch{
			{Titel: "Gregs Tagebuch", Barcode: "B-00042", TageUeberfaellig: 9,
				AusgeliehenAm: time.Date(2026, time.March, 3, 23, 30, 0, 0, time.UTC), Frist: time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)},
			{Titel: "Atlas", Barcode: "B-00815", TageUeberfaellig: 29,
				AusgeliehenAm: time.Date(2026, time.February, 10, 9, 0, 0, 0, time.UTC), Frist: time.Date(2026, time.April, 14, 23, 30, 0, 0, time.UTC)},
		},
	}}, MahnbriefVorlage{
		Betreff:  "Mahnung für {{.Vorname}} {{.Nachname}}",
		Text:     "Liebe Eltern von {{.Vorname}} {{.Nachname}}, Frist war {{.Frist}}.\n{{.BuchListe}}\nMit Gruß",
		Absender: "Testschule · Schulweg 1 · 61381 Friedrichsdorf",
	})
	if err != nil {
		t.Fatalf("Mahnbrief drucken: %v", err)
	}

	gesetzt := strings.Join(pdftest.TexteInReihenfolge(t, roh), " | ")
	folge := strings.Join([]string{
		"Testschule · Schulweg 1 · 61381 Friedrichsdorf",
		"Eltern von Mia Musterkind", "Blumenweg 7", "61169 Friedberg",
		"Datum: " + schulzeit.Jetzt().Format(dateFormatDE),
		"Mahnung für Mia Musterkind",
		"Liebe Eltern von Mia Musterkind, Frist war 15.04.2026.",
		"Titel", "Barcode", "Ausgeliehen", "Tage überfällig",
		"Gregs Tagebuch", "B-00042", "04.03.2026", "9",
		"Atlas", "B-00815", "10.02.2026", "29",
		"Mit Gruß",
	}, " | ")
	if gesetzt != folge {
		t.Errorf("gesetzt:\n%s\nerwartet:\n%s", gesetzt, folge)
	}
}

// Je Schüler ein Brief auf einer eigenen Seite, mit seinen Büchern und keinem fremden.
func TestMahnbrief_JeSchuelerEinBriefAufEigenerSeite(t *testing.T) {
	erster := testMahnbriefEmpfaenger()
	zweiter := MahnbriefEmpfaenger{Vorname: "Ben", Nachname: "Birne", Buecher: []MahnbriefBuch{
		{Titel: "Atlas", Barcode: "BC-2", AusgeliehenAm: time.Now(), Frist: time.Now(), TageUeberfaellig: 3}}}
	roh, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{erster, zweiter},
		MahnbriefVorlage{Betreff: "Mahnung", Text: "Eltern von {{.Vorname}}:\n{{.BuchListe}}", Absender: "Testschule"})
	if err != nil {
		t.Fatalf("Mahnbriefe drucken: %v", err)
	}

	seiten := pdftest.TexteJeSeite(t, roh)
	if len(seiten) != 2 {
		t.Fatalf("%d Seiten, erwartet 2", len(seiten))
	}
	pruefeBlatt(t, strings.Join(seiten[0], "\n"), []string{"Eltern von Mia Musterkind", "Testband", "BC-1"}, []string{"Birne", "Atlas", "BC-2"})
	pruefeBlatt(t, strings.Join(seiten[1], "\n"), []string{"Eltern von Ben Birne", "Atlas", "BC-2"}, []string{"Musterkind", "Testband", "BC-1"})
}

// Der Titel steht gekürzt mit Auslassungszeichen da, sobald er gedruckt breiter wäre als seine
// Spalte: 60 breite Buchstaben passen nicht, 80 schmale schon. Der Strichcode steht als Bild
// über seiner Nummer; ohne Barcode bleibt die Zelle leer.
func TestMahnbrief_TitelBleibtInSeinerSpalteUndStrichcodeAlsBild(t *testing.T) {
	breit, schmal := strings.Repeat("W", 60), strings.Repeat("i", 80)
	e := testMahnbriefEmpfaenger()
	e.Buecher[0].Titel = breit
	e.Buecher = append(e.Buecher, e.Buecher[0])
	e.Buecher[1].Titel, e.Buecher[1].Barcode = schmal, "BC-2"
	mit, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{e}, MahnbriefVorlage{Text: "{{.BuchListe}}"})
	if err != nil {
		t.Fatalf("Mahnbrief drucken: %v", err)
	}
	texte := pdftest.TexteInReihenfolge(t, mit)
	for stelle, text := range texte {
		// Vor der Nummer des Strichcodes steht der Titel der Zeile.
		switch text {
		case "BC-1":
			pdftest.InSpalte(t, texte[stelle-1], breit, 10, 73)
			if texte[stelle-1] == breit {
				t.Errorf("60 breite Buchstaben stehen ungekürzt in einer Spalte von 75 mm")
			}
		case "BC-2":
			if texte[stelle-1] != schmal {
				t.Errorf("80 schmale Buchstaben passen in die Spalte, gedruckt ist %q", texte[stelle-1])
			}
		}
	}
	if n := bytes.Count(mit, []byte("/Subtype /Image")); n != 2 {
		t.Errorf("Brief mit zwei Barcodes trägt %d Bilder, erwartet 2", n)
	}

	e.Buecher = e.Buecher[:1]
	e.Buecher[0].Barcode = ""
	ohne, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{e}, MahnbriefVorlage{Text: "{{.BuchListe}}"})
	if err != nil {
		t.Fatalf("Mahnbrief drucken: %v", err)
	}
	if bytes.Contains(ohne, []byte("/Subtype /Image")) {
		t.Error("Brief ohne Barcode trägt ein Bild")
	}
}

// Fehlt nur ein Teil der Anschrift, steht der andere da und kein Vermerk.
func TestMahnbrief_HalbeAnschriftOhneVermerk(t *testing.T) {
	nurOrt := testMahnbriefEmpfaenger()
	nurOrt.PLZ, nurOrt.Ort = "61169", "Friedberg"
	pruefeBlatt(t, renderElternMahnbrief(t, nurOrt), []string{"61169 Friedberg"}, []string{"keine Adresse hinterlegt"})

	nurStrasse := testMahnbriefEmpfaenger()
	nurStrasse.Strasse = "Blumenweg"
	pruefeBlatt(t, renderElternMahnbrief(t, nurStrasse), []string{"Blumenweg"}, []string{"keine Adresse hinterlegt"})
}

// langerMahnbrief baut einen Brief mit n Büchern, jedes mit eigenen Texten.
func langerMahnbrief(vorname string, n int) MahnbriefEmpfaenger {
	e := MahnbriefEmpfaenger{Vorname: vorname, Nachname: "Musterkind", Strasse: "Blumenweg", Hausnummer: "7", PLZ: "61169", Ort: "Friedberg"}
	for i := 1; i <= n; i++ {
		e.Buecher = append(e.Buecher, MahnbriefBuch{
			Titel: fmt.Sprintf("Titel %02d", i), Barcode: fmt.Sprintf("B-9%03d", i),
			AusgeliehenAm:    time.Date(2026, time.January, i, 12, 0, 0, 0, time.UTC),
			Frist:            time.Date(2026, time.March, i, 12, 0, 0, 0, time.UTC),
			TageUeberfaellig: 100 + i,
		})
	}
	return e
}

// mahnbriefZeilenJeSeite prüft jede Zeile der Tabelle: Nummer, Tag und Tage stehen auf der Seite
// ihres Titels und in seiner Höhe, dazu genau ein Bild. Die Spaltenköpfe stehen auf jeder Seite
// mit Zeilen und auf keiner ohne. Geliefert wird die Zahl der Zeilen je Seite.
func mahnbriefZeilenJeSeite(t *testing.T, seiten []seiteMitOrten, anzahl int) []int {
	t.Helper()
	// Eine Zeile ist 15 mm hoch; halb so viel über und unter dem Titel gehört zu ihr.
	const halbeZeile = 15.0 / 2 / 25.4 * 72
	var jeSeite []int
	for nr, seite := range seiten {
		zeilen := 0
		for i := 1; i <= anzahl; i++ {
			titelHoehe, da := seite.texte[fmt.Sprintf("Titel %02d", i)]
			if !da {
				continue
			}
			zeilen++
			for _, teil := range []string{fmt.Sprintf("B-9%03d", i), fmt.Sprintf("%02d.01.2026", i), fmt.Sprint(100 + i)} {
				hoehe, da := seite.texte[teil]
				if !da {
					t.Errorf("%d Bücher, Seite %d: %q steht nicht auf der Seite seines Titels", anzahl, nr+1, teil)
				} else if math.Abs(hoehe-titelHoehe) > halbeZeile {
					t.Errorf("%d Bücher, Seite %d: %q steht %.0f Punkt neben seinem Titel", anzahl, nr+1, teil, hoehe-titelHoehe)
				}
			}
			if bilder := seite.bilderInDerZeile(titelHoehe, halbeZeile); bilder != 1 {
				t.Errorf("%d Bücher, Seite %d, Zeile %d: %d Bilder in der Zeile, erwartet den Strichcode", anzahl, nr+1, i, bilder)
			}
		}
		if _, koepfe := seite.texte["Ausgeliehen"]; koepfe != (zeilen > 0) {
			t.Errorf("%d Bücher, Seite %d: Spaltenköpfe = %v bei %d Zeilen", anzahl, nr+1, koepfe, zeilen)
		}
		if len(seite.bildMitten) != zeilen {
			t.Errorf("%d Bücher, Seite %d: %d Bilder bei %d Zeilen", anzahl, nr+1, len(seite.bildMitten), zeilen)
		}
		jeSeite = append(jeSeite, zeilen)
	}
	return jeSeite
}

// Eine Zeile setzt Strichcode und Nummer an feste Stellen. Bricht gofpdf mitten in ihr um,
// stehen ihre Teile auf drei Seiten; Schulbücher eines Jahres sind schnell mehr als acht.
func TestMahnbrief_LangeTabelleHaeltJedeZeileBeisammen(t *testing.T) {
	vorlage := MahnbriefVorlage{Betreff: "Mahnung", Text: "Zeile eins\n{{.BuchListe}}\nSchluss-Zeile", Absender: "Testschule"}
	for anzahl := 1; anzahl <= 30; anzahl++ {
		roh, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{langerMahnbrief("Mia", anzahl)}, vorlage)
		if err != nil {
			t.Fatalf("%d Bücher: %v", anzahl, err)
		}
		seiten := seitenMitOrten(t, roh)
		jeSeite := mahnbriefZeilenJeSeite(t, seiten, anzahl)
		gedruckt := 0
		for _, zeilen := range jeSeite {
			gedruckt += zeilen
		}
		if gedruckt != anzahl {
			t.Errorf("%d Bücher: %d Zeilen gedruckt", anzahl, gedruckt)
		}
		if anzahl == 30 && fmt.Sprint(jeSeite) != "[9 16 5]" {
			t.Errorf("30 Bücher: Zeilen je Seite %v, erwartet [9 16 5]", jeSeite)
		}
	}
}

// Ein Druck trägt viele Briefe hintereinander. Jede Folgeseite nennt, zu wessen Brief sie
// gehört, ob die Tabelle sie füllt oder der Text; die erste Seite eines Briefs nennt es nicht.
func TestMahnbrief_FolgeseitenNennenDenEmpfaenger(t *testing.T) {
	vorlage := MahnbriefVorlage{Betreff: "Mahnung", Text: "Zeile eins\n{{.BuchListe}}\n" + strings.Repeat("Schluss-Zeile\n", 45), Absender: "Testschule"}
	roh, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{langerMahnbrief("Mia", 12), langerMahnbrief("Ben", 2)}, vorlage)
	if err != nil {
		t.Fatalf("Mahnbriefe drucken: %v", err)
	}

	var folge []string
	for _, seite := range pdftest.TexteJeSeite(t, roh) {
		text := strings.Join(seite, "\n")
		switch {
		case strings.Contains(text, "Eltern von Mia Musterkind"):
			folge = append(folge, "Brief Mia")
		case strings.Contains(text, "Eltern von Ben Musterkind"):
			folge = append(folge, "Brief Ben")
		case strings.Contains(text, "Fortsetzung: Mia Musterkind"):
			folge = append(folge, "Fortsetzung Mia")
		case strings.Contains(text, "Fortsetzung: Ben Musterkind"):
			folge = append(folge, "Fortsetzung Ben")
		default:
			folge = append(folge, "ohne Namen")
		}
		if strings.Contains(text, "Eltern von") && strings.Contains(text, "Fortsetzung:") {
			t.Errorf("die erste Seite eines Briefs trägt die Zeile der Folgeseite:\n%s", text)
		}
	}
	// Mia: Brief, die Tabelle läuft auf Seite 2 weiter, der Schluss des Texts auf Seite 3.
	// Ben: Brief, der Schluss des Texts auf einer Folgeseite.
	if erwartet := "Brief Mia | Fortsetzung Mia | Fortsetzung Mia | Brief Ben | Fortsetzung Ben"; strings.Join(folge, " | ") != erwartet {
		t.Errorf("Seiten des Drucks: %s\nerwartet:          %s", strings.Join(folge, " | "), erwartet)
	}
}

// Füllt der Text die Seite so weit, dass nur noch die Köpfe der Tabelle passen, beginnt die
// Tabelle auf der nächsten Seite: Köpfe ohne Zeile darunter nennen Spalten ohne Inhalt.
func TestMahnbrief_KoepfeStehenNichtAlleinAmSeitenfuss(t *testing.T) {
	vorlage := MahnbriefVorlage{Betreff: "Mahnung", Text: strings.Repeat("Zeile\n", 24) + "{{.BuchListe}}", Absender: "Testschule"}
	roh, err := GenerateMahnbriefePDF([]MahnbriefEmpfaenger{langerMahnbrief("Mia", 3)}, vorlage)
	if err != nil {
		t.Fatalf("Mahnbrief drucken: %v", err)
	}
	if jeSeite := mahnbriefZeilenJeSeite(t, seitenMitOrten(t, roh), 3); fmt.Sprint(jeSeite) != "[0 3]" {
		t.Errorf("Zeilen je Seite: %v, erwartet [0 3]", jeSeite)
	}
}
