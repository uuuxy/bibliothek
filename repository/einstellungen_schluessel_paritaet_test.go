package repository_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Die Schlüssel der Einstellungen: Oberfläche gegen Patch.
//
// Die Seite der Einstellungen baut ihren Patch aus benannten Schlüsseln: Jede Kategorie nennt
// in ihrer Eingabe `const eingabe = () => ({...})` die Felder, die KategorieRahmen über
// speichereKategorie schickt. Auf der anderen Seite steht
// `repository.EinstellungenPatch` mit seinen JSON-Namen. Ohne diese Prüfung verbindet beide
// nur die Aufmerksamkeit dessen, der ein Feld hinzufügt: Die Kategorie Mail schickte ihre
// Felder in einer Schreibweise, die der Typ nicht kannte, der Decoder verwarf sie, und die
// Oberfläche meldete „gespeichert". Die Tür dekodiert inzwischen streng und lehnt ab; diese
// Prüfung meldet den Fehler schon beim Bauen.
//
// Beide Richtungen zählen:
//
//   - Ein Schlüssel der Oberfläche, den der Typ nicht kennt: Der Wert kommt nie an.
//   - Ein Feld im Typ, das keine Kategorie schickt: eine Einstellung, die niemand einstellen
//     kann.
func TestEinstellungsSchluesselDeckenSichMitDemPatchStruct(t *testing.T) {
	inOberflaeche := schluesselAusKategorien(t)
	imStruct := jsonTags(repository.EinstellungenPatch{})

	for _, k := range sortiereMenge(inOberflaeche) {
		if !imStruct[k] {
			t.Errorf("die Seite der Einstellungen schickt %q, repository.EinstellungenPatch kennt "+
				"dieses Feld nicht. Die Tür lehnte das Speichern ab.", k)
		}
	}
	for _, k := range sortiereMenge(imStruct) {
		if !inOberflaeche[k] {
			t.Errorf("EinstellungenPatch trägt %q, aber keine Kategorie schickt es: "+
				"eine Einstellung, die niemand einstellen kann.", k)
		}
	}
}

// schluesselAusKategorien liest nur die Eingaben der Kategorien. Die Kategorien lesen ihre
// Felder auch (`start.theke_leeren_minuten`); ein Detektor, der beides zusammenwirft, meldet
// ein Feld weiter als versorgt, nachdem der Schreibweg entfernt wurde.
func schluesselAusKategorien(t *testing.T) map[string]bool {
	t.Helper()
	dir := filepath.Join("..", "frontend", "src", "lib", "components", "einstellungen", "kategorien")
	eintraege, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Kategorien-Verzeichnis lesen: %v", err)
	}

	// Zwei Schreibweisen kommen vor: `zahlen: [{ schluessel: 'x', … }]` für Zahlen mit
	// Untergrenze und `felder: { x: wert }` für Texte und Schalter. Ein Detektor, der nur eine
	// davon kennt, meldet die halbe Oberfläche als tot.
	ausZahlen := regexp.MustCompile(`schluessel:\s*'([a-z_]+)'`)
	ausFeldern := regexp.MustCompile(`felder:\s*\{([^}]*)\}`)
	feldName := regexp.MustCompile(`([a-z_]+)\s*:`)

	menge := map[string]bool{}
	dateien := 0
	for _, e := range eintraege {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svelte") {
			continue
		}
		roh, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s lesen: %v", e.Name(), err)
		}
		start := strings.Index(string(roh), eingabeMarke)
		if start == -1 {
			continue
		}
		dateien++
		eingabe := eingabeText(string(roh)[start+len(eingabeMarke)-1:])

		for _, m := range ausZahlen.FindAllStringSubmatch(eingabe, -1) {
			menge[m[1]] = true
		}
		for _, block := range ausFeldern.FindAllStringSubmatch(eingabe, -1) {
			for _, m := range feldName.FindAllStringSubmatch(block[1], -1) {
				menge[m[1]] = true
			}
		}
	}
	if dateien < 5 || len(menge) == 0 {
		t.Fatalf("nur %d Kategorien mit %d Schlüsseln gefunden: Der Detektor greift ins Leere",
			dateien, len(menge))
	}
	return menge
}

// eingabeMarke leitet die Eingabe einer Kategorie ein; ihre letzte Klammer öffnet das Objekt.
const eingabeMarke = "const eingabe = () => ("

// eingabeText schneidet die Eingabe ab ihrer öffnenden Klammer bis zur schließenden aus. Die
// ganze Datei zu durchsuchen wäre falsch, aus dem Grund, der über schluesselAusKategorien steht.
func eingabeText(ab string) string {
	tiefe := 0
	for i, c := range ab {
		switch c {
		case '(':
			tiefe++
		case ')':
			tiefe--
			if tiefe == 0 {
				return ab[:i]
			}
		}
	}
	return ab
}

func jsonTags(v any) map[string]bool {
	menge := map[string]bool{}
	typ := reflect.TypeOf(v)
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		menge[strings.Split(tag, ",")[0]] = true
	}
	return menge
}

func sortiereMenge(m map[string]bool) []string {
	aus := make([]string, 0, len(m))
	for k := range m {
		aus = append(aus, k)
	}
	sort.Strings(aus)
	return aus
}
