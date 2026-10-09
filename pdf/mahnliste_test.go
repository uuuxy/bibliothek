package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/pkg/coverdatei"
	"bibliothek/pkg/imageutil"
)

// Die Mahnliste, geprüft am gedruckten Blatt. Der Erzeuger kennt weder Klassen noch Abfragen:
// Er druckt je Schüler eine Seite aus dem, was seine Eingabe trägt. Dass die Tür sie richtig
// füllt, prüft api/mahnwesen_mail_test.go.

// mahnlisteUmgebung legt ein Arbeitsverzeichnis mit dem Ordner der Cover an. Der Erzeuger sucht
// ein Cover vom Arbeitsverzeichnis des Servers aus.
func mahnlisteUmgebung(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir(coverdatei.Wurzel, 0o750); err != nil {
		t.Fatalf("Ordner der Cover anlegen: %v", err)
	}
}

// schreibeCoverDatei legt ein Cover als WebP ab, wie der Abruf der Cover es tut
// (inventur/cover_storage.go), und liefert die Adresse, wie sie am Titel steht.
func schreibeCoverDatei(t *testing.T, name string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 60, 90))
	for y := 0; y < 90; y++ {
		for x := 0; x < 60; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 2), B: uint8(x + y), A: 255})
		}
	}
	daten, err := imageutil.EncodeImageWebP(img, 80)
	if err != nil {
		t.Fatalf("WebP schreiben: %v", err)
	}
	if err := os.WriteFile(filepath.Join(coverdatei.Wurzel, name), daten, 0o600); err != nil {
		t.Fatalf("Cover ablegen: %v", err)
	}
	return "/" + coverdatei.Wurzel + "/" + name
}

// mahnlisteMit baut eine Liste mit einem Schüler und einem Buch über der Frist.
func mahnlisteMit(coverURL string) []MahnlisteSchueler {
	return []MahnlisteSchueler{{
		Name:   "Test Schüler",
		Klasse: "07B",
		Medien: []MahnlisteMedium{{
			Titel:            "Das überfällige Buch",
			Autor:            "A. Utor",
			Barcode:          "B-0001",
			CoverURL:         coverURL,
			FaelligAm:        "01.06.2026",
			TageUeberfaellig: 21,
		}},
	}}
}

func mahnliste(t *testing.T, schueler []MahnlisteSchueler) []byte {
	t.Helper()
	roh, err := GenerateMahnlistePDF(schueler)
	if err != nil {
		t.Fatalf("Mahnliste drucken: %v", err)
	}
	pdftest.IstPDF(t, roh, "Mahnliste")
	return roh
}

// gofpdf erkennt den Bildtyp an der Dateiendung und liest kein WebP. Der Fehler bliebe am
// Dokument stehen und kostete bei Output die ganze Liste.
func TestMahnliste_WebPCoverBrichtDasDokumentNicht(t *testing.T) {
	mahnlisteUmgebung(t)
	coverURL := schreibeCoverDatei(t, "cover_auto_9780000000163.webp")

	mitCover := mahnliste(t, mahnlisteMit(coverURL))
	ohneCover := mahnliste(t, mahnlisteMit(""))

	// Ein still weggelassenes Cover bestünde den Test sonst auch. Das Bild des Strichcodes
	// tragen beide Blätter, deshalb zählt das zweite Bild.
	if n := bytes.Count(mitCover, []byte("/Subtype /Image")); n != 2 {
		t.Errorf("Blatt mit Cover trägt %d Bilder, erwartet 2 (Cover und Strichcode)", n)
	}
	if n := bytes.Count(ohneCover, []byte("/Subtype /Image")); n != 1 {
		t.Errorf("Blatt ohne Cover trägt %d Bilder, erwartet 1 (Strichcode)", n)
	}
}

