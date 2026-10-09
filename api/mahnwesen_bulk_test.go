package api

import (
	"reflect"
	"testing"
	"time"

	"bibliothek/pdf"
	"bibliothek/repository"
)

// Der Mahnbrief entsteht aus den Zeilen der Abfrage. Jedes Feld seiner Eingabe kommt aus seiner
// Quelle, die Werte sind je Feld verschieden; ein Feld, das die Eingabe später dazubekommt und
// das niemand füllt, fällt an leereFelder auf.
func TestMahnbriefEmpfaenger_FuelltJedesFeldDerEingabe(t *testing.T) {
	ausgeliehen := time.Date(2026, time.March, 3, 9, 0, 0, 0, time.UTC)
	frist := time.Date(2026, time.May, 4, 21, 59, 59, 0, time.UTC)
	briefe := []repository.MahnbriefEmpfaenger{
		{SchuelerID: "s-1", Vorname: "Mia", Nachname: "Musterkind", Strasse: "Blumenweg", Hausnummer: "7", PLZ: "61169", Ort: "Friedberg",
			Buecher: []repository.MahnbriefBuch{
				{Titel: "Titel 1", Barcode: "B-1", AusgeliehenAm: ausgeliehen, Frist: frist, TageUeberfaellig: 9},
				{Titel: "Titel 2", Barcode: "B-2", AusgeliehenAm: ausgeliehen.AddDate(0, 0, 1), Frist: frist.AddDate(0, 0, 2), TageUeberfaellig: 7},
			}},
		{SchuelerID: "s-2", Vorname: "Ben", Nachname: "Birne", Strasse: "Am Hang", Hausnummer: "12b", PLZ: "61381", Ort: "Friedrichsdorf",
			Buecher: []repository.MahnbriefBuch{
				{Titel: "Titel 3", Barcode: "B-3", AusgeliehenAm: ausgeliehen.AddDate(0, 0, 3), Frist: frist.AddDate(0, 0, 4), TageUeberfaellig: 5},
			}},
	}

	eingabe := mahnbriefEmpfaenger(briefe)
	want := []pdf.MahnbriefEmpfaenger{
		{Vorname: "Mia", Nachname: "Musterkind", Strasse: "Blumenweg", Hausnummer: "7", PLZ: "61169", Ort: "Friedberg",
			Buecher: []pdf.MahnbriefBuch{
				{Titel: "Titel 1", Barcode: "B-1", AusgeliehenAm: ausgeliehen, Frist: frist, TageUeberfaellig: 9},
				{Titel: "Titel 2", Barcode: "B-2", AusgeliehenAm: ausgeliehen.AddDate(0, 0, 1), Frist: frist.AddDate(0, 0, 2), TageUeberfaellig: 7},
			}},
		{Vorname: "Ben", Nachname: "Birne", Strasse: "Am Hang", Hausnummer: "12b", PLZ: "61381", Ort: "Friedrichsdorf",
			Buecher: []pdf.MahnbriefBuch{
				{Titel: "Titel 3", Barcode: "B-3", AusgeliehenAm: ausgeliehen.AddDate(0, 0, 3), Frist: frist.AddDate(0, 0, 4), TageUeberfaellig: 5},
			}},
	}
	if !reflect.DeepEqual(eingabe, want) {
		t.Errorf("Eingabe des Mahnbriefs =\n%+v\nerwartet\n%+v", eingabe, want)
	}
	if leer := leereFelder(reflect.ValueOf(eingabe), "[]pdf.MahnbriefEmpfaenger"); len(leer) > 0 {
		t.Errorf("mahnbriefEmpfaenger füllt diese Felder nicht: %v", leer)
	}
}

// Ohne Zeilen gibt es keinen Brief.
func TestMahnbriefEmpfaenger_OhneZeilenKeinBrief(t *testing.T) {
	if eingabe := mahnbriefEmpfaenger(nil); len(eingabe) != 0 {
		t.Errorf("%d Briefe ohne Zeilen, erwartet keinen", len(eingabe))
	}
}
