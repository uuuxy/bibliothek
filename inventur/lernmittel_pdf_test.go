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

// Titel, Autor und Zweig stehen gekürzt mit Auslassungszeichen da, sobald sie gedruckt breiter
// wären als ihre Spalte; kurze Texte stehen ganz da. Gemessen wird die Breite, nicht die Zahl
// der Zeichen.
func TestSchulbuecherAlsPDF_Kuerzen(t *testing.T) {
	const (
		langerTitel = "DEUTSCHBUCH GYMNASIUM – ALLGEMEINE AUSGABE 2019, 5. SCHULJAHR, SCHÜLERBUCH MIT ARBEITSHEFT"
		langerAutor = "Annegret Müller-Lüdenscheidt und Kollegen"
		langerZweig = "Gymnasialzweig und Realschulzweig"
	)
	doc, err := SchulbuecherAlsPDF([]LernmittelTitel{
		{ID: "1", Title: langerTitel, Autor: langerAutor, ISBN: "ISBN-PROBE-1", JahrgangVon: 7, JahrgangBis: 7, Track: langerZweig},
		{ID: "2", Title: "Atlas", Autor: "Diercke", ISBN: "ISBN-PROBE-2", JahrgangVon: 7, JahrgangBis: 7, Track: "G"},
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Vor der ISBN stehen Titel und Autor der Zeile, hinter ihr Jahrgang und Zweig. Die Spalte
	// des Titels nimmt, was die übrigen von der Nutzbreite lassen.
	breiteTitel := nutzBreite - (spCover + spAutor + spISBN + spJg + spZweig + 3*spZahl)
	texte := pdftest.TexteInReihenfolge(t, doc)
	gefunden := 0
	for stelle, text := range texte {
		switch text {
		case "ISBN-PROBE-1":
			gefunden++
			titel, autor, zweig := texte[stelle-2], texte[stelle-1], texte[stelle+2]
			pdftest.InSpalte(t, titel, langerTitel, 8, breiteTitel-2)
			pdftest.InSpalte(t, autor, langerAutor, 8, spAutor-2)
			pdftest.InSpalte(t, zweig, langerZweig, 8, spZweig-2)
			if titel == langerTitel || autor == langerAutor || zweig == langerZweig {
				t.Errorf("lange Texte stehen ungekürzt da: %q, %q, %q", titel, autor, zweig)
			}
		case "ISBN-PROBE-2":
			gefunden++
			if titel, autor, zweig := texte[stelle-2], texte[stelle-1], texte[stelle+2]; titel != "Atlas" || autor != "Diercke" || zweig != "G" {
				t.Errorf("kurze Texte stehen als %q, %q und %q da", titel, autor, zweig)
			}
		}
	}
	if gefunden != 2 {
		t.Errorf("%d von 2 Zeilen gefunden", gefunden)
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
		{"5-10 ist eine Angabe wie jede andere", LernmittelTitel{JahrgangVon: 5, JahrgangBis: 10}, "5–10"},
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

// Der Jahrgang im Ausdruck ist der Zwilling der Browser-Seite (frontend/src/lib/utils/format.js,
// jahrgangSpanne). Beide Seiten lesen dieselben Prüffälle (jahrgangSpanne.faelle.json) —
// rechnen sie verschieden, nennt der Ausdruck einen Jahrgang anders als Titelliste und Buchakte.
func TestJahrgangText_WieImBrowser(t *testing.T) {
	const faelleDatei = "../frontend/src/lib/utils/jahrgangSpanne.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Faelle []struct {
			Fall string `json:"fall"`
			Von  int    `json:"von"`
			Bis  int    `json:"bis"`
			Soll string `json:"soll"`
		} `json:"faelle"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle parsen: %v", err)
	}
	if len(pruefung.Faelle) == 0 {
		t.Fatal("keine Prüffälle gelesen — zeigt der Pfad noch auf jahrgangSpanne.faelle.json?")
	}
	for _, f := range pruefung.Faelle {
		if ist := jahrgangText(LernmittelTitel{JahrgangVon: f.Von, JahrgangBis: f.Bis}); ist != f.Soll {
			t.Errorf("%s: jahrgangText(%d, %d) = %q, erwartet %q", f.Fall, f.Von, f.Bis, ist, f.Soll)
		}
	}
	// Die Browser-Seite muss dieselbe Datei lesen, sonst prüft jede Seite ihre eigenen Fälle.
	vitest, err := os.ReadFile("../frontend/src/lib/utils/format.test.js")
	if err != nil {
		t.Fatalf("format.test.js lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./jahrgangSpanne.faelle.json") {
		t.Error("format.test.js liest jahrgangSpanne.faelle.json nicht mehr ein")
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

// Ein Cover im Querformat bleibt in seiner Spalte. Skaliert nur über die Höhe, wäre es breiter
// als die Spalte und läge über dem Titel.
func TestSchulbuecherAlsPDF_QuerformatBleibtInSeinerSpalte(t *testing.T) {
	verzeichnis := t.TempDir()
	t.Chdir(verzeichnis)
	if err := os.MkdirAll(filepath.Join(verzeichnis, "uploads"), 0o750); err != nil {
		t.Fatal(err)
	}
	bild := image.NewRGBA(image.Rect(0, 0, 120, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			bild.Set(x, y, color.RGBA{R: 160, G: 60, B: 20, A: 255})
		}
	}
	var puffer bytes.Buffer
	if err := png.Encode(&puffer, bild); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(verzeichnis, "uploads", "quer.png"), puffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	doc, err := SchulbuecherAlsPDF([]LernmittelTitel{{ID: "1", Title: "Atlas", CoverURL: "/uploads/quer.png", Gesamt: 1}}, "Erdkunde", "")
	if err != nil {
		t.Fatal(err)
	}
	const mm = 72 / 25.4
	treffer := bildMatrix.FindAllStringSubmatch(string(pdftest.Inhalt(t, doc)), -1)
	if len(treffer) != 1 {
		t.Fatalf("%d Bildplatzierungen, erwartet das eine Cover", len(treffer))
	}
	var masse [3]float64 // Breite, Höhe, linker Rand
	for i := range masse {
		zahl, err := strconv.ParseFloat(treffer[0][i+1], 64)
		if err != nil {
			t.Fatalf("Bildmatrix unlesbar: %v", treffer[0])
		}
		masse[i] = zahl
	}
	breite, hoehe, links := masse[0], masse[1], masse[2]
	if breite/mm > coverBrt+0.05 || hoehe/mm > coverH+0.05 {
		t.Errorf("Cover ist %.1f × %.1f mm, der Platz %.0f × %.0f", breite/mm, hoehe/mm, coverBrt, coverH)
	}
	if links/mm < randLinks || (links+breite)/mm > randLinks+spCover {
		t.Errorf("Cover reicht von %.1f bis %.1f mm, die Spalte von %.0f bis %.0f", links/mm, (links+breite)/mm, randLinks, randLinks+spCover)
	}
}
