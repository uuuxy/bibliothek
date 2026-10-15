package docs

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Gate gegen verrottete Fundstellen im Invarianten-Katalog. Eine Zeilennummer
// (`schema.sql:370`) wandert nicht mit, wenn schema.sql wächst, und weil an der genannten
// Zeile dann irgendetwas steht, merkt es beim Lesen niemand. Der Katalog nennt deshalb Namen;
// wird ein Constraint umbenannt oder entfernt, wird dieser Test rot.
//
// Reparatur bei Rot: Entweder heißt das Objekt in schema.sql anders (dann den Namen in
// invarianten.md nachziehen) oder es gibt es nicht mehr (dann gehört die Invariante
// überprüft — sie ist womöglich von 🟢 auf 🟡 gefallen).
func TestInvariantenFundstellenExistieren(t *testing.T) {
	katalog, err := os.ReadFile("invarianten.md")
	if err != nil {
		t.Fatalf("invarianten.md lesen: %v", err)
	}
	schema, err := os.ReadFile("../schema.sql")
	if err != nil {
		t.Fatalf("schema.sql lesen: %v", err)
	}
	schemaText := string(schema)

	bezeichner := sammleSchemaBezeichner(string(katalog))

	// Sanity-Floor: Findet der Scanner (durch ein geändertes Tabellenformat o. Ä.)
	// plötzlich fast nichts, ist das Gate faktisch abgeschaltet — dann lieber laut
	// scheitern als still grün. Aktuell sind es ~30 Bezeichner.
	if len(bezeichner) < 20 {
		t.Fatalf("nur %d Schema-Bezeichner in invarianten.md erkannt — der Scanner greift "+
			"vermutlich nicht mehr (erwartet >20). Tabellenformat geändert?", len(bezeichner))
	}

	for _, name := range bezeichner {
		if !kommtVorAlsWort(schemaText, name) {
			t.Errorf("invarianten.md nennt %q als Fundstelle, in schema.sql gibt es das nicht.\n"+
				"→ Entweder heißt das Objekt inzwischen anders (Namen nachziehen) oder es ist "+
				"entfallen (dann stimmt die Durchsetzungsebene der Invariante nicht mehr).", name)
		}
	}
}

// fundstelleSpalte greift die letzte Spalte einer Markdown-Tabellenzeile ab.
var fundstelleSpalte = regexp.MustCompile(`(?m)^\|.*\|([^|]*)\|\s*$`)

// backtickToken liest die in Backticks gesetzten Bezeichner einer Zelle.
var backtickToken = regexp.MustCompile("`([^`]+)`")

// sammleSchemaBezeichner liest aus der Fundstellen-Spalte alle Namen, die ein Objekt in
// schema.sql bezeichnen sollen. Dateipfade (`internal/lusd/anwenden.go`, `migrations/042`) gehören
// nicht dazu — sie zeigen bewusst woandershin und werden hier übersprungen.
func sammleSchemaBezeichner(katalog string) []string {
	gesehen := map[string]bool{}
	var namen []string

	for _, zeile := range fundstelleSpalte.FindAllStringSubmatch(katalog, -1) {
		for _, treffer := range backtickToken.FindAllStringSubmatch(zeile[1], -1) {
			for _, name := range zerlegeBezeichner(treffer[1]) {
				if !gesehen[name] {
					gesehen[name] = true
					namen = append(namen, name)
				}
			}
		}
	}
	return namen
}

// istDateiverweis erkennt Zellen, die bewusst woandershin zeigen: Quelldateien
// (`internal/service/ausleihe_am_leser.go`), Verzeichnisse (`migrations/042`) und
// Frontend-Pfade. Sie werden als GANZES übersprungen — würde man sie an `/` und `.`
// zerlegen, entstünden Scheinbezeichner wie "internal" oder "service".
var istDateiverweis = regexp.MustCompile(`/|\.(go|js|svelte|sql|sh|md|yml|json|properties)$`)

// schemaBezeichner beschreibt, wie Objekte in schema.sql heißen: kleingeschrieben,
// Unterstriche, mindestens drei Zeichen. Alles andere (Prosa, "XOR", Zahlen) ist
// keine Fundstelle.
var schemaBezeichner = regexp.MustCompile(`^[a-z][a-z0-9_]{2,}$`)

// zerlegeBezeichner macht aus einer Zelle wie `vormerkungen (titel_id, schueler_id)` oder
// `schueler.barcode_id` die einzelnen zu prüfenden Namen.
func zerlegeBezeichner(roh string) []string {
	roh = strings.TrimSpace(roh)
	if roh == "" || istDateiverweis.MatchString(roh) {
		return nil
	}

	roh = strings.NewReplacer("(", " ", ")", " ", ",", " ", ".", " ").Replace(roh)

	var namen []string
	for _, name := range strings.Fields(roh) {
		if schemaBezeichner.MatchString(name) {
			namen = append(namen, name)
		}
	}
	return namen
}

// kommtVorAlsWort prüft, ob der Name in schema.sql als eigenständiges Wort auftaucht —
// nicht als Teilstück eines längeren Bezeichners.
func kommtVorAlsWort(text, name string) bool {
	muster := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	return muster.MatchString(text)
}
