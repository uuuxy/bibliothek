package littera

import (
	"strings"
	"testing"
)

// Kopfzeilen und Werte stammen aus dem echten Export (littera_sav.mdb, 04.08.2026).
const personenCSV = `Buchungsdatum,Buchungsnummer,Name,Flags
"06/15/01 10:06:38",1,"Neebe, Reinhard","0000"
"07/10/01 15:40:54",3,"Dorn . Bader","0000"
"07/10/01 16:11:51",4,"Kuhn","0000"
"07/11/01 13:40:14",5,"Linder","0000"
,6,"","0000"
`

// Funktion 0 = Verfasser; 1 = Illustrator, 2 = Herausgeber (Personen_Funktionen).
const zuordnungCSV = `Buchungsdatum,Buchungsnummer,Titel,Person,Funktion
"07/10/01 15:40:54",3,17,4,0
"07/10/01 15:40:55",4,17,5,0
"07/10/01 15:40:56",5,17,1,2
"07/10/01 15:52:53",6,19,3,0
"07/10/01 15:52:54",7,20,1,1
"07/10/01 15:52:55",8,21,99,0
"07/10/01 15:52:56",9,22,6,0
`

func TestAutorenJeTitel(t *testing.T) {
	personen, err := LesePersonen(strings.NewReader(personenCSV))
	if err != nil {
		t.Fatalf("Personen lesen: %v", err)
	}
	autoren, err := AutorenJeTitel(personen, strings.NewReader(zuordnungCSV))
	if err != nil {
		t.Fatalf("Zuordnung lesen: %v", err)
	}

	// Zwei Verfasser, in Erfassungsreihenfolge (Buchungsnummer 3 vor 4) — NICHT
	// alphabetisch: Bei einem Schulbuch ist der Erstgenannte der Hauptverfasser.
	if autoren["17"] != "Kuhn; Linder" {
		t.Errorf("Titel 17: erwartet \"Kuhn; Linder\", war %q", autoren["17"])
	}

	// Ein Verfasser.
	if autoren["19"] != "Dorn . Bader" {
		t.Errorf("Titel 19: %q", autoren["19"])
	}

	// Titel 20 hat NUR einen Illustrator (Funktion 1) — das ist kein Verfasser.
	if _, da := autoren["20"]; da {
		t.Errorf("Illustrator darf nicht als Autor gelten, war %q", autoren["20"])
	}

	// Titel 21 zeigt auf eine Person, die es nicht gibt — kein Eintrag statt Leerstring.
	if _, da := autoren["21"]; da {
		t.Errorf("unbekannte Person darf keinen Autor erzeugen, war %q", autoren["21"])
	}

	// Titel 22 zeigt auf eine Person mit leerem Namen — ebenfalls kein Eintrag.
	if _, da := autoren["22"]; da {
		t.Errorf("leerer Personenname darf keinen Autor erzeugen, war %q", autoren["22"])
	}
}

func TestAutorenReihenfolgeIstErfassungsreihenfolge(t *testing.T) {
	personen := map[string]string{"1": "Zweiter", "2": "Erster"}
	// Buchungsnummer 10 (Person 1) kommt VOR 11 (Person 2) — obwohl "Erster"
	// alphabetisch vorne stuende.
	const zuo = `Buchungsnummer,Titel,Person,Funktion
10,5,1,0
11,5,2,0
`
	autoren, err := AutorenJeTitel(personen, strings.NewReader(zuo))
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if autoren["5"] != "Zweiter; Erster" {
		t.Errorf("Erfassungsreihenfolge erwartet (\"Zweiter; Erster\"), war %q", autoren["5"])
	}
}

func TestMitAutoren_ZuordnungGewinntUndLaesstBestehendesStehen(t *testing.T) {
	titel := []Titel{
		{ID: "1", Autor: "Freitext aus Verfasserangabe"},
		{ID: "2", Autor: "Bleibt stehen"},
		{ID: "3", Autor: ""},
	}
	autoren := map[string]string{"1": "Aufgeloest, Anna", "3": "Neu, Norbert"}

	ergebnis := MitAutoren(titel, autoren)

	// Die gepflegte Zuordnung gewinnt gegen den Freitext ...
	if ergebnis[0].Autor != "Aufgeloest, Anna" {
		t.Errorf("Zuordnung muss gewinnen, war %q", ergebnis[0].Autor)
	}
	// ... aber wo sie nichts liefert, bleibt der Freitext: besser als ein leeres Feld.
	if ergebnis[1].Autor != "Bleibt stehen" {
		t.Errorf("vorhandene Angabe darf nicht geleert werden, war %q", ergebnis[1].Autor)
	}
	if ergebnis[2].Autor != "Neu, Norbert" {
		t.Errorf("leeres Feld muss gefuellt werden, war %q", ergebnis[2].Autor)
	}

	// Die Eingabe bleibt unangetastet — MitAutoren arbeitet auf einer Kopie.
	if titel[0].Autor != "Freitext aus Verfasserangabe" {
		t.Error("MitAutoren darf die uebergebene Liste nicht veraendern")
	}
}

func TestMedienartNamen(t *testing.T) {
	const csv = `Buchungsdatum,Buchungsnummer,Medienart,KurzBez
,1,"Buch","BU"
,2,"CD-ROM","CD"
`
	namen, err := MedienartNamen(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("lesen: %v", err)
	}
	if namen["1"] != "Buch" || namen["2"] != "CD-ROM" {
		t.Errorf("Medienart-Aufloesung falsch: %v", namen)
	}
}
