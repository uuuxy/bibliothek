package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"bibliothek/pdf"
	"bibliothek/repository"
)

// Der Bildschirm zeigt jeden gelieferten Abschnitt als Feld mit seiner Zahl
// (frontend/src/lib/components/bestand/Bestandsbuch.svelte). Dass in einem Topf nichts
// zuging, sagt dort die Null im Feld, und die gibt es nur, wenn der Server den leeren Topf
// mitliefert. Geprüft an der Antwort, wie sie über den Draht geht: Die Oberfläche zählt
// `zeilen.length`, ein `null` an dieser Stelle bräche die Seite ab.

type abschnittAmDraht struct {
	Topf   string          `json:"topf"`
	Titel  string          `json:"titel"`
	Zeilen json.RawMessage `json:"zeilen"`
}

func zugangsbuchAmDraht(t *testing.T, zeilen []repository.ZugangsZeile) []abschnittAmDraht {
	t.Helper()
	roh, err := json.Marshal(zugangsbuchAntwort(repository.Zugangsbuch{Zeilen: zeilen}))
	if err != nil {
		t.Fatalf("Antwort schreiben: %v", err)
	}
	var antwort struct {
		Abschnitte []abschnittAmDraht `json:"abschnitte"`
	}
	if err := json.Unmarshal(roh, &antwort); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	return antwort.Abschnitte
}

func toepfe(abschnitte []abschnittAmDraht) []string {
	aus := make([]string, len(abschnitte))
	for i, a := range abschnitte {
		aus[i] = a.Topf
	}
	return aus
}

func TestBestandsbuch_BeideToepfeStehenAuchLeerInDerAntwort(t *testing.T) {
	abschnitte := zugangsbuchAmDraht(t, nil)

	erwartet := []string{repository.MittelLand, repository.MittelSchultraeger}
	if !slices.Equal(toepfe(abschnitte), erwartet) {
		t.Fatalf("Abschnitte eines leeren Buchs: %q, erwartet %q", toepfe(abschnitte), erwartet)
	}
	for _, a := range abschnitte {
		if a.Titel == "" {
			t.Errorf("Topf %q: die Überschrift fehlt, das Feld hätte keinen Namen", a.Topf)
		}
		if string(a.Zeilen) != "[]" {
			t.Errorf("Topf %q: zeilen = %s, erwartet eine leere Liste", a.Topf, a.Zeilen)
		}
	}
}

func TestBestandsbuch_OhneZuordnungNurMitZeilenUndZuletzt(t *testing.T) {
	abschnitte := zugangsbuchAmDraht(t, []repository.ZugangsZeile{
		{Barcode: "OHNE-1"},
		{Barcode: "LAND-1", Topf: repository.MittelLand},
	})

	erwartet := []string{repository.MittelLand, repository.MittelSchultraeger, ""}
	if !slices.Equal(toepfe(abschnitte), erwartet) {
		t.Fatalf("Abschnitte: %q, erwartet %q", toepfe(abschnitte), erwartet)
	}
	if ohne := abschnitte[2]; ohne.Titel != mittelOhneZuordnung {
		t.Errorf("Überschrift des dritten Abschnitts: %q, erwartet %q", ohne.Titel, mittelOhneZuordnung)
	}
	var zeilen []repository.ZugangsZeile
	if err := json.Unmarshal(abschnitte[2].Zeilen, &zeilen); err != nil {
		t.Fatalf("Zeilen lesen: %v", err)
	}
	if len(zeilen) != 1 || zeilen[0].Barcode != "OHNE-1" {
		t.Errorf("Zeilen ohne Zuordnung: %+v, erwartet nur OHNE-1", zeilen)
	}
}

// leereFelder nennt jedes Feld eines Werts, das seinen Nullwert trägt, auch in den Elementen
// einer Liste. Eine leere Liste zählt als leer.
func leereFelder(wert reflect.Value, pfad string) []string {
	switch wert.Kind() {
	case reflect.Struct:
		if _, zeit := wert.Interface().(time.Time); zeit {
			break
		}
		var leer []string
		for i := 0; i < wert.NumField(); i++ {
			leer = append(leer, leereFelder(wert.Field(i), pfad+"."+wert.Type().Field(i).Name)...)
		}
		return leer
	case reflect.Slice:
		if wert.Len() == 0 {
			return []string{pfad}
		}
		var leer []string
		for i := 0; i < wert.Len(); i++ {
			leer = append(leer, leereFelder(wert.Index(i), fmt.Sprintf("%s[%d]", pfad, i))...)
		}
		return leer
	}
	if wert.IsZero() {
		return []string{pfad}
	}
	return nil
}

