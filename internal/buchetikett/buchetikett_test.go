package buchetikett

import (
	"reflect"
	"testing"

	"bibliothek/pdf"
	"bibliothek/pkg/mitteltopf"
)

// Die Erzeuger in pdf/ drucken, was PDFEtiketten ihnen reicht. Ein Feld, das hier fehlt,
// fehlt auf dem Etikett aller Druckwege, und weder Status noch Dateigröße zeigen es.

// Jedes Feld der Eingabe kommt aus seiner Quelle; die Werte sind je Feld verschieden, damit
// eine Vertauschung auffällt.
func TestPDFEtiketten_FuelltJedesFeldDerEingabe(t *testing.T) {
	kopf := Kopf{
		Schulname:                         "Schule",
		Eigentumsvermerk:                  "Vermerk Land",
		EigentumsvermerkSchuelerbuecherei: "Vermerk Bücherei",
	}
	item := Etikett{
		BarcodeID: "B-1", Titel: "Titel", Autor: "Autor", ISBN: "ISBN",
		AnschaffungsJahr: "2016", Signatur: "Signatur", Topf: mitteltopf.Land,
	}

	etiketten := PDFEtiketten([]Etikett{item}, kopf)
	if len(etiketten) != 1 {
		t.Fatalf("%d Etiketten aus einem Auftrag, erwartet 1", len(etiketten))
	}
	want := pdf.BuchEtikett{
		Schulname: "Schule", BarcodeID: "B-1", Titel: "Titel", Autor: "Autor",
		AnschaffungsJahr: "2016", Signatur: "Signatur", Eigentumsvermerk: "Vermerk Land",
	}
	if etiketten[0] != want {
		t.Errorf("Eingabe des Erzeugers = %+v, erwartet %+v", etiketten[0], want)
	}

	// Ein Feld, das pdf.BuchEtikett später dazubekommt und das hier niemand füllt, bliebe
	// leer, und der Vergleich oben sähe es nicht: Beide Seiten trügen den Nullwert.
	wert := reflect.ValueOf(etiketten[0])
	for i := 0; i < wert.NumField(); i++ {
		if wert.Field(i).IsZero() {
			t.Errorf("pdf.BuchEtikett.%s bleibt leer: PDFEtiketten füllt das Feld nicht",
				wert.Type().Field(i).Name)
		}
	}
}

// Der Vermerk folgt dem Topf des Exemplars. Ohne Topf (Exemplar unbekannt) gilt
// der allgemeine Vermerk; ein Buch der Schülerbücherei trägt ohne eigenen Vermerk keinen.
func TestPDFEtiketten_VermerkFolgtDemTopf(t *testing.T) {
	kopf := Kopf{Eigentumsvermerk: "Vermerk Land", EigentumsvermerkSchuelerbuecherei: "Vermerk Bücherei"}
	items := []Etikett{
		{BarcodeID: "B-1", Topf: mitteltopf.Land},
		{BarcodeID: "B-2", Topf: mitteltopf.Schultraeger},
		{BarcodeID: "B-3"},
	}

	etiketten := PDFEtiketten(items, kopf)
	if len(etiketten) != len(items) {
		t.Fatalf("%d Etiketten aus %d Aufträgen", len(etiketten), len(items))
	}
	for i, want := range []string{"Vermerk Land", "Vermerk Bücherei", "Vermerk Land"} {
		if etiketten[i].BarcodeID != items[i].BarcodeID {
			t.Errorf("Etikett %d trägt die Nummer %q, erwartet %q: die Reihenfolge des Auftrags bleibt",
				i, etiketten[i].BarcodeID, items[i].BarcodeID)
		}
		if etiketten[i].Eigentumsvermerk != want {
			t.Errorf("Topf %q: Vermerk %q, erwartet %q", items[i].Topf, etiketten[i].Eigentumsvermerk, want)
		}
	}

	ohne := PDFEtiketten(items, Kopf{Eigentumsvermerk: "Vermerk Land"})
	if ohne[1].Eigentumsvermerk != "" {
		t.Errorf("Schülerbücherei ohne hinterlegten Vermerk: %q gedruckt, erwartet keinen", ohne[1].Eigentumsvermerk)
	}
}
