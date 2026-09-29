package repository

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Ratsche zu docs/OFFEN.md 5.10: Jeder Schlüssel neben der Kennung eines Lesers ist
// eingeordnet — die Tilgung nimmt ihn, oder er bleibt mit Grund.
//
// Viermal derselbe Fehler: LUSD-ID (6d01f27a), Ausweisnummer (131a534a), Sperrgrund
// (5b50202d), Name und Freitext der Titel-Löschspur (08406df4). Jedes Mal kam ein Wert der
// Leserzeile oder ein Freitext neben schueler_id ins Protokoll, und die Tilgung entfernte ihn
// nicht. Die Rundreise (api/dsgvo_paar_rundreise_pg_test.go) sieht nur Werte, die sie selbst
// anlegt.
//
// Detektor am Merkmal, nicht an einer Liste von Schreibern: jede Map im Nicht-Test-Code, die
// den Schlüssel "schueler_id" bekommt — als Literal, per Index-Zuweisung oder als JSON-Text
// —, mit allen Schlüsseln derselben Variable in derselben Funktion. Ob eine Map ins Protokoll
// geht, rät das Gate nicht: Auch ein Hilfsaufruf wie bescheidAudit schreibt eine. Eine Map,
// die kein Protokolleintrag ist, steht mit Ort und Grund in keinProtokoll.
//
// „getilgt" heißt: der Schlüssel steht in protokollSchluesselMitPersonenbezug
// (protokoll_personenbezug.go), aus der die Tilgung ihre Anweisungen für audit_logs und
// audit_log baut. Dass die Anweisungen die Schlüssel wirklich nehmen, zeigen die PG-Tests
// (api/titel_loeschspur_tilgung_pg_test.go, api/sperrgrund_tilgung_pg_test.go).
//
// Blind für: Schlüssel, die erst zur Laufzeit entstehen (Variable als Schlüssel); verschachtelte
// Maps (eingeordnet wird der äußere Schlüssel); Strukturen mit JSON-Tag schueler_id als
// Protokoll-Details (heute keine); Protokolleinträge, die einen Leser ohne schueler_id meinen
// (docs/OFFEN.md 5.35); generierter Code unter docs/.

// bleibtSchluessel: Schlüssel neben schueler_id, die die Tilgung stehen lässt, mit Grund.
var bleibtSchluessel = map[string]string{
	"action":          "fester Merker der Spur (titel_geloescht_mit_…), kein Wert der Person",
	"aufgeloest_id":   "Kennung der zusammengeführten, danach gelöschten Leserzeile — ein Pseudonym wie schueler_id",
	"ausgeliehen_am":  "Datum; die Ausleihspur verliert schueler_id und entleiher (Lesehistorie-Befristung, Tilgung)",
	"ausleihe_id":     "Kennung der Ausleihe, die mit dem Titel gelöscht wurde",
	"ausleihen":       "Zahl der umgehängten Ausleihen beim Zusammenführen",
	"barcode_id":      "Nummer des Exemplars, nicht des Lesers",
	"benutzer_id":     "Konto-Kennung aus der Zeit getrennter Lehrerausleihen; alle Aufrufer von LogAusleihe und LogRueckgabe übergeben heute einen leeren Wert",
	"bescheid_id":     "Kennung des Bescheids; er bleibt als Beleg, sein Empfänger wird getilgt",
	"betrag":          "Betrag der Forderung, Beleg",
	"erstellt_am":     "Datum",
	"exemplar_id":     "Kennung des Exemplars",
	"gesamtbetrag":    "Betrag des Bescheids, Beleg",
	"mittel":          "Topf des Bescheids (MittelGueltig)",
	"positionen":      "Zahl der Positionen im Bescheid",
	"referenznummer":  "Referenznummer des Bescheids; an ihr werden Zahlungen zugeordnet, der Empfänger wird getilgt",
	"schadensfall_id": "Kennung der Forderung",
	"schaeden":        "Zahl der umgehängten Forderungen beim Zusammenführen",
	"status":          "Zustand der Vormerkung (wartend, abholbereit)",
	"tabelle":         "Name der Tabelle, aus der der Bezug stammt",
	"titel":           "Buchtitel; nach der Anonymisierung hängt er an einem Pseudonym (Frist dieser Spuren: docs/OFFEN.md 5.35)",
	"vom_programm":    "Wahrheitswert: Die Sperre davor kam vom Programm",
	"von_hand":        "Wahrheitswert: Die Sperre davor kam von Hand",
	"vormerkungen":    "Zahl der umgehängten Vormerkungen beim Zusammenführen",
	"zeitpunkt":       "Zeitpunkt der Buchung",
}

