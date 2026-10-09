package pdf

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
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
