package littera

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Personen wie im echten Export: Verfasser in Katalogform und, dazwischen, was die
// Bibliothek als Standort eingetragen hat.
const standortPersonenCSV = `Buchungsdatum,Buchungsnummer,Name,Flags
"06/15/01 10:06:38",1,"Neebe, Reinhard","0000"
"06/15/01 10:06:39",2,"Bibliothek Klassensatz Regal 11","0000"
"06/15/01 10:06:40",3,"U plus","0000"
"06/15/01 10:06:41",4,"Buchan, John","0000"
"06/15/01 10:06:42",5,"Louise Carleton-Gertsch","0000"
"06/15/01 10:06:43",6,"LMF/Bibliothek","0000"
"06/15/01 10:06:44",7,"Schulseelsorge","0000"
`

// Flags: zweite Stelle = erster Verfasser, vierte Stelle = dritter Verfasser. Die Kette ist
// im Export 32 oder 31 Zeichen lang.
const (
	flagsErster  = "11000000000000000000000000000000"
	flagsDritter = "10010000000000000000000000000000"
	flagsKurz    = "1001000000000000000000000000000"
)

const standortZuordnungCSV = `Buchungsdatum,Buchungsnummer,Titel,Person,Funktion,Flags
"07/10/01 15:40:54",10,17,1,0,` + flagsErster + `
"07/10/01 15:40:55",11,17,2,0,` + flagsDritter + `
"07/10/01 15:40:56",12,18,3,0,` + flagsErster + `
"07/10/01 15:40:57",13,18,3,0,` + flagsKurz + `
"07/10/01 15:40:58",14,19,1,0,` + flagsErster + `
"07/10/01 15:40:59",15,19,4,0,` + flagsDritter + `
"07/10/01 15:41:00",16,20,5,0,` + flagsDritter + `
"07/10/01 15:41:01",17,21,6,0,` + flagsDritter + `
"07/10/01 15:41:02",18,22,7,2,` + flagsDritter + `
"07/10/01 15:41:03",19,23,2,0,` + flagsErster + `
`

func leseStandortZuordnungen(t *testing.T, regel Standortregel) Zuordnungen {
	t.Helper()
	personen, err := LesePersonen(strings.NewReader(standortPersonenCSV))
	if err != nil {
		t.Fatalf("Personen lesen: %v", err)
	}
	z, err := LeseZuordnungen(personen, strings.NewReader(standortZuordnungCSV), regel)
	if err != nil {
		t.Fatalf("Zuordnung lesen: %v", err)
	}
	return z
}

// Ein Eintrag ist Verfasser oder Standortvermerk, nie beides.
func TestStandortvermerkStehtNichtAlsVerfasserDa(t *testing.T) {
	z := leseStandortZuordnungen(t, Standortregel{})

	faelle := []struct {
		titel, autor string
		vermerke     []string
		warum        string
	}{
		{"17", "Neebe, Reinhard", []string{"Bibliothek Klassensatz Regal 11"},
			"dritte Stelle ohne Komma ist der Standort"},
		{"18", "", []string{"U plus"},
			"ein bekannter Vermerk gilt an jeder Stelle und zählt je Titel einmal"},
		{"19", "Neebe, Reinhard; Buchan, John", nil,
			"ein Name in Katalogform an der dritten Stelle bleibt Verfasser"},
		{"20", "", []string{"Louise Carleton-Gertsch"},
			"ohne Ausnahme sieht ein Verfasser ohne Komma wie ein Standort aus"},
		{"21", "", []string{"LMF/Bibliothek"}, "Vermerk für beide Orte"},
		{"22", "", nil, "ein Herausgeber ist kein Verfasser und kein Standort"},
		{"23", "Bibliothek Klassensatz Regal 11", nil,
			"derselbe Wortlaut an erster Stelle ist nicht als Standort ausgewiesen"},
	}
	for _, f := range faelle {
		if z.Autoren[f.titel] != f.autor {
			t.Errorf("Titel %s (%s): Verfasser %q, erwartet %q", f.titel, f.warum, z.Autoren[f.titel], f.autor)
		}
		if !reflect.DeepEqual(z.Standortvermerke[f.titel], f.vermerke) {
			t.Errorf("Titel %s (%s): Vermerke %q, erwartet %q", f.titel, f.warum, z.Standortvermerke[f.titel], f.vermerke)
		}
	}
	erkannt := map[string]bool{
		"Bibliothek Klassensatz Regal 11": true, "U plus": true,
		"Louise Carleton-Gertsch": true, "LMF/Bibliothek": true,
	}
	if !reflect.DeepEqual(z.VermerkNamen, erkannt) {
		t.Errorf("VermerkNamen nennt, was an einem Titel als Vermerk erkannt ist: %v", z.VermerkNamen)
	}
}