// Das Blatt entsteht aus der Antwort für den Bildschirm. Jedes Feld seiner Eingabe kommt aus
// seiner Quelle, die Werte sind je Feld verschieden; ein Feld, das die Eingabe später
// dazubekommt und das niemand füllt, fällt an leereFelder auf.
func TestAbgangsbuchBlatt_FuelltJedesFeldDerEingabe(t *testing.T) {
	von := time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
	tag := time.Date(2026, time.April, 12, 10, 0, 0, 0, time.UTC)
	buch := repository.Abgangsbuch{
		Von: von, Bis: bis,
		Zeilen: []repository.AbgangsZeile{
			{Datum: tag, Barcode: "B-1", Titel: "Titel 1", Signatur: "Sig 1", Grund: "VERLUST", GrundText: "Verlust", Topf: repository.MittelSchultraeger},
			{Datum: tag.AddDate(0, 0, 1), Barcode: "B-2", Titel: "Titel 2", Signatur: "Sig 2", Grund: "AUSSORTIERT", GrundText: "Aussortiert", Topf: repository.MittelLand},
		},
		OhneZeitpunkt: 7, AusKatalogGeloescht: 3,
	}

	blatt := abgangsbuchBlatt(abgangsbuchAntwort(buch))
	want := pdf.Abgangsbuch{
		Von: von, Bis: bis,
		Abschnitte: []pdf.AbgangsAbschnitt{
			{Titel: mittelBeschriftung(repository.MittelLand), Zeilen: []pdf.AbgangsZeile{
				{Datum: tag.AddDate(0, 0, 1), Barcode: "B-2", Titel: "Titel 2", Signatur: "Sig 2", GrundText: "Aussortiert"}}},
			{Titel: mittelBeschriftung(repository.MittelSchultraeger), Zeilen: []pdf.AbgangsZeile{
				{Datum: tag, Barcode: "B-1", Titel: "Titel 1", Signatur: "Sig 1", GrundText: "Verlust"}}},
		},
		Gesamt: 2, OhneZeitpunkt: 7, AusKatalogGeloescht: 3,
	}
	if !reflect.DeepEqual(blatt, want) {
		t.Errorf("Eingabe des Blatts =\n%+v\nerwartet\n%+v", blatt, want)
	}
	if leer := leereFelder(reflect.ValueOf(blatt), "pdf.Abgangsbuch"); len(leer) > 0 {
		t.Errorf("abgangsbuchBlatt füllt diese Felder nicht: %v", leer)
	}
}

func TestZugangsbuchBlatt_FuelltJedesFeldDerEingabe(t *testing.T) {
	von := time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC)
	bis := time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC)
	tag := time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC)
	buch := repository.Zugangsbuch{
		Von: von, Bis: bis,
		Zeilen: []repository.ZugangsZeile{
			{Datum: tag, Barcode: "B-1", Titel: "Titel 1", Signatur: "Sig 1", Lieferant: "Lieferant 1"},
			{Datum: tag.AddDate(0, 0, 1), Barcode: "B-2", Titel: "Titel 2", Signatur: "Sig 2", Lieferant: "Lieferant 2", Topf: repository.MittelSchultraeger},
			{Datum: tag.AddDate(0, 0, 2), Barcode: "B-3", Titel: "Titel 3", Signatur: "Sig 3", Lieferant: "Lieferant 3", Topf: repository.MittelLand},
		},
	}

	blatt := zugangsbuchBlatt(zugangsbuchAntwort(buch))
	want := pdf.Zugangsbuch{
		Von: von, Bis: bis,
		Abschnitte: []pdf.ZugangsAbschnitt{
			{Titel: mittelBeschriftung(repository.MittelLand), Zeilen: []pdf.ZugangsZeile{
				{Datum: tag.AddDate(0, 0, 2), Barcode: "B-3", Titel: "Titel 3", Lieferant: "Lieferant 3"}}},
			{Titel: mittelBeschriftung(repository.MittelSchultraeger), Zeilen: []pdf.ZugangsZeile{
				{Datum: tag.AddDate(0, 0, 1), Barcode: "B-2", Titel: "Titel 2", Lieferant: "Lieferant 2"}}},
			{Titel: mittelOhneZuordnung, Zeilen: []pdf.ZugangsZeile{
				{Datum: tag, Barcode: "B-1", Titel: "Titel 1", Lieferant: "Lieferant 1"}}},
		},
		Gesamt: 3, OhneZuordnung: true,
	}
	if !reflect.DeepEqual(blatt, want) {
		t.Errorf("Eingabe des Blatts =\n%+v\nerwartet\n%+v", blatt, want)
	}
	if leer := leereFelder(reflect.ValueOf(blatt), "pdf.Zugangsbuch"); len(leer) > 0 {
		t.Errorf("zugangsbuchBlatt füllt diese Felder nicht: %v", leer)
	}
}

// Den Abschnitt ohne Zuordnung erklärt das Blatt nur, wenn er Zeilen trägt.
func TestZugangsbuchBlatt_OhneZuordnungNurMitSolchenZeilen(t *testing.T) {
	nurLand := zugangsbuchBlatt(zugangsbuchAntwort(repository.Zugangsbuch{
		Zeilen: []repository.ZugangsZeile{{Barcode: "LAND-1", Topf: repository.MittelLand}}}))
	if nurLand.OhneZuordnung {
		t.Error("ohne Zeilen ohne Topf verlangt die Eingabe trotzdem die Erklärung zu „ohne Zuordnung“")
	}
	if leer := zugangsbuchBlatt(zugangsbuchAntwort(repository.Zugangsbuch{})); leer.OhneZuordnung || leer.Gesamt != 0 {
		t.Errorf("leeres Buch: %+v, erwartet keine Erklärung und die Zahl 0", leer)
	}
	mit := zugangsbuchBlatt(zugangsbuchAntwort(repository.Zugangsbuch{
		Zeilen: []repository.ZugangsZeile{{Barcode: "OHNE-1"}}}))
	if !mit.OhneZuordnung {
		t.Error("mit einer Zeile ohne Topf fehlt die Erklärung zu „ohne Zuordnung“")
	}
}