// keinProtokoll: Maps mit schueler_id, die kein Protokolleintrag sind (Datei:Funktion → Grund).
// Am 29.09.2026 leer: Jede solche Map im Code geht ins Protokoll.
var keinProtokoll = map[string]string{}

type leserkennungStelle struct {
	ort        string // Datei:Funktion
	art        string // Literal, Index, JSON
	schluessel []string
}

var (
	jsonMitLeserkennung = regexp.MustCompile(`"schueler_id"\s*:`)
	jsonSchluessel      = regexp.MustCompile(`"([A-Za-z0-9_]+)"\s*:`)
)

// sammleLeserkennungStellen durchsucht den Nicht-Test-Code des Repositorys.
func sammleLeserkennungStellen(t *testing.T) []leserkennungStelle {
	t.Helper()
	wurzel := ".."
	var stellen []leserkennungStelle
	fset := token.NewFileSet()
	err := filepath.WalkDir(wurzel, func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "frontend", "vendor", "docs":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		datei, err := parser.ParseFile(fset, pfad, nil, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(wurzel, pfad)
		if err != nil {
			return err
		}
		for _, decl := range datei.Decls {
			name := "(Paketebene)"
			var knoten ast.Node = decl
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Body == nil {
					continue
				}
				name, knoten = fn.Name.Name, fn.Body
			}
			stellen = append(stellen, stellenIn(t, filepath.ToSlash(rel)+":"+name, knoten)...)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Quelltext lesen: %v", err)
	}
	return stellen
}

// stellenIn sammelt in einer Funktion (oder einer Deklaration auf Paketebene) jede Map mit
// dem Schlüssel schueler_id. Schlüssel werden je Variable zusammengeführt: das Literal der
// Zuweisung und jede spätere Index-Zuweisung.
func stellenIn(t *testing.T, ort string, knoten ast.Node) []leserkennungStelle {
	schluesselJeVariable := map[string]map[string]bool{}
	perIndex := map[string]bool{}
	gebunden := map[*ast.CompositeLit]bool{}
	merke := func(variable, schluessel string) {
		if schluesselJeVariable[variable] == nil {
			schluesselJeVariable[variable] = map[string]bool{}
		}
		schluesselJeVariable[variable][schluessel] = true
	}
	bindeLiteral := func(variable string, wert ast.Expr) {
		if cl, ok := wert.(*ast.CompositeLit); ok {
			gebunden[cl] = true
			for _, k := range literalSchluessel(t, ort, cl) {
				merke(variable, k)
			}
		}
	}

	var stellen []leserkennungStelle
	ast.Inspect(knoten, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for i, links := range x.Lhs {
				if ie, ok := links.(*ast.IndexExpr); ok {
					if id, ok := ie.X.(*ast.Ident); ok {
						if bl, ok := ie.Index.(*ast.BasicLit); ok && bl.Kind == token.STRING {
							k := entpacke(t, ort, bl)
							merke(id.Name, k)
							if k == "schueler_id" {
								perIndex[id.Name] = true
							}
						}
					}
				}
				if id, ok := links.(*ast.Ident); ok && i < len(x.Rhs) {
					bindeLiteral(id.Name, x.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			for i, id := range x.Names {
				if i < len(x.Values) {
					bindeLiteral(id.Name, x.Values[i])
				}
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				text := entpacke(t, ort, x)
				if strings.Contains(text, "jsonb_build_object") && strings.Contains(text, "'schueler_id'") {
					t.Errorf("%s: jsonb_build_object mit 'schueler_id' — diese Form liest das Gate nicht; "+
						"als Map in Go bauen oder das Gate erweitern", ort)
				}
				if jsonMitLeserkennung.MatchString(text) {
					var ks []string
					for _, m := range jsonSchluessel.FindAllStringSubmatch(text, -1) {
						ks = append(ks, m[1])
					}
					stellen = append(stellen, leserkennungStelle{ort: ort, art: "JSON", schluessel: ks})
				}
			}
		}
		return true
	})
	// Literale mit schueler_id, die an keine Variable gebunden sind (direkt übergeben).
	ast.Inspect(knoten, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok && !gebunden[cl] {
			ks := literalSchluessel(t, ort, cl)
			for _, k := range ks {
				if k == "schueler_id" {
					stellen = append(stellen, leserkennungStelle{ort: ort, art: "Literal", schluessel: ks})
					break
				}
			}
		}
		return true
	})
	for variable, ks := range schluesselJeVariable {
		if !ks["schueler_id"] {
			continue
		}
		art := "Literal"
		if perIndex[variable] {
			art = "Index"
		}
		var liste []string
		for k := range ks {
			liste = append(liste, k)
		}
		sort.Strings(liste)
		stellen = append(stellen, leserkennungStelle{ort: ort, art: art, schluessel: liste})
	}
	return stellen
}

func literalSchluessel(t *testing.T, ort string, cl *ast.CompositeLit) []string {
	var ks []string
	for _, e := range cl.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if bl, ok := kv.Key.(*ast.BasicLit); ok && bl.Kind == token.STRING {
				ks = append(ks, entpacke(t, ort, bl))
			}
		}
	}
	return ks
}