// Was der Aufrufer ausnimmt, bleibt Verfasser.
func TestKeinStandortBleibtVerfasser(t *testing.T) {
	z := leseStandortZuordnungen(t, Standortregel{KeinStandort: map[string]bool{"Louise Carleton-Gertsch": true}})

	if z.Autoren["20"] != "Louise Carleton-Gertsch" {
		t.Errorf("ausgenommener Eintrag: Verfasser %q", z.Autoren["20"])
	}
	if len(z.Standortvermerke["20"]) != 0 {
		t.Errorf("ausgenommener Eintrag steht als Standort da: %q", z.Standortvermerke["20"])
	}
	if !reflect.DeepEqual(z.Standortvermerke["17"], []string{"Bibliothek Klassensatz Regal 11"}) {
		t.Errorf("die Ausnahme hat einen anderen Vermerk mitgenommen: %q", z.Standortvermerke["17"])
	}
}

// Die freie Verfasserangabe kennt keine Stellen: Dort fällt weg, was an irgendeinem Titel als
// Vermerk erkannt ist.
func TestOhneVermerke(t *testing.T) {
	vermerke := map[string]bool{"Schulseelsorge": true, "Bibliothek Klassensatz Regal 11": true}
	faelle := map[string]string{
		"Stefan Wolf ; Schulseelsorge":    "Stefan Wolf",
		"Schulseelsorge":                  "",
		"Bibliothek Klassensatz Regal 11": "",
		"Schulseelsorger, Hans":           "Schulseelsorger, Hans",
		"":                                "",
	}
	for eingabe, erwartet := range faelle {
		if got := ohneVermerke(eingabe, vermerke); got != erwartet {
			t.Errorf("ohneVermerke(%q) = %q, erwartet %q", eingabe, got, erwartet)
		}
	}
	if got := ohneVermerke("Stefan Wolf ; Schulseelsorge", nil); got != "Stefan Wolf ; Schulseelsorge" {
		t.Errorf("ohne erkannte Vermerke bleibt die Angabe unberührt, war %q", got)
	}
}

// Der eigene Sonderstandort gewinnt gegen den Vermerk am Titel.
func TestStandortVon(t *testing.T) {
	vermerke := []string{"Klassensatz/Bibliothek"}
	faelle := []struct {
		e        Exemplar
		vermerke []string
		standort string
		vomTitel bool
	}{
		{Exemplar{Sonderstandort: "Lehrerschrank"}, vermerke, "Lehrerschrank", false},
		{Exemplar{}, vermerke, "Klassensatz/Bibliothek", true},
		{Exemplar{}, []string{"LMF", "U plus"}, "LMF; U plus", true},
		{Exemplar{}, nil, "", false},
	}
	for _, f := range faelle {
		standort, vomTitel := StandortVon(f.e, f.vermerke)
		if standort != f.standort || vomTitel != f.vomTitel {
			t.Errorf("StandortVon(%q, %q) = %q, %v — erwartet %q, %v",
				f.e.Sonderstandort, f.vermerke, standort, vomTitel, f.standort, f.vomTitel)
		}
	}
}

