package bestellung

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/internal/repository"
)

// Titel, Autor und ISBN stehen gekürzt da, sobald sie gedruckt breiter wären als ihre Spalte:
// gofpdf druckt Überlanges über die Nachbarzelle. „Fundamente der Mathematik 8 Gymnasium Hessen"
// ist in 8 Punkt 64,6 mm breit, die Spalte 62 mm. Jede Seite mit Zeilen trägt die Spaltenköpfe.
func TestBestellliste_TextBleibtInSeinerSpalteUndKoepfeAufJederSeite(t *testing.T) {
	bedarf := []repository.Bestellbedarf{
		{Titel: "Fundamente der Mathematik 8 Gymnasium Hessen", Autor: "Annegret Müller-Lüdenscheidt, Bernd Kollege",
			ISBN: "978-3-06-040322-6", Bedarf: 1, GesamtBestand: 2, Fehlen: 1001},
		{Titel: strings.Repeat("W", 60), Autor: strings.Repeat("M", 30), ISBN: strings.Repeat("9", 40), Bedarf: 1, GesamtBestand: 2, Fehlen: 1002},
		{Titel: "Kurz", Autor: "A. Utor", ISBN: "978", Bedarf: 1, GesamtBestand: 2, Fehlen: 1003},
	}
	for i := 0; i < 60; i++ {
		bedarf = append(bedarf, repository.Bestellbedarf{Titel: fmt.Sprintf("Füllzeile %d", i), Fehlen: 2000 + i})
	}
	var roh bytes.Buffer
	if err := baueBestelllistePDF(bedarf).Output(&roh); err != nil {
		t.Fatal(err)
	}

	texte := pdftest.TexteInReihenfolge(t, roh.Bytes())
	gekuerzt := map[string]bool{}
	for stelle, text := range texte {
		for _, b := range bedarf[:3] {
			if text != fmt.Sprint(b.Fehlen) {
				continue
			}
			// Vor der Menge stehen Titel, Autor, ISBN, Bedarf und Bestand der Zeile.
			titel, autor, isbn := texte[stelle-5], texte[stelle-4], texte[stelle-3]
			pdftest.InSpalte(t, titel, b.Titel, 8, 60)
			pdftest.InSpalte(t, autor, b.Autor, 8, 33)
			pdftest.InSpalte(t, isbn, b.ISBN, 8, 31)
			gekuerzt[text] = titel != b.Titel || autor != b.Autor || isbn != b.ISBN
		}
	}
	if erwartet := map[string]bool{"1001": true, "1002": true, "1003": false}; fmt.Sprint(gekuerzt) != fmt.Sprint(erwartet) {
		t.Errorf("gekürzt: %v\nerwartet: %v", gekuerzt, erwartet)
	}

	seiten := pdftest.TexteJeSeite(t, roh.Bytes())
	if len(seiten) < 2 {
		t.Fatalf("%d Seiten, erwartet mindestens zwei", len(seiten))
	}
	for i, seite := range seiten {
		koepfe := false
		for _, text := range seite {
			koepfe = koepfe || text == "Buchtitel"
		}
		if !koepfe {
			t.Errorf("Seite %d trägt keine Spaltenköpfe", i+1)
		}
	}
}
