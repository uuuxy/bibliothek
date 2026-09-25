package inventur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
)

// Bildplatzierungen im Inhaltsstrom: `q 10.00 0 0 15.00 17.00 260.00 cm /I1 Do Q`.
// Die vierte Zahl ist die Höhe, die sechste die y-Position VON UNTEN (PDF-Koordinaten).
var bildMatrix = regexp.MustCompile(`([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm`)

// TestSchulbuecherAlsPDF_CoverBleibtBeiSeinerZeile hält den Seitenumbruch fest.
//
// gofpdf prüft den Umbruch erst in der ersten Tabellenzelle. Das Cover wird aber mit
// fester Position gezeichnet und ging deshalb noch auf die ALTE Seite, während seine
// Zeile schon auf der neuen stand: ab der 14. Zeile klebte auf jedem Blattende ein
// herrenloses Buchcover im Fußsteg, und die zugehörige Zeile hatte oben ein leeres
// Bildkästchen — was aussieht wie „für dieses Buch ist kein Cover hinterlegt"
// (Befund Rasterdurchgang 03.09.2026). Der PG-Test sah es nicht: drei Titel, eine Seite.
func TestSchulbuecherAlsPDF_CoverBleibtBeiSeinerZeile(t *testing.T) {
	// coverdatei liest relativ zum Arbeitsverzeichnis; ein eigenes uploads/ mit einem
	// echten Bild darin macht den Cover-Pfad im Test begehbar.
	verzeichnis := t.TempDir()
	t.Chdir(verzeichnis)
	if err := os.MkdirAll(filepath.Join(verzeichnis, "uploads"), 0o750); err != nil {
		t.Fatal(err)
	}
	bild := image.NewRGBA(image.Rect(0, 0, 40, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 40; x++ {
			bild.Set(x, y, color.RGBA{R: 20, G: 60, B: 160, A: 255})
		}
	}
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(verzeichnis, "uploads", "cover.png"), puffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	titel := make([]LernmittelTitel, 0, 30)
	for i := 1; i <= 30; i++ {
		titel = append(titel, LernmittelTitel{
			ID: strconv.Itoa(i), Title: fmt.Sprintf("Titel %02d", i), Autor: "Verlag",
			CoverURL: "/uploads/cover.png", Gesamt: i, Verfuegbar: i,
		})
	}

	doc, err := SchulbuecherAlsPDF(titel, "Biologie", "")
	if err != nil {
		t.Fatal(err)
	}
	pdftest.IstPDF(t, doc, "Schulbuch-Export über mehrere Seiten")

	treffer := bildMatrix.FindAllStringSubmatch(string(pdftest.Inhalt(t, doc)), -1)
	if len(treffer) < 20 {
		t.Fatalf("erwartet ein Cover je Zeile, gefunden %d Bildplatzierungen — "+
			"der Detektor greift nicht mehr", len(treffer))
	}
	// Kein Bild darf in den Fußsteg ragen: Unterkante = y (von unten) muss mindestens
	// dem unteren Rand entsprechen. Mit dem alten Fehler lag ein Bild je Seite bei y≈5.
	for _, m := range treffer {
		y, err := strconv.ParseFloat(m[4], 64)
		if err != nil {
			t.Fatalf("Bildmatrix unlesbar: %v", m)
		}
		if y < 15 {
			t.Errorf("Cover ragt in den Fußsteg (y=%.1f von unten) — es gehört zur Zeile "+
				"auf der nächsten Seite, steht aber noch auf dieser", y)
		}
	}
}

func TestSchulbuecherAlsPDF_Inhalt(t *testing.T) {
	titel := []LernmittelTitel{
		{
			Title:       "Chemie Heute",
			Autor:       "Schroedel",
			ISBN:        "978-3-507-86221-0",
			Subject:     "Chemie",
			JahrgangVon: 8,
			JahrgangBis: 10,
			Track:       "G-Zweig",
			Gezaehlt:    "01.01.2023",
			Gesamt:      50,
			Verliehen:   30,
			Verfuegbar:  20,
		},
	}

	doc, err := SchulbuecherAlsPDF(titel, "Chemie", "Filter")
	if err != nil {
		t.Fatal(err)
	}

	texte := pdftest.Texte(t, doc)

	erwartet := []string{
		"Chemie",
		"Chemie Heute",
		"Schroedel",
		"978-3-507-86221-0",
		"8–10",
		"G-Zweig",
		"01.01.2023",
		"50",
		"30",
		"20",
		"Filter · 1 Titel · Stand",
	}

	for _, e := range erwartet {
		gefunden := false
		for _, text := range texte {
			if strings.Contains(text, e) {
				gefunden = true
				break
			}
		}
		if !gefunden {
			t.Errorf("Erwarteter Text %q fehlt im PDF", e)
		}
	}
}

func TestSchulbuecherAlsPDF_Kuerzen(t *testing.T) {
	langerTitel := "Das ist ein extrem langer Buchtitel der auf jeden Fall gekürzt werden muss damit er nicht in die nächste Spalte überläuft und das Layout zerstört"
	langerAutor := "Ein Autor mit einem unfassbar langen Namen der das Feld sprengt"

	titel := []LernmittelTitel{
		{
			Title: langerTitel,
			Autor: langerAutor,
		},
	}

	doc, err := SchulbuecherAlsPDF(titel, "", "")
	if err != nil {
		t.Fatal(err)
	}

	texte := pdftest.Texte(t, doc)

	// kuerze schneidet bei limit ab und setzt ein '…'
	// spAutor ist 24.0. 16 ist das Limit in zeichneSchulbuchZeile für kuerze() beim Autor.
	erwarteterAutorGekuerzt := "Ein Autor mit e…"

	// Titel Limit ist breiteTitel/1.6
	// nutzBreite = 178.0, belegt ohne Gezaehlt = 12+24+23+12+21+(3*8) = 116. breiteTitel = 62.
	// 62 / 1.6 = 38
	erwarteterTitelGekuerzt := string([]rune(langerTitel)[:38-1]) + "…"

	gefundenAutor := false
	gefundenTitel := false
	for _, text := range texte {
		if text == erwarteterAutorGekuerzt {
			gefundenAutor = true
		}
		if text == erwarteterTitelGekuerzt {
			gefundenTitel = true
		}
	}
	if !gefundenAutor {
		t.Errorf("Langer Autor wurde nicht korrekt gekürzt (erwartet %q)", erwarteterAutorGekuerzt)
	}
	if !gefundenTitel {
		t.Errorf("Langer Titel wurde nicht korrekt gekürzt (erwartet %q)", erwarteterTitelGekuerzt)
	}
}

func TestSchulbuecherAlsPDF_LeereAuswahl(t *testing.T) {
	doc, err := SchulbuecherAlsPDF(nil, "", "")
	if err != nil {
		t.Fatal(err)
	}

	texte := pdftest.Texte(t, doc)

	gefunden := false
	for _, text := range texte {
		if text == "Keine Schulbücher in dieser Auswahl." {
			gefunden = true
			break
		}
	}

	if !gefunden {
		t.Errorf("Hinweistext für leere Auswahl fehlt im PDF")
	}
}

func TestSchulbuecherAlsPDF_GezaehltSpalte(t *testing.T) {
	// 1. Ohne Gezaehlt
	titelOhne := []LernmittelTitel{{Title: "Buch 1"}}
	docOhne, err := SchulbuecherAlsPDF(titelOhne, "", "")
	if err != nil {
		t.Fatal(err)
	}
	texteOhne := pdftest.Texte(t, docOhne)
	for _, text := range texteOhne {
		if text == "Gezählt" {
			t.Errorf("Spalte 'Gezählt' darf nicht erscheinen, wenn kein Buch gezählt wurde")
		}
	}

	// 2. Mit Gezaehlt
	titelMit := []LernmittelTitel{{Title: "Buch 1", Gezaehlt: "02.02.2022"}}
	docMit, err := SchulbuecherAlsPDF(titelMit, "", "")
	if err != nil {
		t.Fatal(err)
	}
	texteMit := pdftest.Texte(t, docMit)
	gefunden := false
	for _, text := range texteMit {
		if text == "Gezählt" {
			gefunden = true
			break
		}
	}
	if !gefunden {
		t.Errorf("Spalte 'Gezählt' muss erscheinen, wenn mindestens ein Buch gezählt wurde")
	}
}

func TestSchulbuecherAlsPDF_JahrgangText(t *testing.T) {
	tests := []struct {
		name string
		t    LernmittelTitel
		want string
	}{
		{"0-0", LernmittelTitel{JahrgangVon: 0, JahrgangBis: 0}, ""},
		{"5-10 (unbekannt)", LernmittelTitel{JahrgangVon: 5, JahrgangBis: 10}, ""},
		{"Einzeljahrgang", LernmittelTitel{JahrgangVon: 7, JahrgangBis: 7}, "7"},
		{"Von-Bis", LernmittelTitel{JahrgangVon: 7, JahrgangBis: 13}, "7–13"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jahrgangText(tt.t)
			if got != tt.want {
				t.Errorf("jahrgangText() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Die Beschriftung einer Auflage im Ausdruck ist der Zwilling der Browser-Seite
// (frontend/src/lib/utils/auflagenText.js). Beide Seiten lesen dieselben Prüffälle
// (auflagenText.faelle.json, Bauart wie code39.faelle.json) — rechnen sie verschieden, nennt
// das PDF eine Auflage anders als der Bildschirm. Bis zum Rasterdurchgang vom 25.09.2026 standen
// die Fälle zweimal von Hand da.
func TestAuflagenBeschriftung_WieImBrowser(t *testing.T) {
	const faelleDatei = "../frontend/src/lib/utils/auflagenText.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Beschriftung []struct {
			Fall    string `json:"fall"`
			Auflage string `json:"auflage"`
			Jahr    int    `json:"jahr"`
			Soll    string `json:"soll"`
		} `json:"beschriftung"`
		Aufschluesselung []struct {
			Fall     string `json:"fall"`
			Auflagen []struct {
				Auflage string `json:"auflage"`
				Jahr    int    `json:"jahr"`
				Bestand int    `json:"bestand"`
			} `json:"auflagen"`
			Soll string `json:"soll"`
		} `json:"aufschluesselung"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}
	if len(pruefung.Beschriftung) < 6 || len(pruefung.Aufschluesselung) < 2 {
		t.Fatalf("%d Beschriftungen, %d Aufschlüsselungen — erwartet mindestens 6 und 2: liest der Test "+
			"noch auf auflagenText.faelle.json?", len(pruefung.Beschriftung), len(pruefung.Aufschluesselung))
	}
	for _, f := range pruefung.Beschriftung {
		if ist := auflagenBeschriftung(f.Auflage, f.Jahr); ist != f.Soll {
			t.Errorf("%s: auflagenBeschriftung(%q, %d) = %q, erwartet %q", f.Fall, f.Auflage, f.Jahr, ist, f.Soll)
		}
	}
	for _, f := range pruefung.Aufschluesselung {
		auflagen := make([]AuflageImBestand, 0, len(f.Auflagen))
		for _, a := range f.Auflagen {
			auflagen = append(auflagen, AuflageImBestand{Auflage: a.Auflage, Erscheinungsjahr: a.Jahr, Gesamt: a.Bestand})
		}
		if ist := auflagenAufschluesselung(auflagen); ist != f.Soll {
			t.Errorf("%s: auflagenAufschluesselung = %q, erwartet %q", f.Fall, ist, f.Soll)
		}
	}

	// Liest die JavaScript-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../frontend/src/lib/utils/auflagenText.test.js")
	if err != nil {
		t.Fatalf("Vitest lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./auflagenText.faelle.json") {
		t.Error("auflagenText.test.js liest auflagenText.faelle.json nicht mehr ein")
	}
}

// Ein Buch in mehreren Auflagen (docs/OFFEN.md 4.18, Stufe 6): Unter dem Titel steht im
// Ausdruck, woraus die Zahlen der Zeile bestehen — ein Buch ohne weitere Auflage bleibt, wie es
// war.
func TestSchulbuecherAlsPDF_AuflagenUnterDemTitel(t *testing.T) {
	titel := []LernmittelTitel{
		{
			Title: "Mathe 7", ISBN: "978-B2", Subject: "Mathematik", JahrgangVon: 7, JahrgangBis: 7,
			Gesamt: 58, Verliehen: 12, Verfuegbar: 46,
			AuflagenBestand: []AuflageImBestand{
				{ID: "neu", Auflage: "4. Aufl.", Erscheinungsjahr: 2023, Gesamt: 16},
				{ID: "alt", Auflage: "3. Aufl.", Erscheinungsjahr: 2019, Gesamt: 42},
			},
		},
		{Title: "Mathe 8", ISBN: "978-B8", Subject: "Mathematik", JahrgangVon: 8, JahrgangBis: 8, Gesamt: 5, Verfuegbar: 5},
	}
	doc, err := SchulbuecherAlsPDF(titel, "Mathematik", "")
	if err != nil {
		t.Fatal(err)
	}
	texte := pdftest.Texte(t, doc)
	alles := strings.Join(texte, "\n")
	for _, e := range []string{"Mathe 7", "58", "Mathe 8"} {
		if !strings.Contains(alles, e) {
			t.Errorf("Erwarteter Text %q fehlt im PDF", e)
		}
	}
	// Die Aufschlüsselung bricht in der Titelspalte um, und pdftest liefert die Zeilen sortiert,
	// nicht in Lesereihenfolge: Der Satz muss aus einer oder zwei Zeilen zusammengehen.
	const soll = "Bestand aus 2 Auflagen: 4. Aufl. · 2023 (16), 3. Aufl. · 2019 (42)"
	gefunden := false
	for _, a := range texte {
		for _, b := range append([]string{""}, texte...) {
			if strings.TrimSpace(a+" "+b) == soll {
				gefunden = true
			}
		}
	}
	if !gefunden {
		t.Errorf("Aufschlüsselung %q fehlt im PDF (Texte: %q)", soll, texte)
	}
	if n := strings.Count(alles, "Bestand aus"); n != 1 {
		t.Errorf("die Aufschlüsselung steht %d-mal im PDF — erwartet nur beim Buch in zwei Auflagen", n)
	}
}
