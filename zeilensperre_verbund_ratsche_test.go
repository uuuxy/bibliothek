package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Eine Zeilensperre über einen Verbund nennt die Tabelle, die sie meint (FOR UPDATE OF v).
//
// Ohne die Angabe sperrt Postgres jede Zeile des Verbunds, auch die nur mitgelesene. Mit SKIP
// LOCKED fällt dann ein Kandidat weg, weil eine fremde Buchung gerade seine Nachbarzeile hält;
// ohne SKIP LOCKED wartet die Anweisung auf sie, und zwei Wege mit verschiedener Reihenfolge
// warten aufeinander. So übersprang das Nachrücken einer Vormerkung den Ältesten der
// Warteschlange, während die Theke die Zeile dieses Schülers für eine Ausleihe hielt
// (repository/vormerkung_nachruecken_sperre_pg_test.go).
//
// Gelesen wird der Text jeder Anweisung, wie er zusammengesetzt ist: Ein Verbund im einen Stück
// und die Sperre im nächsten gehören zusammen.
//
// Reparatur bei Rot: die Tabelle nennen, deren Zeile gesperrt werden soll (`FOR UPDATE OF a`).
// Das gilt auch, wenn der Verbund nur in einer Unterabfrage steht; die Angabe schadet dort nicht.
//
// BLINDHEIT: Einen Verbund über ein Komma (`FROM a, b`) sieht der Detektor nicht, ebenso wenig
// eine Anweisung, deren Stücke über Variablen in mehreren Anweisungen des Programms
// zusammenkommen, oder eine Sperre über eine Sicht, die selbst ein Verbund ist.
var (
	sperrKlausel = regexp.MustCompile(`(?i)\bFOR\s+(NO\s+KEY\s+UPDATE|KEY\s+SHARE|UPDATE|SHARE)\b(\s+OF\b)?`)
	verbund      = regexp.MustCompile(`(?i)\bJOIN\b`)
)

// sperreUeberVerbund sagt, ob der Text eine Zeilensperre trägt, und ob eine davon über einen
// Verbund geht, ohne ihre Tabelle zu nennen.
func sperreUeberVerbund(text string) (sperrt, ohneTabelle bool) {
	for _, m := range sperrKlausel.FindAllStringSubmatch(text, -1) {
		sperrt = true
		if m[2] == "" && verbund.MatchString(text) {
			ohneTabelle = true
		}
	}
	return sperrt, ohneTabelle
}

