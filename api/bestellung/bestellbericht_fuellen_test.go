package bestellung

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"bibliothek/internal/tuertest"
	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

// Die Tür füllt die Blöcke des Bestellberichts aus den Zeilen der Abfrage. Jedes Feld kommt aus
// seiner Quelle, die Werte sind je Feld verschieden; ein Feld, das einer der Typen des
// Erzeugers später dazubekommt und das niemand füllt, fällt an tuertest.LeereFelder auf.

func TestBestellberichtToepfe_FuelltJedesFeldAusDerZeile(t *testing.T) {
	tag := func(tag int) time.Time { return time.Date(2026, 9, tag, 10, 0, 0, 0, time.UTC) }
	// In der Folge der Abfrage, nach dem Bestelldatum: Die Töpfe stehen durcheinander.
	zeilen := []repository.BerichtBestellung{
		{ID: "id-1", LieferantName: "Lieferant 1", Kundennummer: "K-1", Bestelldatum: tag(1), Gesamtbetrag: 1050, AnzahlExemplare: 3, Mittel: "",
			Positionen: []repository.BerichtPosition{{TitelName: "Titel 1", ISBN: "isbn-1", Menge: 3, Einzelpreis: 350}}},
		{ID: "id-2", LieferantName: "Lieferant 2", Kundennummer: "K-2", Bestelldatum: tag(2), Gesamtbetrag: 4550, AnzahlExemplare: 5, Mittel: mitteltopf.Schultraeger,
			Positionen: []repository.BerichtPosition{{TitelName: "Titel 2", ISBN: "isbn-2", Menge: 5, Einzelpreis: 910}}},
		{ID: "id-3", LieferantName: "Lieferant 3", Kundennummer: "K-3", Bestelldatum: tag(3), Gesamtbetrag: 12000, AnzahlExemplare: 30, Mittel: mitteltopf.Land,
			Positionen: []repository.BerichtPosition{
				{TitelName: "Titel 3a", ISBN: "isbn-3a", Menge: 10, Einzelpreis: 400},
				{TitelName: "Titel 3b", ISBN: "isbn-3b", Menge: 20, Einzelpreis: 425},
			}},
		{ID: "id-4", LieferantName: "Lieferant 4", Kundennummer: "K-4", Bestelldatum: tag(4), Gesamtbetrag: 8000, AnzahlExemplare: 20, Mittel: mitteltopf.Land,
			Positionen: []repository.BerichtPosition{{TitelName: "Titel 4", ISBN: "isbn-4", Menge: 20, Einzelpreis: 475}}},
	}
	will := []pdf.BestellberichtTopf{
		{Beschriftung: "Lernmittelfreiheit (Land)", Bestellungen: []pdf.BestellberichtBestellung{
			{LieferantName: "Lieferant 3", Kundennummer: "K-3", Bestelldatum: tag(3), Gesamtbetrag: 12000, AnzahlExemplare: 30,
				Positionen: []pdf.BestellberichtPosition{
					{TitelName: "Titel 3a", ISBN: "isbn-3a", Menge: 10, Einzelpreis: 400},
					{TitelName: "Titel 3b", ISBN: "isbn-3b", Menge: 20, Einzelpreis: 425},
				}},
			{LieferantName: "Lieferant 4", Kundennummer: "K-4", Bestelldatum: tag(4), Gesamtbetrag: 8000, AnzahlExemplare: 20,
				Positionen: []pdf.BestellberichtPosition{{TitelName: "Titel 4", ISBN: "isbn-4", Menge: 20, Einzelpreis: 475}}},
		}},
		{Beschriftung: "Schülerbücherei (Schulträger)", Bestellungen: []pdf.BestellberichtBestellung{
			{LieferantName: "Lieferant 2", Kundennummer: "K-2", Bestelldatum: tag(2), Gesamtbetrag: 4550, AnzahlExemplare: 5,
				Positionen: []pdf.BestellberichtPosition{{TitelName: "Titel 2", ISBN: "isbn-2", Menge: 5, Einzelpreis: 910}}},
		}},
		{Beschriftung: "ohne Zuordnung", Bestellungen: []pdf.BestellberichtBestellung{
			{LieferantName: "Lieferant 1", Kundennummer: "K-1", Bestelldatum: tag(1), Gesamtbetrag: 1050, AnzahlExemplare: 3,
				Positionen: []pdf.BestellberichtPosition{{TitelName: "Titel 1", ISBN: "isbn-1", Menge: 3, Einzelpreis: 350}}},
		}},
	}
	ist, err := bestellberichtToepfe(zeilen)
	if err != nil {
		t.Fatalf("Blöcke füllen: %v", err)
	}
	if !cmp.Equal(ist, will) {
		t.Errorf("Blöcke =\n%+v\nerwartet\n%+v", ist, will)
	}
	if leer := tuertest.LeereFelder(reflect.ValueOf(ist), "Blöcke"); len(leer) > 0 {
		t.Errorf("bestellberichtToepfe füllt diese Felder nicht: %v", leer)
	}
}

