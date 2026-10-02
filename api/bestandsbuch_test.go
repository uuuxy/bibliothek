package api

import (
	"encoding/json"
	"slices"
	"testing"

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
