package api

import (
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
)

// Was hängt an der Bestellmail — und was ausdrücklich NICHT?
//
// Der Hauptlieferant bekam bis zum 06.08.2026 beide fertigen Etikettenbögen ins Postfach
// UND einen Link, über den er dieselben Bögen abrufen und dabei bestätigen sollte. Wer die
// Anlage druckt, klickt den Link nie: Die Bücher kommen beklebt an, die Bestellhistorie
// wartet weiter auf eine Bestätigung, und niemand merkt, dass das Warten sinnlos ist.
//
// Geprüft wird an den Dateinamen der fertigen Anlagen, nicht an Zwischenzuständen — das
// ist das, was im Postfach des Händlers ankommt.

func testEtiketten() []BarcodeLabelDetail {
	return []BarcodeLabelDetail{
		{BarcodeID: "B-10001", Titel: "Mathematik 5", Autor: "Autor", Signatur: "LMF-Mathe 5"},
		{BarcodeID: "B-10002", Titel: "Mathematik 5", Autor: "Autor", Signatur: "LMF-Mathe 5"},
	}
}

func testBestellMail() BestellMail {
	return BestellMail{
		Empfaenger:       "haendler@example.invalid",
		Betreff:          "Buchbestellung",
		Text:             "Sehr geehrte Damen und Herren,",
		Positionen:       []OrderedItem{{Titel: "Mathematik 5", Autor: "Autor", ISBN: "978", Menge: 2}},
		Etiketten:        testEtiketten(),
		MitVorabBarcodes: true,
		Schule:           pdf.SchuleInfo{Name: "Testschule"},
		Mittel:           mitteltopf.Land,
	}
}

// anhangNamen liefert die Dateinamen ohne das angehängte Datum, damit die Erwartung nicht
// morgen eine andere ist.
func anhangNamen(t *testing.T, m BestellMail) []string {
	t.Helper()
	anhaenge, err := bestellAnhaenge(m)
	if err != nil {
		t.Fatalf("Anhänge erzeugen: %v", err)
	}
	namen := make([]string, 0, len(anhaenge))
	for _, a := range anhaenge {
		if len(a.Data) == 0 {
			t.Errorf("Anlage %q ist leer — der Händler öffnet eine kaputte Datei", a.Name)
		}
		namen = append(namen, a.Name[:strings.LastIndex(a.Name, "_")])
	}
	return namen
}

func TestBestellAnhaenge_MitLinkKeineEtikettenbogen(t *testing.T) {
	m := testBestellMail()
	m.IstHauptlieferant = true
	m.MitBestaetigungsLink = true

	namen := anhangNamen(t, m)

	for _, name := range namen {
		if strings.HasPrefix(name, "etiketten_") {
			t.Errorf("%q hängt an der Mail, obwohl die Etiketten hinter dem Link liegen — "+
				"der Händler druckt aus dem Postfach und bestätigt nie", name)
		}
	}
	pruefeEnthalten(t, namen, "bestellanschreiben", "barcode_mapping")
}

// Rückfallebene: Ohne Link (keine öffentliche Adresse hinterlegt) MUSS der Bogen beiliegen
// — sonst hat der Händler nichts zum Bekleben, und die Exemplare gelten trotzdem schon als
// etikettiert (siehe order_service.go, beklebtGeliefert).
func TestBestellAnhaenge_OhneLinkLiegenDieBoegenBei(t *testing.T) {
	m := testBestellMail()
	m.IstHauptlieferant = true
	m.MitBestaetigungsLink = false

	pruefeEnthalten(t, anhangNamen(t, m),
		"bestellanschreiben", "barcode_mapping", "etiketten_klein", "etiketten_gross")
}

// Ein normaler Händler klebt nicht selbst: kleiner Bogen ja, großes Lernmittel-Etikett nein.
func TestBestellAnhaenge_NormalerLieferantOhneGrossesEtikett(t *testing.T) {
	namen := anhangNamen(t, testBestellMail())

	pruefeEnthalten(t, namen, "bestellanschreiben", "barcode_mapping", "etiketten_klein")
	for _, name := range namen {
		if name == "etiketten_gross" {
			t.Error("normaler Lieferant bekommt das große Lernmittel-Etikett, das nur der Hauptlieferant klebt")
		}
	}
}