// Ein ausgenommener Sonderstandort kommt nicht mit, der Vermerk des Titels tritt an seine Stelle.
func TestAusgenommenerSonderstandort(t *testing.T) {
	exemplare := []Exemplar{{ID: "1", Sonderstandort: "4711 Mustermann, Max"}, {ID: "2", Sonderstandort: "Videoschrank"}}
	Standortregel{KeinStandort: map[string]bool{"4711 Mustermann, Max": true}}.wendeStandortregelAn(exemplare)

	if exemplare[0].Sonderstandort != "" || exemplare[1].Sonderstandort != "Videoschrank" {
		t.Errorf("nach der Regel: %q und %q", exemplare[0].Sonderstandort, exemplare[1].Sonderstandort)
	}
}

// Der Vermerk „LMF" am Titel macht ihn zum Lernmittel, auch ohne LMF-Signatur.
func TestVermerkMachtLernmittel(t *testing.T) {
	ab := &Altbestand{
		Signaturen: map[string]string{"1": "Ea 1 / Xyz", "2": "Ea 1 / Xyz", "3": "LMF Deu 7 / Bie", "4": "Ea 1 / Xyz"},
		Standortvermerke: map[string][]string{
			"1": {"LMF"}, "2": {"LMF/Bibliothek"}, "4": {"Filmfest"},
		},
	}
	faelle := map[string]bool{"1": true, "2": true, "3": true, "4": false, "5": false}
	for titelID, erwartet := range faelle {
		if lern, _ := ab.lernmittelFuer(titelID); lern.IstLernmittel != erwartet {
			t.Errorf("Titel %s: Lernmittel %v, erwartet %v", titelID, lern.IstLernmittel, erwartet)
		}
	}
	// Fach und Jahrgang kommen weiter aus der Signatur.
	if lern, _ := ab.lernmittelFuer("3"); lern.Fach == "" || lern.Stufe != 7 {
		t.Errorf("Titel 3: Fach %q, Stufe %d", lern.Fach, lern.Stufe)
	}
}

func TestZaehleStandorte(t *testing.T) {
	ab := &Altbestand{
		Signaturen: map[string]string{"1": "LMF Deu 7 / Bie"},
		Standortvermerke: map[string][]string{
			"1": {"LMF"}, "2": {"LMF"}, "3": {"Schulseelsorge"},
		},
		Exemplare: []Exemplar{
			{ID: "a", TitelID: "1"}, {ID: "b", TitelID: "1", Sonderstandort: "Lehrerschrank"},
			{ID: "c", TitelID: "2"}, {ID: "d", TitelID: "3"}, {ID: "e", TitelID: "4"},
			{ID: "f", TitelID: "4", Sonderstandort: "Lehrerschrank"},
		},
	}
	b := ZaehleStandorte(ab)

	if b.ExemplareEigen != 2 || b.ExemplareVomTitel != 3 || b.LernmittelNurVermerk != 1 {
		t.Errorf("eigen %d, vom Titel %d, Lernmittel nur nach Vermerk %d — erwartet 2, 3, 1",
			b.ExemplareEigen, b.ExemplareVomTitel, b.LernmittelNurVermerk)
	}
	if !reflect.DeepEqual(b.Vermerke, []Standortzahl{{"LMF", 2}, {"Schulseelsorge", 1}}) {
		t.Errorf("Vermerke: %v", b.Vermerke)
	}
	if !reflect.DeepEqual(b.Sonderstandorte, []Standortzahl{{"Lehrerschrank", 2}}) {
		t.Errorf("Sonderstandorte: %v", b.Sonderstandorte)
	}
}

func TestLeseExemplareLiestSonderstandort(t *testing.T) {
	csv := `Buchungsnummer,Titel,Exemplarnummer,Sig1,Sonderstandort,Sonderstandortkurz,Sig2
1,17,101,"Ea 1"," Videoschrank ","Videoschrank","Xyz"
2,17,102,"Ea 1","","","Xyz"
`
	exemplare, err := LeseExemplare(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("Exemplare lesen: %v", err)
	}
	if len(exemplare) != 2 || exemplare[0].Sonderstandort != "Videoschrank" || exemplare[1].Sonderstandort != "" {
		t.Errorf("Sonderstandorte: %+v", exemplare)
	}
}