// Was auch immer im Ordner der Cover liegt: Die Liste wird gedruckt, mit allem außer dem Bild.
func TestMahnliste_DefektesCoverKostetNurDasBild(t *testing.T) {
	mahnlisteUmgebung(t)
	kaputt := filepath.Join(coverdatei.Wurzel, "cover_kaputt.webp")
	if err := os.WriteFile(kaputt, []byte("das ist kein bild"), 0o600); err != nil {
		t.Fatalf("Datei ablegen: %v", err)
	}

	roh := mahnliste(t, mahnlisteMit("/"+coverdatei.Wurzel+"/cover_kaputt.webp"))

	pruefeBlatt(t, blattText(t, roh), []string{"Test Schüler", "Das überfällige Buch", "B-0001", "21 Tage"}, nil)
	if n := bytes.Count(roh, []byte("/Subtype /Image")); n != 1 {
		t.Errorf("Blatt trägt %d Bilder, erwartet 1 (nur der Strichcode)", n)
	}
}

// Mehrere Exemplare desselben Titels über der Frist: Das Cover steht einmal im Dokument.
func TestMahnliste_MehrfachesCoverWirdEinmalEingebettet(t *testing.T) {
	mahnlisteUmgebung(t)
	coverURL := schreibeCoverDatei(t, "cover_doppelt.webp")

	schueler := mahnlisteMit(coverURL)
	zweites := schueler[0].Medien[0]
	zweites.Barcode = "B-0002"
	schueler[0].Medien = append(schueler[0].Medien, zweites)

	roh := mahnliste(t, schueler)

	// Zwei Strichcodes und ein Cover.
	if n := bytes.Count(roh, []byte("/Subtype /Image")); n != 3 {
		t.Errorf("Blatt trägt %d Bilder, erwartet 3 (zwei Strichcodes, ein Cover)", n)
	}
}

func TestMahnliste_JeSchuelerEineSeiteMitSeinenBuechern(t *testing.T) {
	roh := mahnliste(t, []MahnlisteSchueler{
		{Name: "Anna Apfel", Klasse: "05F", Medien: []MahnlisteMedium{
			{Titel: "Gregs Tagebuch", Autor: "Jeff Kinney", Barcode: "B-00042", FaelligAm: "01.06.2026", TageUeberfaellig: 9},
		}},
		{Name: "Ben Birne", Klasse: "Ehemalige", Medien: []MahnlisteMedium{
			{Titel: "Atlas", Autor: "Diercke", Barcode: "B-00815", FaelligAm: "12.05.2026", TageUeberfaellig: 29},
			{Titel: "Mathebuch 7", Autor: "Lambacher", Barcode: "B-00816", FaelligAm: "13.05.2026", TageUeberfaellig: 28},
		}},
	})

	seiten := pdftest.TexteJeSeite(t, roh)
	if len(seiten) != 2 {
		t.Fatalf("%d Seiten, erwartet 2 (je Schüler eine)", len(seiten))
	}
	pruefeBlatt(t, strings.Join(seiten[0], "\n"), []string{
		"Anna Apfel", "05F", "Bitte gib die folgenden 1 Medium umgehend in der Schulbibliothek ab.",
		"Gregs Tagebuch", "Jeff Kinney", "B-00042", "01.06.2026", "9 Tage",
	}, []string{"Ben Birne", "Ehemalige", "Atlas", "B-00815", "Medien"})
	pruefeBlatt(t, strings.Join(seiten[1], "\n"), []string{
		"Ben Birne", "Ehemalige", "Bitte gib die folgenden 2 Medien umgehend in der Schulbibliothek ab.",
		"Atlas", "Diercke", "B-00815", "12.05.2026", "29 Tage",
		"Mathebuch 7", "Lambacher", "B-00816", "13.05.2026", "28 Tage",
	}, []string{"Anna Apfel", "05F", "Gregs Tagebuch", "B-00042"})

	// Name und Klasse stehen hinter ihrem Wort, die Spalten einer Zeile in der Reihenfolge der Köpfe.
	gesetzt := strings.Join(pdftest.TexteInReihenfolge(t, roh), " | ")
	for _, zeile := range []string{
		"Schüler/in: | Anna Apfel | Klasse: | 05F",
		"Buchtitel | Autor | Barcode | Fällig | Tage überfällig",
		"Gregs Tagebuch | Jeff Kinney | B-00042 | 01.06.2026 | 9 Tage",
		"Atlas | Diercke | B-00815 | 12.05.2026 | 29 Tage | Mathebuch 7 | Lambacher | B-00816 | 13.05.2026 | 28 Tage",
	} {
		if !strings.Contains(gesetzt, zeile) {
			t.Errorf("auf dem Blatt fehlt die Folge %q:\n%s", zeile, gesetzt)
		}
	}
}

