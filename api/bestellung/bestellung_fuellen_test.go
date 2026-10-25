package bestellung

import (
	"reflect"
	"testing"
	"time"

	"bibliothek/internal/buchetikett"
	"bibliothek/internal/service"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/mitteltopf"

	"github.com/google/go-cmp/cmp"
)

// Die Tür füllt den Auftrag an das Anlegen der Bestellung aus der Anfrage und die Angaben der
// Bestellmail aus dem Ergebnis. Jedes Feld kommt aus seiner Quelle, die Werte sind je Feld
// verschieden; ein Feld, das einer der Typen später dazubekommt und das niemand füllt, fällt an
// tuertest.LeereFelder auf.

func TestBestellAuftrag_FuelltJedesFeldAusDerAnfrage(t *testing.T) {
	anfrage := bestellungAnlegenAnfrage{
		LieferantID:          "lieferant-1",
		IdempotenzSchluessel: "schluessel-1",
		Mittel:               mitteltopf.Schultraeger,
		Positionen: []bestellPosition{
			{TitelID: "titel-1", Menge: 3, Preis: 950, MitVorabBarcode: true},
			{TitelID: "titel-2", Menge: 7, Preis: 1225, MitVorabBarcode: true},
		},
	}
	will := service.BestellAuftrag{
		LieferantID:          "lieferant-1",
		IdempotenzSchluessel: "schluessel-1",
		Mittel:               mitteltopf.Schultraeger,
		Positionen: []service.BestellAuftragPosition{
			{TitelID: "titel-1", Menge: 3, Preis: 950, MitVorabBarcode: true},
			{TitelID: "titel-2", Menge: 7, Preis: 1225, MitVorabBarcode: true},
		},
	}
	ist := bestellAuftrag(anfrage)
	if !cmp.Equal(ist, will) {
		t.Errorf("Auftrag =\n%+v\nerwartet\n%+v", ist, will)
	}
	if leer := tuertest.LeereFelder(reflect.ValueOf(ist), "service.BestellAuftrag"); len(leer) > 0 {
		t.Errorf("bestellAuftrag füllt diese Felder nicht: %v", leer)
	}
	// Eine Position ohne Vorab-Barcode bleibt ohne: Der Händler bekäme sonst Etiketten für
	// Exemplare, die ohne bestellt sind.
	anfrage.Positionen[1].MitVorabBarcode = false
	if bestellAuftrag(anfrage).Positionen[1].MitVorabBarcode {
		t.Error("eine Position ohne Vorab-Barcode kommt mit Vorab-Barcode im Auftrag an")
	}
}

func TestBestellmailDatenAus_FuelltJedesFeldAusDerBestellung(t *testing.T) {
	ablauf := time.Date(2027, time.March, 4, 12, 0, 0, 0, time.UTC)
	ergebnis := &service.BestellErgebnis{
		LieferantName:      "Lieferant",
		LieferantEmail:     "haendler@example.invalid",
		Kundennummer:       "K-7",
		Menge:              5,
		IstHauptlieferant:  true,
		BestellungID:       "bestellung-1",
		BestaetigungsToken: "token-1",
		LinkGueltigBis:     &ablauf,
		Mittel:             mitteltopf.Schultraeger,
		BestellteTitel: []service.BestellterTitel{
			{Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", Verlag: "Verlag 1", Menge: 3},
			{Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", Verlag: "Verlag 2", Menge: 2},
		},
		Etiketten: []service.BestellEtikett{
			{BarcodeID: "B-1", Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", AnschaffungsJahr: "2027", Signatur: "Sig 1", Topf: mitteltopf.Schultraeger},
			{BarcodeID: "B-2", Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", AnschaffungsJahr: "2026", Signatur: "Sig 2", Topf: mitteltopf.Land},
		},
	}
	will := bestellmailDaten{
		Empfaenger:        "haendler@example.invalid",
		Kundennummer:      "K-7",
		Mittel:            mitteltopf.Schultraeger,
		Exemplare:         5,
		IstHauptlieferant: true,
		Token:             "token-1",
		LinkGueltigBis:    &ablauf,
		Positionen: []bestellterTitel{
			{Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", Verlag: "Verlag 1", Menge: 3},
			{Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", Verlag: "Verlag 2", Menge: 2},
		},
		Etiketten: []buchetikett.Etikett{
			{BarcodeID: "B-1", Titel: "Titel 1", Autor: "Autor 1", ISBN: "isbn-1", AnschaffungsJahr: "2027", Signatur: "Sig 1", Topf: mitteltopf.Schultraeger},
			{BarcodeID: "B-2", Titel: "Titel 2", Autor: "Autor 2", ISBN: "isbn-2", AnschaffungsJahr: "2026", Signatur: "Sig 2", Topf: mitteltopf.Land},
		},
	}
	ist := bestellmailDatenAus(ergebnis)
	if !cmp.Equal(ist, will) {
		t.Errorf("Angaben der Mail =\n%+v\nerwartet\n%+v", ist, will)
	}
	if leer := tuertest.LeereFelder(reflect.ValueOf(ist), "bestellmailDaten"); len(leer) > 0 {
		t.Errorf("bestellmailDatenAus füllt diese Felder nicht: %v", leer)
	}
}
