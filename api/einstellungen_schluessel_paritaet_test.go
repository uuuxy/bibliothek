package api

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

// Einstellungs-Schlüssel: Oberfläche gegen Struct.
//
// Die Einstellungsseite baut ihren Patch aus benannten Schlüsseln — jede Kategorie
// nennt in `speichereKategorie({...})` die Felder, die sie schickt. Auf der anderen
// Seite steht `repository.EinstellungenPatch` mit seinen json-Tags. Zwischen beiden gab
// es bis zum 23.08.2026 keine Verbindung außer der Aufmerksamkeit dessen, der ein Feld
// hinzufügt.
//
// Was das kostet, ist an diesem Tag zweimal passiert: Die Mail-Kategorie schickte ihre
// Felder in Unterstrich-Schreibweise, das Struct kannte sie nicht, der Decoder verwarf
// sie still — und die Oberfläche meldete "gespeichert" (Commit 488f51d9). Seit heute
// dekodiert der Endpunkt streng (400 statt stiller Verlust); dieses Gate zieht die
// Grenze eine Stufe früher, nämlich beim Bauen.
//
// Beide Richtungen zählen:
//
//   - Schlüssel in der Oberfläche, den das Struct nicht kennt → der Wert kommt nie an.
//   - Feld im Struct, das keine Kategorie schickt → eine Einstellung, die niemand
//     einstellen kann (das Gegenstück zum toten Recht `view_stats`).
func TestEinstellungsSchluesselDeckenSichMitDemPatchStruct(t *testing.T) {
	inOberflaeche := schluesselAusKategorien(t)
	imStruct := jsonTags(repository.EinstellungenPatch{})

	for _, k := range sortiereMenge(inOberflaeche) {
		if !imStruct[k] {
			t.Errorf("die Einstellungsseite schickt %q — repository.EinstellungenPatch kennt "+
				"dieses Feld nicht. Der Wert käme nie an (bzw. seit heute als 400 zurück).", k)
		}
	}
	for _, k := range sortiereMenge(imStruct) {
		if !inOberflaeche[k] {
			t.Errorf("EinstellungenPatch trägt %q, aber keine Kategorie schickt es — "+
				"eine Einstellung, die niemand einstellen kann.", k)
		}
	}
}

// schluesselAusKategorien liest NUR die speichereKategorie-Aufrufe. Die Kategorien
// LESEN ihre Felder auch (`start.theke_leeren_minuten`); ein Detektor, der beides in
// einen Topf wirft, meldet ein Feld weiterhin als versorgt, nachdem der Schreibweg
// entfernt wurde — genau daran ist der erste Entwurf des Schwester-Gates im Frontend
// gescheitert.
func schluesselAusKategorien(t *testing.T) map[string]bool {
	t.Helper()
	dir := filepath.Join("..", "frontend", "src", "lib", "components", "settings", "kategorien")
	eintraege, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Kategorien-Verzeichnis lesen: %v", err)
	}

	// Zwei Schreibweisen, beide echt: `zahlen: [{ schluessel: 'x', … }]` für Zahlen mit
	// Untergrenze und `felder: { x: wert }` für Texte und Schalter. Ein Detektor, der nur
	// eine davon kennt, meldet die halbe Oberfläche als tot — beim ersten Entwurf dieses
	// Tests genau so passiert (acht Fehlalarme).
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
		start := strings.Index(string(roh), "speichereKategorie(")
		if start == -1 {
			continue
		}
		dateien++
		aufruf := speicherAufrufText(string(roh)[start:])

		for _, m := range ausZahlen.FindAllStringSubmatch(aufruf, -1) {
			menge[m[1]] = true
		}
		for _, block := range ausFeldern.FindAllStringSubmatch(aufruf, -1) {
			for _, m := range feldName.FindAllStringSubmatch(block[1], -1) {
				menge[m[1]] = true
			}
		}
	}
	if dateien < 5 || len(menge) == 0 {
		t.Fatalf("nur %d Kategorien mit %d Schlüsseln gefunden — der Detektor greift ins Leere",
			dateien, len(menge))
	}
	return menge
}

// speicherAufrufText schneidet den Aufruf `speichereKategorie( … )` klammergenau aus.
// Die ganze Datei zu durchsuchen wäre falsch: Jede Kategorie LIEST ihre Felder auch
// (`start.theke_leeren_minuten`), und ein Detektor, der beides in einen Topf wirft,
// meldet ein Feld weiterhin als versorgt, nachdem der Schreibweg entfernt wurde.
func speicherAufrufText(ab string) string {
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