func TestMahnliste_OhneSchuelerSagtEs(t *testing.T) {
	for name, schueler := range map[string][]MahnlisteSchueler{"keine Liste": nil, "leere Liste": {}} {
		seiten := pdftest.TexteJeSeite(t, mahnliste(t, schueler))
		if len(seiten) != 1 || len(seiten[0]) != 1 || seiten[0][0] != "Keine überfälligen Ausleihen vorhanden." {
			t.Errorf("%s: gedruckt %q, erwartet eine Seite mit dem Satz, dass nichts überfällig ist", name, seiten)
		}
	}
}

// Die Spalten sind schmal: Titel und Autor stehen gekürzt mit Auslassungszeichen da, nach
// Zeichen gezählt und nicht nach Bytes. Buchstaben außerhalb von cp1252 bleiben lesbar.
func TestMahnliste_KuerztTitelUndAutorNachZeichen(t *testing.T) {
	genau := strings.Repeat("ä", 38)
	zuLang := strings.Repeat("ö", 39)
	roh := mahnliste(t, []MahnlisteSchueler{{Name: "Şafak Łukasz", Klasse: "07B", Medien: []MahnlisteMedium{
		{Titel: genau, Autor: strings.Repeat("é", 19), Barcode: "B-1"},
		{Titel: zuLang, Autor: strings.Repeat("ü", 20), Barcode: "B-2"},
	}}})

	pruefeBlatt(t, strings.Join(pdftest.Texte(t, roh), "\n"), []string{
		"Safak Lukasz",
		genau, strings.Repeat("é", 19),
		strings.Repeat("ö", 37) + "…", strings.Repeat("ü", 18) + "…",
	}, []string{zuLang, strings.Repeat("ü", 20)})
}

// Ab 15 Tagen über der Frist steht die Zahl der Tage in Rot, bis 14 nicht.
func TestMahnliste_AbFuenfzehnTagenStehtDieZahlInRot(t *testing.T) {
	const rot = "0.784 0.118 0.118 rg"
	for _, f := range []struct {
		tage    int
		willRot bool
	}{{14, false}, {15, true}} {
		roh := mahnliste(t, []MahnlisteSchueler{{Name: "N", Klasse: "K", Medien: []MahnlisteMedium{
			{Titel: "T", Barcode: "B-1", TageUeberfaellig: f.tage},
		}}})
		if hat := bytes.Contains(pdftest.Inhalt(t, roh), []byte(rot)); hat != f.willRot {
			t.Errorf("%d Tage: Rot gesetzt = %v, erwartet %v", f.tage, hat, f.willRot)
		}
	}
}

// Ohne Barcode bleibt die Zelle leer; mit Barcode stehen Bild und Nummer da.
func TestMahnliste_BarcodeAlsBildUndNummer(t *testing.T) {
	mit := mahnliste(t, []MahnlisteSchueler{{Name: "N", Klasse: "K", Medien: []MahnlisteMedium{{Titel: "T", Barcode: "B-4711"}}}})
	ohne := mahnliste(t, []MahnlisteSchueler{{Name: "N", Klasse: "K", Medien: []MahnlisteMedium{{Titel: "T"}}}})

	if !bytes.Contains(mit, []byte("/Subtype /Image")) || !strings.Contains(blattText(t, mit), "B-4711") {
		t.Error("mit Barcode fehlt das Bild des Strichcodes oder die Nummer darunter")
	}
	if bytes.Contains(ohne, []byte("/Subtype /Image")) {
		t.Error("ohne Barcode trägt das Blatt ein Bild")
	}
}

