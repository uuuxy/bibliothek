package api

import (
	"strings"
	"testing"

	"bibliothek/pdf"
	"bibliothek/repository"
)

// Der Vermerk des Topfs steht auf dem FERTIGEN Anschreiben — und nie der des anderen.
//
// Geprüft am entpackten PDF-Inhaltsstrom (internal/pdftest), nicht am Eingabetext: Bis zum
// 10.09.2026 stand auf jedem Anschreiben „für unsere Schulbibliothek", auch für
// Lernmittel-Klassensätze — der Händler hätte danach den falschen Nachlass gewährt, und
// am Bildschirm sah es niemand, weil das PDF erst im Postfach des Händlers aufgeht.
func TestBestellanschreibenTraegtDenVermerkDesTopfsUndNieDenAnderen(t *testing.T) {
	items := []OrderedItem{{Titel: "Mathematik 7", Autor: "Autor", ISBN: "978", Menge: 30}}
	schule := pdf.SchuleInfo{Name: "Testschule"}

	// Die Wörter, an denen die beiden Töpfe im Brief zu unterscheiden sind. Kurz
	// gehalten, weil gofpdf lange Zeilen umbricht und ein ganzer Satz dann nicht mehr
	// am Stück im Strom steht.
	kennzeichen := map[string][]string{
		repository.MittelLand:         {"Lernmittelfreiheit", "Eigentum des Landes"},
		repository.MittelSchultraeger: {"lerb", "Schultr"},
	}

	for mittel, erwartet := range kennzeichen {
		t.Run(mittel, func(t *testing.T) {
			roh, err := GenerateOrderSummaryPDF(items, schule, bogenLiegtBei, mittel)
			if err != nil {
				t.Fatalf("Anschreiben erzeugen: %v", err)
			}
			text := pdfText(t, roh)
			for _, wort := range erwartet {
				if !strings.Contains(text, wort) {
					t.Errorf("%q fehlt auf dem Anschreiben für %s", wort, mittel)
				}
			}
			// Der ANDERE Topf darf nicht vorkommen — das ist der eigentliche Fehler von
			// vorher. Für den Schulträger-Brief heißt das: „Lernmittelfreiheit" darf nur
			// in der Verneinung stehen; der Brief nennt es einmal („keine Beschaffung im
			// Rahmen der Lernmittelfreiheit"), also prüfen wir das Eigentums-Wort.
			if mittel == repository.MittelSchultraeger && strings.Contains(text, "Eigentum des Landes") {
				t.Error("Schulträger-Anschreiben behauptet Eigentum des Landes")
			}
			if mittel == repository.MittelLand && strings.Contains(text, "Schultr") {
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
		if _, err := GenerateOrderSummaryPDF(nil, pdf.SchuleInfo{}, ohneEtiketten, mittel); err == nil {
			t.Errorf("Topf %q: Anschreiben entstand ohne Vermerk", mittel)
		}
	}
}

// Die Mail-Anlagen hängen am selben Schalter: Ein BestellMail ohne Topf scheitert an der
// Anschreiben-Erzeugung, bevor irgendetwas verschickt wird.
func TestBestellAnhaengeOhneTopfScheitern(t *testing.T) {
	m := testBestellMail()
	m.Mittel = ""
	if _, err := bestellAnhaenge(m); err == nil {
		t.Error("Anlagen entstanden ohne Topf")
	}
}
