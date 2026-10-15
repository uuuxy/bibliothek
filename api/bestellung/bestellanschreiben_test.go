package bestellung

import (
	"reflect"
	"strings"
	"testing"

	"bibliothek/internal/pdftest"
	"bibliothek/internal/tuertest"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"

	"github.com/google/go-cmp/cmp"
)

// anschreibenBlatt druckt das Anschreiben, wie die Anlagen der Bestellmail es tun.
func anschreibenBlatt(positionen []bestellterTitel, schule pdf.SchuleInfo, weg pdf.EtikettenWeg, mittel string) ([]byte, error) {
	anschreiben, err := bestellanschreiben(positionen, weg, mittel)
	if err != nil {
		return nil, err
	}
	return pdf.ErzeugeBestellanschreiben(anschreiben, schule)
}

// Jedes Feld der Eingabe kommt aus seiner Quelle, die Werte sind je Feld verschieden; ein Feld,
// das die Eingabe später dazubekommt und das niemand füllt, fällt an tuertest.LeereFelder auf. Den Verlag
// druckt das Anschreiben nicht.
func TestBestellanschreiben_FuelltJedesFeldDerEingabe(t *testing.T) {
	positionen := []bestellterTitel{
		{Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", Verlag: "Verlag 1", Menge: 30},
		{Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", Verlag: "Verlag 2", Menge: 2},
	}
	texteJeTopf := map[string]mitteltopf.Texte{}
	for _, mittel := range []string{mitteltopf.Land, mitteltopf.Schultraeger} {
		eingabe, err := bestellanschreiben(positionen, pdf.BogenHinterLink, mittel)
		if err != nil {
			t.Fatalf("%s: %v", mittel, err)
		}
		texte, err := mitteltopf.TexteFuer(mittel)
		if err != nil {
			t.Fatalf("%s: %v", mittel, err)
		}
		texteJeTopf[mittel] = texte
		want := pdf.Bestellanschreiben{
			Betreff: texte.Betreff, Vermerk: texte.Vermerk, Etiketten: pdf.BogenHinterLink,
			Positionen: []pdf.BestellPosition{
				{Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", Menge: 30},
				{Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", Menge: 2},
			},
		}
		if !cmp.Equal(eingabe, want) {
			t.Errorf("%s: Eingabe des Anschreibens =\n%+v\nerwartet\n%+v", mittel, eingabe, want)
		}
		if leer := tuertest.LeereFelder(reflect.ValueOf(eingabe), "pdf.Bestellanschreiben"); len(leer) > 0 {
			t.Errorf("%s: bestellanschreiben füllt diese Felder nicht: %v", mittel, leer)
		}
	}
	land, traeger := texteJeTopf[mitteltopf.Land], texteJeTopf[mitteltopf.Schultraeger]
	if land.Betreff == traeger.Betreff || land.Vermerk == traeger.Vermerk {
		t.Error("die Texte der zwei Töpfe sind gleich; der Test unterschiede sie nicht")
	}
}

// Der Vermerk des Topfs steht auf dem fertigen Anschreiben — und nie der des anderen.
//
// Geprüft am entpackten PDF-Inhaltsstrom (internal/pdftest), nicht am Eingabetext: Stünde
// auf jedem Anschreiben „für unsere Schulbibliothek", auch für Lernmittel-Klassensätze,
// gewährte der Händler danach den falschen Nachlass, und am Bildschirm sähe es niemand, weil
// das PDF erst im Postfach des Händlers aufgeht.
func TestBestellanschreibenTraegtDenVermerkDesTopfsUndNieDenAnderen(t *testing.T) {
	items := []bestellterTitel{{Titel: "Mathematik 7", Autor: "Autor", ISBN: "978", Menge: 30}}
	schule := pdf.SchuleInfo{Name: "Testschule"}

	// Die Wörter, an denen die beiden Töpfe im Brief zu unterscheiden sind. Kurz
	// gehalten, weil gofpdf lange Zeilen umbricht und ein ganzer Satz dann nicht mehr
	// am Stück im Strom steht.
	kennzeichen := map[string][]string{
		mitteltopf.Land:         {"Lernmittelfreiheit", "Sammelbestellung"},
		mitteltopf.Schultraeger: {"lerb", "Schultr"},
	}

	for mittel, erwartet := range kennzeichen {
		t.Run(mittel, func(t *testing.T) {
			roh, err := anschreibenBlatt(items, schule, pdf.BogenLiegtBei, mittel)
			if err != nil {
				t.Fatalf("Anschreiben erzeugen: %v", err)
			}
			text := pdftest.Suchtext(t, roh)
			for _, wort := range erwartet {
				if !strings.Contains(text, wort) {
					t.Errorf("%q fehlt auf dem Anschreiben für %s", wort, mittel)
				}
			}
			// Der andere Topf darf nicht vorkommen, das ist der eigentliche Fehler. Für den
			// Schulträger-Brief heißt das: „Lernmittelfreiheit" darf nur
			// in der Verneinung stehen; der Brief nennt es einmal („keine Beschaffung im
			// Rahmen der Lernmittelfreiheit"), also prüfen wir das Eigentums-Wort.
			if mittel == mitteltopf.Schultraeger && strings.Contains(text, "Sammelbestellung") {
				t.Error("Schulträger-Anschreiben nennt sich Sammelbestellung — das ist der Lernmittel-Vermerk")
			}
			if mittel == mitteltopf.Land && strings.Contains(text, "Schultr") {
				t.Error("Land-Anschreiben nennt den Schulträger")
			}
			if strings.Contains(text, "unsere Schulbibliothek") {
				t.Error("der alte Satz „für unsere Schulbibliothek“ steht noch im Brief")
			}
		})
	}
}

// Ohne gültigen Topf gibt es kein Anschreiben — ein Brief ohne Vermerk wäre genau das
// Dokument, das dieses Paket abschafft.
func TestBestellanschreibenVerweigertUnbekanntenTopf(t *testing.T) {
	for _, mittel := range []string{"", "kreis", "Land"} {
		if _, err := anschreibenBlatt(nil, pdf.SchuleInfo{}, pdf.OhneEtiketten, mittel); err == nil {
			t.Errorf("Topf %q: Anschreiben entstand ohne Vermerk", mittel)
		}
	}
}

// Die Mail-Anlagen hängen am selben Schalter: Ein bestellMail ohne Topf scheitert an der
// Anschreiben-Erzeugung, bevor irgendetwas verschickt wird.
func TestBestellAnhaengeOhneTopfScheitern(t *testing.T) {
	m := testBestellMail()
	m.Mittel = ""
	if _, err := bestellAnhaenge(m); err == nil {
		t.Error("Anlagen entstanden ohne Topf")
	}
}