// Text und Bild im Inhaltsstrom einer Seite, in Punkt und von unten gemessen:
// `BT 76.54 584.37 Td (Titel 1)Tj ET` und `q 19.84 0 0 48.19 51.02 562.68 cm /I… Do Q`.
var (
	mahnlisteTextOrt = regexp.MustCompile(`BT ([0-9.]+) ([0-9.]+) Td \(((?:\\.|[^()\\])*)\)Tj ET`)
	mahnlisteBildOrt = regexp.MustCompile(`q ([0-9.]+) 0 0 ([0-9.]+) ([0-9.]+) ([0-9.]+) cm /I[0-9a-f]+ Do Q`)
)

// mahnlisteSeite ist, was auf einer Seite steht: jeder Text mit seiner Höhe und die Mitten der
// Bilder.
type mahnlisteSeite struct {
	texte       map[string]float64
	bildMitten  []float64
	reihenfolge []string
}

func zahl(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("Zahl im Inhaltsstrom unlesbar: %q", s)
	}
	return f
}

func mahnlisteSeiten(t *testing.T, roh []byte) []mahnlisteSeite {
	t.Helper()
	var seiten []mahnlisteSeite
	for _, strom := range pdftest.InhaltJeSeite(t, roh) {
		seite := mahnlisteSeite{texte: map[string]float64{}}
		for _, m := range mahnlisteTextOrt.FindAllSubmatch(strom, -1) {
			seite.texte[string(m[3])] = zahl(t, string(m[2]))
			seite.reihenfolge = append(seite.reihenfolge, string(m[3]))
		}
		for _, m := range mahnlisteBildOrt.FindAllSubmatch(strom, -1) {
			seite.bildMitten = append(seite.bildMitten, zahl(t, string(m[4]))+zahl(t, string(m[2]))/2)
		}
		seiten = append(seiten, seite)
	}
	return seiten
}

// langeMahnliste baut einen Schüler mit n Büchern, jedes mit Cover und eigenen Texten.
func langeMahnliste(n int, coverURL string) []MahnlisteSchueler {
	medien := make([]MahnlisteMedium, 0, n)
	for i := 1; i <= n; i++ {
		medien = append(medien, MahnlisteMedium{
			Titel: fmt.Sprintf("Titel %02d", i), Autor: fmt.Sprintf("Autor %02d", i), Barcode: fmt.Sprintf("B-9%03d", i),
			CoverURL: coverURL, FaelligAm: fmt.Sprintf("%02d.06.2026", i), TageUeberfaellig: 100 + i,
		})
	}
	return []MahnlisteSchueler{{Name: "Anna Apfel", Klasse: "05F", Medien: medien}}
}