// schreibeExport legt ein Verzeichnis mit allen Pflichtdateien an; was der Fall nicht nennt,
// ist eine leere Tabelle.
func schreibeExport(t *testing.T, dateien map[string]string) string {
	t.Helper()
	verzeichnis := t.TempDir()
	for _, name := range []string{
		DateiTitel, DateiExemplar, DateiVerlag, DateiMedienart, DateiPersonen, DateiPersonenZuordnung,
		DateiLeser, DateiLeserUG, DateiVerleih, DateiSchlagworte, DateiSchlagZuord,
		DateiVerweiseSchlagworte, DateiVerweisZuSchlagworte, DateiInteressenkreise, DateiIntZuMed,
	} {
		inhalt, da := dateien[name]
		if !da {
			inhalt = "Buchungsnummer\n"
		}
		if err := os.WriteFile(filepath.Join(verzeichnis, name), []byte(inhalt), 0o600); err != nil {
			t.Fatalf("%s schreiben: %v", name, err)
		}
	}
	return verzeichnis
}

// Der ganze Leseweg: Vermerke gelten je Titel, fallen aus der freien Verfasserangabe, und
// eine Ausnahme des Aufrufers trifft beide Stellen.
func TestLeseAltbestandMitStandorten(t *testing.T) {
	verzeichnis := schreibeExport(t, map[string]string{
		DateiTitel: `Buchungsnummer,Haupttitel,Verfasserangabe
17,"Klassensatz","Neebe, Reinhard ; Bibliothek Klassensatz Regal 11"
24,"Nur Freitext","Bibliothek Klassensatz Regal 11"
20,"Ohne Komma","Louise Carleton-Gertsch"
`,
		DateiExemplar: `Buchungsnummer,Titel,Exemplarnummer,Sig1,Sonderstandort,Sig2
1,17,101,"Ea 1","","Xyz"
2,17,102,"Ea 1","Lehrerschrank","Xyz"
3,24,103,"Ea 1","4711 Mustermann, Max","Xyz"
`,
		DateiPersonen:          standortPersonenCSV,
		DateiPersonenZuordnung: standortZuordnungCSV,
	})

	ab, err := LeseAltbestandMit(verzeichnis, Standortregel{KeinStandort: map[string]bool{
		"Louise Carleton-Gertsch": true, "4711 Mustermann, Max": true,
	}})
	if err != nil {
		t.Fatalf("LeseAltbestandMit: %v", err)
	}

	autor := map[string]string{}
	for _, ti := range ab.Titel {
		autor[ti.ID] = ti.Autor
	}
	if autor["17"] != "Neebe, Reinhard" {
		t.Errorf("Titel 17: Verfasser %q", autor["17"])
	}
	if autor["24"] != "" {
		t.Errorf("Titel 24: der Vermerk steht als Verfasser im Freitext: %q", autor["24"])
	}
	if autor["20"] != "Louise Carleton-Gertsch" {
		t.Errorf("Titel 20: die ausgenommene Verfasserin fehlt: %q", autor["20"])
	}
	if !reflect.DeepEqual(ab.Standortvermerke["17"], []string{"Bibliothek Klassensatz Regal 11"}) {
		t.Errorf("Titel 17: Vermerke %q", ab.Standortvermerke["17"])
	}

	standort := map[string]string{}
	for _, e := range ab.Exemplare {
		standort[e.ID], _ = StandortVon(e, ab.Standortvermerke[e.TitelID])
	}
	erwartet := map[string]string{"1": "Bibliothek Klassensatz Regal 11", "2": "Lehrerschrank", "3": ""}
	if !reflect.DeepEqual(standort, erwartet) {
		t.Errorf("Standorte der Exemplare: %v, erwartet %v", standort, erwartet)
	}
}