// entpacke liest eine Zeichenkette aus dem Quelltext. Scheitert das, sähe das Gate den
// Schlüssel nicht — deshalb bricht es ab, statt still weiterzulesen.
func entpacke(t *testing.T, ort string, bl *ast.BasicLit) string {
	t.Helper()
	text, err := strconv.Unquote(bl.Value)
	if err != nil {
		t.Fatalf("%s: Zeichenkette %s nicht lesbar: %v", ort, bl.Value, err)
	}
	return text
}

func getilgtSchluessel() map[string]bool {
	getilgt := map[string]bool{}
	for _, k := range protokollSchluesselMitPersonenbezug {
		getilgt[k] = true
	}
	return getilgt
}

func TestProtokollPersonenbezug_JederSchluesselNebenDerLeserkennungIstEingeordnet(t *testing.T) {
	stellen := sammleLeserkennungStellen(t)
	getilgt := getilgtSchluessel()

	// Nicht-leer je Form: Fällt ein Zweig des Detektors aus, bliebe das Gate für diese Form
	// still grün. Am 29.09.2026 gefunden und gegen die Durchsicht von Hand gehalten: 6 Literale,
	// 5 Index-Zuweisungen, 3 JSON-Texte. Die Untergrenzen liegen darunter, damit ein Umbau
	// einer Stelle das Gate nicht rot macht.
	jeArt := map[string]int{}
	for _, s := range stellen {
		jeArt[s.art]++
	}
	for art, mindestens := range map[string]int{"Literal": 4, "Index": 3, "JSON": 2} {
		if jeArt[art] < mindestens {
			t.Errorf("nur %d Stellen der Form %s gefunden, erwartet mindestens %d — der Detektor sieht diese Form nicht mehr",
				jeArt[art], art, mindestens)
		}
	}

	for _, s := range stellen {
		if _, ausgenommen := keinProtokoll[s.ort]; ausgenommen {
			continue
		}
		for _, k := range s.schluessel {
			if k == "schueler_id" || getilgt[k] {
				continue
			}
			if _, bleibt := bleibtSchluessel[k]; !bleibt {
				t.Errorf("%s (%s): Schlüssel %q neben schueler_id ist nicht eingeordnet. Trägt er einen Wert der "+
					"Leserzeile oder Freitext, gehört er in protokollSchluesselMitPersonenbezug "+
					"(repository/protokoll_personenbezug.go); sonst mit Grund in bleibtSchluessel.", s.ort, s.art, k)
			}
		}
	}
}

func TestProtokollPersonenbezug_ListenSindSauber(t *testing.T) {
	stellen := sammleLeserkennungStellen(t)
	getilgt := getilgtSchluessel()

	gesehen := map[string]bool{}
	orte := map[string]bool{}
	for _, s := range stellen {
		orte[s.ort] = true
		for _, k := range s.schluessel {
			gesehen[k] = true
		}
	}
	for k := range bleibtSchluessel {
		if getilgt[k] {
			t.Errorf("%q steht in bleibtSchluessel und in protokollSchluesselMitPersonenbezug", k)
		}
		if !gesehen[k] {
			t.Errorf("%q steht in bleibtSchluessel, aber kein Protokolleintrag mit schueler_id trägt ihn — austragen", k)
		}
	}
	for ort := range keinProtokoll {
		if !orte[ort] {
			t.Errorf("keinProtokoll nennt %s, dort steht keine Map mit schueler_id mehr — austragen", ort)
		}
	}
	// Die Schlüssel gehen als Text in die SQL-Anweisung der Tilgung (tilgePersonenbezugImProtokoll).
	nurBuchstaben := regexp.MustCompile(`^[a-z_]+$`)
	for _, k := range protokollSchluesselMitPersonenbezug {
		if !nurBuchstaben.MatchString(k) {
			t.Errorf("Schlüssel %q in protokollSchluesselMitPersonenbezug: nur Kleinbuchstaben und _", k)
		}
	}
}