// Eine Zeile setzt Cover, Strichcode und Nummer an feste Stellen. Bricht gofpdf mitten in ihr
// um, stehen ihre Teile auf drei Seiten; Schulbücher eines Jahres sind schnell mehr als zehn.
func TestMahnliste_LangeListeHaeltJedeZeileBeisammen(t *testing.T) {
	mahnlisteUmgebung(t)
	const anzahl = 25
	seiten := mahnlisteSeiten(t, mahnliste(t, langeMahnliste(anzahl, schreibeCoverDatei(t, "cover_lang.webp"))))

	// Eine Zeile ist 18 mm hoch; halb so viel über und unter dem Titel gehört zu ihr.
	const halbeZeile = 18.0 / 2 / 25.4 * 72
	gefunden := 0
	var jeSeite []int
	for nr, seite := range seiten {
		zeilen := 0
		for i := 1; i <= anzahl; i++ {
			titelHoehe, da := seite.texte[fmt.Sprintf("Titel %02d", i)]
			if !da {
				continue
			}
			zeilen++
			for _, teil := range []string{fmt.Sprintf("Autor %02d", i), fmt.Sprintf("B-9%03d", i), fmt.Sprintf("%02d.06.2026", i), fmt.Sprintf("%d Tage", 100+i)} {
				hoehe, da := seite.texte[teil]
				if !da {
					t.Errorf("Seite %d: %q steht nicht auf der Seite seines Titels", nr+1, teil)
				} else if math.Abs(hoehe-titelHoehe) > halbeZeile {
					t.Errorf("Seite %d: %q steht %.0f Punkt neben seinem Titel", nr+1, teil, hoehe-titelHoehe)
				}
			}
			bilder := 0
			for _, mitte := range seite.bildMitten {
				if math.Abs(mitte-titelHoehe) <= halbeZeile {
					bilder++
				}
			}
			if bilder != 2 {
				t.Errorf("Seite %d, Zeile %d: %d Bilder in der Zeile, erwartet Cover und Strichcode", nr+1, i, bilder)
			}
		}
		gefunden += zeilen
		jeSeite = append(jeSeite, zeilen)
		if _, koepfe := seite.texte["Buchtitel"]; zeilen > 0 && !koepfe {
			t.Errorf("Seite %d: %d Zeilen ohne Spaltenköpfe", nr+1, zeilen)
		}
		if _, fortsetzung := seite.texte["Fortsetzung: Anna Apfel, 05F"]; fortsetzung != (nr > 0) {
			t.Errorf("Seite %d: Fortsetzung mit Name und Klasse = %v, erwartet ab der zweiten Seite", nr+1, fortsetzung)
		}
		if len(seite.bildMitten) != 2*zeilen {
			t.Errorf("Seite %d: %d Bilder bei %d Zeilen", nr+1, len(seite.bildMitten), zeilen)
		}
	}
	if gefunden != anzahl {
		t.Errorf("%d Zeilen gedruckt, erwartet %d", gefunden, anzahl)
	}
	if fmt.Sprint(jeSeite) != "[10 12 3]" {
		t.Errorf("Zeilen je Seite: %v, erwartet [10 12 3]", jeSeite)
	}
}

// Bis zehn Bücher bleibt es eine Seite. Die Fußzeile steht nie allein auf einem Blatt: Jede
// Seite trägt mindestens eine Zeile, die letzte dazu die Fußzeile.
func TestMahnliste_FusszeileStehtBeiDerLetztenZeile(t *testing.T) {
	const fuss = "Schulbibliothek – Bei Fragen wende dich bitte an das Bibliotheksteam."
	for n := 1; n <= 40; n++ {
		seiten := pdftest.TexteJeSeite(t, mahnliste(t, langeMahnliste(n, "")))
		wieViele := 1 + (max(n, 10)-10+11)/12
		if len(seiten) != wieViele {
			t.Errorf("%d Bücher: %d Seiten, erwartet %d", n, len(seiten), wieViele)
		}
		for nr, seite := range seiten {
			text := strings.Join(seite, "\n")
			if !strings.Contains(text, "Titel ") {
				t.Errorf("%d Bücher, Seite %d: keine Zeile auf der Seite:\n%s", n, nr+1, text)
			}
			if strings.Contains(text, fuss) != (nr == len(seiten)-1) {
				t.Errorf("%d Bücher, Seite %d von %d: Fußzeile = %v, erwartet nur auf der letzten", n, nr+1, len(seiten), strings.Contains(text, fuss))
			}
		}
	}
}

// Ohne Klasse nennt die Folgeseite nur den Namen, ohne ein Komma ins Leere.
func TestMahnliste_FortsetzungOhneKlasseNenntNurDenNamen(t *testing.T) {
	schueler := langeMahnliste(11, "")
	schueler[0].Klasse = ""
	seiten := pdftest.TexteJeSeite(t, mahnliste(t, schueler))
	if len(seiten) != 2 {
		t.Fatalf("%d Seiten, erwartet 2", len(seiten))
	}
	fortsetzung := ""
	for _, text := range seiten[1] {
		if strings.HasPrefix(text, "Fortsetzung") {
			fortsetzung = text
		}
	}
	if fortsetzung != "Fortsetzung: Anna Apfel" {
		t.Errorf("Folgeseite beginnt mit %q, erwartet %q", fortsetzung, "Fortsetzung: Anna Apfel")
	}
}
