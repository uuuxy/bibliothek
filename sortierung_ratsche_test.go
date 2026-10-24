package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Listen ordnen Text nach deutscher Regel: Umlaute bei ihrem Grundbuchstaben, Zahlen nach ihrer
// Größe, Groß und klein gleich. Die Datenbank selbst ordnet nach Codepunkt („Mathe 10" vor
// „Mathe 5", „Ägypten" hinter „Zebra"); die Regel trägt die Kollation deutsch (Migration 165),
// dieselbe wie Intl.Collator('de', {numeric: true}) im Browser. Jede Ordnung nach einer
// Textspalte im SQL des Programms nennt sie deshalb: ORDER BY t.titel COLLATE deutsch.
//
// Blindheit: Liest Zeichenketten-Literale einzeln; ein ORDER BY, dessen Spalte erst aus
// Sprintf-Teilen oder einem zweiten Literal kommt, sieht sie nicht. Ob eine Spalte Text ist,
// entscheidet ihr Name (textSpalte); eine Textspalte mit anderem Namen sieht sie nicht, ebenso
// SQL in scripts/ und in den Migrationen.
func TestSortierung_TextspaltenOrdnenDeutsch(t *testing.T) {
	var verstoesse []string
	ordnungen := 0
	gesehen := map[string]bool{}

	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "testdata", "scripts":
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
		pfad = filepath.ToSlash(pfad)
		for _, decl := range datei.Decls {
			ort := pfad
			if fn, ok := decl.(*ast.FuncDecl); ok {
				ort = pfad + ":" + fn.Name.Name
			}
			for _, sql := range literaleIn(decl) {
				for _, glied := range ordnungsGlieder(sql) {
					if !ordnetNachText(glied) && !nachStelle.MatchString(glied) {
						continue
					}
					ordnungen++
					schluessel := ort + ": " + glied
					gesehen[schluessel] = true
					if _, mitGrund := ohneSprachregel[schluessel]; mitGrund {
						continue
					}
					if !strings.Contains(strings.ToLower(glied), "collate deutsch") {
						verstoesse = append(verstoesse, schluessel)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(verstoesse)
	for _, v := range verstoesse {
		t.Errorf("%s\n→ ordnet nach Codepunkt oder nach Stelle; mit COLLATE deutsch an der Spalte ordnet "+
			"es wie die Oberfläche (ORDER BY t.titel COLLATE deutsch), sonst mit Grund in ohneSprachregel", v)
	}
	for schluessel := range ohneSprachregel {
		if !gesehen[schluessel] {
			t.Errorf("ohneSprachregel nennt %q, die Ordnung gibt es so nicht mehr — Eintrag austragen", schluessel)
		}
	}

	// Nicht-leer-Garantie: Gezählt waren beim Bau 66 Glieder in 40 Dateien.
	if ordnungen < 50 {
		t.Errorf("nur %d Ordnungen nach Text gefunden, erwartet über 50: misst der Detektor noch?", ordnungen)
	}
}

// ohneSprachregel nennt Ordnungen nach einer Textspalte oder nach Stelle, die mit Absicht ohne
// die Kollation stehen, je mit Grund. Der Schlüssel ist Datei:Funktion: Glied, wie die Meldung
// ihn nennt; eine Konstante der Paketebene steht ohne Funktion.
var ohneSprachregel = map[string]string{
	"repository/dsgvo_konto.go: 7 DESC":            "Spalte 7 ist erstellt_am, ein Zeitpunkt",
	"repository/abgangsbuch.go:LadeAbgangsbuch: 6": "Spalte 6 ist der Topf als Code: „land\" vor „schultraeger\" ist die Reihenfolge des Ausdrucks",
	"repository/auflagen.go:ladeWerkeDerTitel: 1":  "Spalte 1 ist eine Kennung (werk_id), geordnet für die Reihenfolge des Sperrens",
}

// nachStelle fasst ein Glied, das nach der Stelle in der Auswahl ordnet (ORDER BY 1). Welchen
// Typ die Spalte hat, sieht der Detektor nicht; ausgeschrieben sieht er ihn am Namen.
var nachStelle = regexp.MustCompile(`(?i)^\d+(?:\s+(?:asc|desc))?(?:\s+nulls\s+(?:first|last))?$`)

// textSpalte erkennt ein Glied, das nach einer Spalte mit Text ordnet, an ihrem Namen.
var textSpalte = regexp.MustCompile(`(?i)(?:^|[^a-z_])(?:[a-z_]+\.)?(titel|untertitel|titel_name|nachname|vorname|name|class_name|autor|wort|bezeichnung|beschreibung|klasse|fach|subject|signatur|ort|standort|verlag)(?:[^a-z_]|$)`)

// ordnetNachText: Ordnet dieses Glied nach Text? Eine Umwandlung in eine Zahl oder ein Datum
// ordnet nach Wert, ein Vergleich nach wahr und falsch, eine Zählung nach ihrer Zahl.
func ordnetNachText(glied string) bool {
	klein := strings.ToLower(glied)
	for _, wert := range []string{"::int", "::integer", "::bigint", "::numeric", "::date", "::timestamp"} {
		if strings.Contains(klein, wert) {
			return false
		}
	}
	if vergleich.MatchString(klein) || strings.HasPrefix(klein, "(select count(") || strings.HasPrefix(klein, "count(") {
		return false
	}
	return textSpalte.MatchString(glied)
}

// vergleich fasst ein Glied, das nach dem Ergebnis eines Vergleichs ordnet.
var vergleich = regexp.MustCompile(`\s(?:=|<>|!=|<=|>=|<|>)\s|\sis\s+(?:not\s+)?null\b`)

// ordnungsGlieder liefert die Glieder jedes ORDER BY im SQL, einzeln und ohne Leerraum am
// Rand. Ein ORDER BY endet am Ende seiner Klammer (in einem Aggregat, einem Fenster, einer
// Unterabfrage), an LIMIT, OFFSET, FOR, FETCH, an einer Mengenoperation, an RETURNING oder am
// Ende des Texts; Kommas in Klammern trennen keine Glieder.
func ordnungsGlieder(sql string) []string {
	var glieder []string
	klein := strings.ToLower(sql)
	for start := 0; ; {
		i := ordnungsBeginn.FindStringIndex(klein[start:])
		if i == nil {
			return glieder
		}
		pos := start + i[1]
		tiefe, anfang := 0, pos
		ende := len(sql)
	zeichen:
		for j := pos; j < len(sql); j++ {
			switch c := sql[j]; {
			case c == '\'':
				if k := strings.IndexByte(sql[j+1:], '\''); k >= 0 {
					j += k + 1
				}
			case c == '(':
				tiefe++
			case c == ')':
				if tiefe == 0 {
					ende = j
					break zeichen
				}
				tiefe--
			case c == ',' && tiefe == 0:
				glieder = append(glieder, strings.Join(strings.Fields(sql[anfang:j]), " "))
				anfang = j + 1
			case c == ';' && tiefe == 0:
				ende = j
				break zeichen
			case tiefe == 0 && ordnungsEnde.MatchString(klein[j:]) && (j == 0 || !istWortzeichen(klein[j-1])):
				ende = j
				break zeichen
			}
		}
		glieder = append(glieder, strings.Join(strings.Fields(sql[anfang:ende]), " "))
		start = ende
	}
}

var (
	ordnungsBeginn = regexp.MustCompile(`\border\s+by\s`)
	ordnungsEnde   = regexp.MustCompile(`^(?:limit|offset|for|fetch|union|except|intersect|returning|window)\b`)
)

func istWortzeichen(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// Die Selbstprobe über die Formen: Glieder in Funktionen und Ausdrücken, mehrere Glieder, ein
// ORDER BY in einem Aggregat und in einer Unterabfrage, Richtung und NULLS, das Ende an LIMIT
// und an FOR UPDATE. Daneben, was nicht nach Text ordnet.
func TestSortierung_Selbstprobe(t *testing.T) {
	for sql, erwartet := range map[string][]string{
		"SELECT * FROM t ORDER BY t.titel":                                                      {"t.titel"},
		"SELECT * FROM t ORDER BY lower(t.titel) DESC, t.id LIMIT $1":                           {"lower(t.titel) DESC", "t.id"},
		"SELECT * FROM t ORDER BY coalesce(t.titel, ''), t.nachname NULLS LAST":                 {"coalesce(t.titel, '')", "t.nachname NULLS LAST"},
		"SELECT string_agg(w.wort, ', ' ORDER BY lower(w.wort)) FROM w":                         {"lower(w.wort)"},
		"SELECT * FROM (SELECT * FROM l ORDER BY klasse) x WHERE a = 1":                         {"klasse"},
		"SELECT * FROM l ORDER BY nachname COLLATE deutsch FOR UPDATE":                          {"nachname COLLATE deutsch"},
		"SELECT * FROM l\n\t\tORDER BY CASE WHEN a THEN '' ELSE s.klasse END,\n\t\t\ts.vorname": {"CASE WHEN a THEN '' ELSE s.klasse END", "s.vorname"},
	} {
		if got := ordnungsGlieder(sql); strings.Join(got, " | ") != strings.Join(erwartet, " | ") {
			t.Errorf("ordnungsGlieder(%q) = %q, erwartet %q", sql, got, erwartet)
		}
	}
	for _, glied := range []string{"t.titel", "lower(s.wort)", "coalesce(t.titel, '')", "name", "s.klasse DESC", "p.titel_name", "v.subject"} {
		if !ordnetNachText(glied) {
			t.Errorf("ordnetNachText(%q) = false, erwartet true", glied)
		}
	}
	for _, glied := range []string{"1", "7 DESC", "2 ASC NULLS LAST"} {
		if !nachStelle.MatchString(glied) {
			t.Errorf("nachStelle fasst %q nicht", glied)
		}
	}
	for _, glied := range []string{"e.titel_id", "a.ausgeliehen_am", "substring(klasse from '^\\d+')::int DESC", "count(*)", "t.id",
		"strpos(lower(s.wort), lower($1)) = 1 DESC", "(SELECT count(*) FROM tokens WHERE suchnorm(b.titel) LIKE tokens.norm) DESC",
		"t.titel IS NULL"} {
		if ordnetNachText(glied) {
			t.Errorf("ordnetNachText(%q) = true, erwartet false", glied)
		}
	}
}