// zusammengesetzteTexte liefert je Zeichenkette einer Datei ihren Text. Eine Verkettung zählt
// als ein Text: Literale wörtlich, jeder andere Teil als Auslassung.
func zusammengesetzteTexte(pfad string, quelle any) ([]string, error) {
	datei, err := parser.ParseFile(token.NewFileSet(), pfad, quelle, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var text func(ast.Expr) (string, bool)
	text = func(e ast.Expr) (string, bool) {
		switch x := e.(type) {
		case *ast.BasicLit:
			if x.Kind != token.STRING {
				return "…", false
			}
			wert, err := strconv.Unquote(x.Value)
			if err != nil {
				return x.Value, true
			}
			return wert, true
		case *ast.ParenExpr:
			return text(x.X)
		case *ast.BinaryExpr:
			if x.Op != token.ADD {
				return "…", false
			}
			links, l := text(x.X)
			rechts, r := text(x.Y)
			return links + rechts, l || r
		}
		return "…", false
	}
	var texte []string
	ast.Inspect(datei, func(knoten ast.Node) bool {
		switch x := knoten.(type) {
		case *ast.ImportSpec:
			return false
		case *ast.BinaryExpr:
			if t, traegtLiteral := text(x); traegtLiteral {
				texte = append(texte, t)
				return false
			}
		case *ast.BasicLit:
			if t, ok := text(x); ok {
				texte = append(texte, t)
			}
		}
		return true
	})
	return texte, nil
}

func TestZeilensperre_UeberEinenVerbundNenntIhreTabelle(t *testing.T) {
	var verstoesse []string
	sperren, mitTabelle := 0, 0
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Versteckte Ordner tragen keinen Quelltext des Programms, wohl aber Arbeitskopien.
			if name := d.Name(); name == "node_modules" || name == "frontend" || name == "testdata" ||
				(strings.HasPrefix(name, ".") && pfad != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		quelle, err := os.ReadFile(filepath.Clean(pfad))
		if err != nil {
			return err
		}
		texte, err := zusammengesetzteTexte(pfad, quelle)
		if err != nil {
			return err
		}
		for _, text := range texte {
			sperrt, ohneTabelle := sperreUeberVerbund(text)
			if !sperrt {
				continue
			}
			sperren++
			if ohneTabelle {
				verstoesse = append(verstoesse, filepath.ToSlash(pfad))
			} else if verbund.MatchString(text) {
				mitTabelle++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Nicht-leer-Garantie: Fände der Detektor kaum eine Sperre und keinen Verbund mit genannter
	// Tabelle, wäre „kein Verstoß" keine Aussage.
	if sperren < 20 || mitTabelle < 4 {
		t.Fatalf("%d Anweisungen mit Zeilensperre gelesen, %d davon über einen Verbund mit genannter Tabelle — "+
			"der Detektor misst offenbar nichts mehr", sperren, mitTabelle)
	}
	sort.Strings(verstoesse)
	for _, pfad := range verstoesse {
		t.Errorf("%s sperrt über einen Verbund, ohne die Tabelle zu nennen: Postgres sperrt dann jede Zeile "+
			"des Verbunds. Bitte `FOR UPDATE OF <Alias>` schreiben.", pfad)
	}
}

// Der Detektor kennt jede Form der Sperre und hält keine für einen Verstoß, die ihre Tabelle
// nennt oder nur eine Tabelle liest.
func TestSperreUeberVerbund_ErkenntJedeForm(t *testing.T) {
	verstoss := []string{
		"SELECT v.id FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id FOR UPDATE",
		"SELECT v.id FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id ORDER BY 1 LIMIT 1 FOR UPDATE SKIP LOCKED",
		"SELECT a.id FROM ausleihen a LEFT JOIN leser l ON l.id = a.schueler_id WHERE a.id = $1 FOR SHARE",
		"select a.id from ausleihen a join leser l on l.id = a.schueler_id for update nowait",
		"SELECT a.id FROM ausleihen a JOIN leser l ON l.id = a.schueler_id\n\t\tFOR NO KEY UPDATE",
		"SELECT a.id FROM ausleihen a JOIN leser l ON l.id = a.schueler_id FOR KEY SHARE",
		"UPDATE x SET y = 1 WHERE id = (SELECT v.id FROM v JOIN s ON s.id = v.s FOR UPDATE SKIP LOCKED)",
		// Eine Sperre nennt ihre Tabelle, die zweite nicht.
		"SELECT 1 FROM a JOIN b ON b.id = a.b FOR UPDATE OF a; SELECT 1 FROM c JOIN d ON d.id = c.d FOR UPDATE",
	}
	for _, text := range verstoss {
		if sperrt, ohneTabelle := sperreUeberVerbund(text); !sperrt || !ohneTabelle {
			t.Errorf("nicht als Verstoß erkannt: %q", text)
		}
	}
	inOrdnung := []string{
		"SELECT v.id FROM vormerkungen v JOIN schueler s ON v.schueler_id = s.id FOR UPDATE OF v SKIP LOCKED",
		"SELECT a.id FROM ausleihen a JOIN leser l ON l.id = a.schueler_id FOR NO KEY UPDATE OF a",
		"SELECT a.id FROM ausleihen a JOIN leser l ON l.id = a.schueler_id FOR UPDATE\n\t\tOF a",
		"SELECT id FROM leser WHERE id = $1 FOR UPDATE",
		"SELECT id FROM inventur_sessions WHERE id = $1 AND abgeschlossen_am IS NULL FOR SHARE",
	}
	for _, text := range inOrdnung {
		if sperrt, ohneTabelle := sperreUeberVerbund(text); !sperrt || ohneTabelle {
			t.Errorf("fälschlich als Verstoß gewertet oder nicht als Sperre erkannt: %q", text)
		}
	}
	for _, text := range []string{"SELECT a.id FROM a JOIN b ON b.id = a.b", "Bitte zuerst einen Ausweis scannen"} {
		if sperrt, _ := sperreUeberVerbund(text); sperrt {
			t.Errorf("als Sperre gewertet, was keine ist: %q", text)
		}
	}
}

// Eine Anweisung aus mehreren Stücken liest der Detektor als einen Text: Der Verbund steht im
// ersten Stück, die Sperre im letzten.
func TestZusammengesetzteTexte_VerkettungIstEinText(t *testing.T) {
	quelle := "package x\n" +
		"const bedingung = `s.deleted_at IS NULL`\n" +
		"var a = `SELECT v.id FROM v JOIN s ON s.id = v.s WHERE ` + bedingung + ` FOR UPDATE SKIP LOCKED`\n" +
		"var b = \"SELECT id FROM leser WHERE id = $1 \" + (\"FOR \" + \"UPDATE\")\n" +
		"var c = 1 + 2\n"
	texte, err := zusammengesetzteTexte("probe.go", quelle)
	if err != nil {
		t.Fatal(err)
	}
	soll := []string{
		"s.deleted_at IS NULL",
		"SELECT v.id FROM v JOIN s ON s.id = v.s WHERE … FOR UPDATE SKIP LOCKED",
		"SELECT id FROM leser WHERE id = $1 FOR UPDATE",
	}
	if strings.Join(texte, "|") != strings.Join(soll, "|") {
		t.Errorf("Texte:\n  ist  %q\n  soll %q", texte, soll)
	}
	if _, ohneTabelle := sperreUeberVerbund(texte[1]); !ohneTabelle {
		t.Error("die Sperre über den Verbund in zwei Stücken ist nicht erkannt")
	}
}