// Ein Topf ohne Bestellung bekommt keinen Block: Die Lieferantenabrechnung zu einem Topf nennt
// den anderen nicht. Ohne Bestellungen gibt es keinen Block und keinen Fehler.
func TestBestellberichtToepfe_TopfOhneBestellungBekommtKeinenBlock(t *testing.T) {
	nur := func(mittel string) []repository.BerichtBestellung {
		return []repository.BerichtBestellung{{ID: "id", LieferantName: "Lieferant", Mittel: mittel}}
	}
	for mittel, beschriftung := range map[string]string{
		mitteltopf.Land:         "Lernmittelfreiheit (Land)",
		mitteltopf.Schultraeger: "Schülerbücherei (Schulträger)",
		"":                      "ohne Zuordnung",
	} {
		toepfe, err := bestellberichtToepfe(nur(mittel))
		if err != nil {
			t.Fatalf("Topf %q: %v", mittel, err)
		}
		if len(toepfe) != 1 || toepfe[0].Beschriftung != beschriftung || len(toepfe[0].Bestellungen) != 1 {
			t.Errorf("Topf %q: Blöcke %+v, erwartet genau den Block %q mit einer Bestellung", mittel, toepfe, beschriftung)
		}
	}
	toepfe, err := bestellberichtToepfe(nil)
	if err != nil || len(toepfe) != 0 {
		t.Errorf("ohne Bestellungen: Blöcke %+v, Fehler %v; erwartet keins von beiden", toepfe, err)
	}
}

// Eine Bestellung mit einem Topf, den die Reihenfolge der Berichte nicht nennt, stünde in
// keinem Block und fehlte in jeder Summe. Die Spalte lässt heute nur die zwei Töpfe und NULL
// zu; kommt ein dritter dazu und der Bericht kennt ihn nicht, entsteht kein Blatt statt eines
// unvollständigen.
func TestBestellberichtToepfe_UnbekannterTopfIstEinFehler(t *testing.T) {
	zeilen := []repository.BerichtBestellung{
		{ID: "id-1", LieferantName: "Lieferant 1", Mittel: mitteltopf.Land},
		{ID: "id-2", LieferantName: "Lieferant 2", Mittel: "kreis"},
	}
	toepfe, err := bestellberichtToepfe(zeilen)
	if err == nil {
		t.Fatalf("Blöcke %+v ohne Fehler: die Bestellung mit dem Topf „kreis“ fehlt darin still", toepfe)
	}
	if toepfe != nil {
		t.Errorf("neben dem Fehler kommen Blöcke zurück: %+v", toepfe)
	}
	if !strings.Contains(err.Error(), "1 von 2") {
		t.Errorf("die Meldung nennt nicht, wie viele Bestellungen betroffen sind: %v", err)
	}
}