// Ohne Vorab-Barcodes gibt es weder Bogen noch CSV — und im Anschreiben keine Klebeanweisung.
func TestBestellAnhaenge_OhneVorabBarcodesNurDasAnschreiben(t *testing.T) {
	m := testBestellMail()
	m.MitVorabBarcodes = false

	if namen := anhangNamen(t, m); len(namen) != 1 || namen[0] != "bestellanschreiben" {
		t.Errorf("Anlagen = %v, erwartet nur das Anschreiben", namen)
	}
}

// Der Satz im Anschreiben und die Anlagen hängen an derselben Entscheidung: Der Brief nennt den
// beigefügten Bogen nur, wenn er beiliegt, den Link nur, wenn die Etiketten dahinter liegen, und
// ohne Vorab-Barcodes keines von beiden.
func TestBestellAnhaenge_DasAnschreibenNenntDenWegSeinerEtiketten(t *testing.T) {
	const bogen, link = "aus dem beigefügten Bogen", "Link in dieser E-Mail"
	for _, f := range []struct {
		name                   string
		vorabBarcodes, mitLink bool
		muss                   string
		bogenLiegtBei          bool
	}{
		{"Bogen an der Mail", true, false, bogen, true},
		{"Etiketten hinter dem Link", true, true, link, false},
		{"ohne Vorab-Barcodes", false, false, "", false},
		{"ohne Vorab-Barcodes, mit Link", false, true, "", false},
	} {
		m := testBestellMail()
		m.MitVorabBarcodes, m.MitBestaetigungsLink = f.vorabBarcodes, f.mitLink
		anhaenge, err := bestellAnhaenge(m)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		brief, liegtBei := "", false
		for _, a := range anhaenge {
			if strings.HasPrefix(a.Name, "bestellanschreiben_") {
				// Der Satz ist länger als die Zeile; verglichen wird ohne die Umbrüche des Blatts.
				brief = strings.Join(pdftest.TexteInReihenfolge(t, a.Data), " ")
			}
			if strings.HasPrefix(a.Name, "etiketten_") {
				liegtBei = true
			}
		}
		if liegtBei != f.bogenLiegtBei {
			t.Errorf("%s: Etikettenbogen an der Mail = %v, erwartet %v", f.name, liegtBei, f.bogenLiegtBei)
		}
		for _, satz := range []string{bogen, link} {
			if steht := strings.Contains(brief, satz); steht != (satz == f.muss) {
				t.Errorf("%s: %q im Anschreiben = %v, erwartet %v:\n%s", f.name, satz, steht, satz == f.muss, brief)
			}
		}
	}
}

func pruefeEnthalten(t *testing.T, namen []string, erwartet ...string) {
	t.Helper()
	vorhanden := make(map[string]bool, len(namen))
	for _, n := range namen {
		vorhanden[n] = true
	}
	for _, e := range erwartet {
		if !vorhanden[e] {
			t.Errorf("%q fehlt in den Anlagen (vorhanden: %v)", e, namen)
		}
	}
}

// Der stille Ausfall: Hauptlieferant, aber keine öffentliche Adresse. Die Mail ging raus
// und die Oberfläche meldete "erfolgreich gesendet" — dass der Händler nichts zum
// Bestätigen hatte, fiel erst auf, als die Bestätigung wochenlang ausblieb.
func TestBestellVersandMeldung_WarntWennDerLinkFehlt(t *testing.T) {
	status, meldung := bestellVersandMeldung("Naacher", true)
	if status != "warning" {
		t.Errorf("status = %q, erwartet \"warning\" — die Oberfläche zeigt sonst einen Erfolgs-Toast", status)
	}
	for _, teil := range []string{"OHNE Bestätigungs-Link", "öffentliche Adresse", "Bestellhistorie"} {
		if !strings.Contains(meldung, teil) {
			t.Errorf("Meldung nennt %q nicht — sie sagt dann nicht, was zu tun ist:\n%s", teil, meldung)
		}
	}

	status, meldung = bestellVersandMeldung("Naacher", false)
	if status != "success" {
		t.Errorf("status = %q, erwartet \"success\"", status)
	}
	if strings.Contains(meldung, "OHNE") {
		t.Errorf("Warnung erscheint, obwohl der Link mitging:\n%s", meldung)
	}
}
