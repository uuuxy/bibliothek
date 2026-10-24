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

// Auflagen eines Schulbuchs: ein Schreibpfad.
//
// Die Regeln der Auflagen (nur Lernmittel, zwei Gruppen werden eine, ein Werk mit weniger als zwei
// Titeln fällt) stehen an einer Stelle, repository/auflagen.go. Eine Tür, die daran vorbei
// schreibt, kennt sie nicht: So ließen die beiden Löschwege eines Titels die dritte Regel aus.
// Zwei Regeln, mechanisch:
//  1. SQL, das werk_id schreibt (SET-Klausel, Spaltenliste eines INSERT) oder die Tabelle
//     werke ändert, steht nur in repository/auflagen.go.
//  2. Jede Funktion mit DELETE FROM buecher_titel ruft ladeWerkeDerTitel und raeumeWerkeAuf.
//
// Blindheit: Liest Zeichenketten-Literale einzeln; SQL, das erst aus Variablen oder
// Sprintf-Teilen entsteht, sieht sie nur, soweit das Merkmal in einem Literal steht. Migrationen
// liest sie nicht; die Skripte hält TestAuflagen_SkripteHaltenDieWerke. Ob raeumeWerkeAuf nach
// dem DELETE und in derselben Transaktion läuft, prüft sie nicht; das messen repository/ und
// repository/titel_loeschen_samt_ausleihen_auflagen_pg_test.go.
func TestAuflagen_EinSchreibpfad(t *testing.T) {
	const tuer = "repository/auflagen.go"
	var verstoesse []string
	schreiberInTuer, loescher := 0, 0

	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "frontend", "testdata":
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
			fn, istFunktion := decl.(*ast.FuncDecl)
			ort := pfad
			if istFunktion {
				ort = pfad + ":" + fn.Name.Name
			}
			loescht := false
			for _, sql := range literaleIn(decl) {
				if schreibtWerk(sql) {
					if pfad == tuer {
						schreiberInTuer++
					} else {
						verstoesse = append(verstoesse, ort+": schreibt werk_id oder werke — nur über "+tuer)
					}
				}
				loescht = loescht || titelLoeschen.MatchString(sql)
			}
			if loescht {
				loescher++
				if !istFunktion || !ruft(fn.Body, "ladeWerkeDerTitel") || !ruft(fn.Body, "raeumeWerkeAuf") {
					verstoesse = append(verstoesse, ort+": löscht Titel, ohne die Werke zu halten — "+
						"zuerst repository.ladeWerkeDerTitel, nach dem DELETE raeumeWerkeAuf")
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
		t.Error(v)
	}

	// Nicht-leer-Garantie je Quelle, von Hand gezählt: In der Tür stehen
	// INSERT INTO werke, drei SET werk_id und DELETE FROM werke; Titel löschen LoescheTitel und
	// inventur.LoescheTitel. Findet der Detektor weniger, misst er nichts mehr.
	if schreiberInTuer < 5 {
		t.Errorf("nur %d Schreiber in %s gefunden — erwartet mindestens 5: misst der Detektor noch?", schreiberInTuer, tuer)
	}
	if loescher < 2 {
		t.Errorf("nur %d Funktionen mit DELETE FROM buecher_titel gefunden — erwartet mindestens 2", loescher)
	}
	for _, form := range []string{
		"UPDATE buecher_titel SET titel = $1, werk_id = $2 WHERE id = $3",
		"INSERT INTO buecher_titel (titel, werk_id) VALUES ($1, $2)",
		"INSERT INTO buecher_titel (titel) VALUES ($1) ON CONFLICT (isbn) DO UPDATE SET werk_id = EXCLUDED.werk_id",
		"DELETE FROM werke WHERE id = $1",
	} {
		if !schreibtWerk(form) {
			t.Errorf("Selbstprobe: Detektor fasst %q nicht", form)
		}
	}
	for _, form := range []string{
		"SELECT werk_id FROM buecher_titel WHERE werk_id = $1",
		"UPDATE buecher_titel SET titel = $1 WHERE werk_id = $2",
		"SELECT count(*) FROM werke",
	} {
		if schreibtWerk(form) {
			t.Errorf("Selbstprobe: Lesezugriff %q gilt als Schreiber", form)
		}
	}
}

var (
	titelLoeschen = regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+buecher_titel\b`)
	werkeAendern  = regexp.MustCompile(`(?i)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+werke\b`)
	titelSpalten  = regexp.MustCompile(`(?is)\bINSERT\s+INTO\s+buecher_titel\s*\(([^)]*)\)`)
	werkIDSetzen  = regexp.MustCompile(`(?i)\bwerk_id\s*=`)
	werkIDSpalte  = regexp.MustCompile(`(?i)\bwerk_id\b`)
)

// schreibtWerk: Schreibt dieses SQL werk_id oder ändert es die Tabelle werke? werk_id in
// einer WHERE-Bedingung ist ein Lesezugriff (zuweisungenInSet schneidet nur die SET-Klauseln).
func schreibtWerk(sql string) bool {
	if werkeAendern.MatchString(sql) {
		return true
	}
	for _, m := range titelSpalten.FindAllStringSubmatch(sql, -1) {
		if werkIDSpalte.MatchString(m[1]) {
			return true
		}
	}
	for _, set := range zuweisungenInSet(sql) {
		if werkIDSetzen.MatchString(set) {
			return true
		}
	}
	return false
}

// literaleIn liefert den Inhalt aller Zeichenketten-Literale einer Deklaration — Rohstrings
// und doppelt quotierte (LoescheTitel schreibt sein DELETE in Anführungszeichen).
func literaleIn(n ast.Node) []string {
	var out []string
	ast.Inspect(n, func(k ast.Node) bool {
		if lit, ok := k.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				out = append(out, s)
			}
		}
		return true
	})
	return out
}

// ruft: Ruft der Knoten — ein Funktionsrumpf oder eine ganze Deklaration — eine Funktion
// dieses Namens auf, direkt oder über ein Paket?
func ruft(n ast.Node, name string) bool {
	gefunden := false
	ast.Inspect(n, func(k ast.Node) bool {
		call, ok := k.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := call.Fun.(type) {
		case *ast.Ident:
			gefunden = gefunden || f.Name == name
		case *ast.SelectorExpr:
			gefunden = gefunden || f.Sel.Name == name
		}
		return true
	})
	return gefunden
}

// Skripte, die Titel löschen, halten dieselbe Regel: unter der Sperre der Auflagen die Werke der
// gelöschten Titel merken und danach jedes Werk mit weniger als zwei Titeln fallen lassen. Ein
// Skript läuft als psql < datei im Container und kann keinen gemeinsamen Baustein einbinden;
// deshalb steht der Block in jedem Skript, und hier wird er gehalten.
//
// Blindheit: prüft den Text (die Sperre mit dem Schlüssel aus repository/auflagen.go, die beiden
// Anweisungen und ihre Reihenfolge zum letzten Löschen), nicht die Wirkung. Ein Skript, das Titel
// über dynamisches SQL löscht, sieht sie nicht.
func TestAuflagen_SkripteHaltenDieWerke(t *testing.T) {
	sperre := regexp.MustCompile(`pg_advisory_xact_lock\(\s*` + auflagenSperrschluessel(t) + `\s*\)`)
	werkLoesen := regexp.MustCompile(`(?i)\bUPDATE\s+buecher_titel\s+SET\s+werk_id\s*=\s*NULL\b`)
	sqlKommentar := regexp.MustCompile(`(?m)--.*$`)
	loeschende := 0
	err := filepath.WalkDir("scripts", func(pfad string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(pfad, ".sql") {
			return err
		}
		roh, err := os.ReadFile(pfad) //nolint:gosec // Testlauf über den eigenen Quellbaum
		if err != nil {
			return err
		}
		sql := sqlKommentar.ReplaceAllString(string(roh), "")
		titel := titelLoeschen.FindAllStringIndex(sql, -1)
		if titel == nil {
			return nil
		}
		loeschende++
		erstes, letztes := titel[0][0], titel[len(titel)-1][1]
		if s := sperre.FindStringIndex(sql); s == nil || s[0] > erstes {
			t.Errorf("%s: löscht Titel ohne vorher die Sperre der Auflagen zu nehmen (pg_advisory_xact_lock wie repository/auflagen.go)", pfad)
		}
		loesen, loeschen := letzterTreffer(werkLoesen, sql), letzterTreffer(werkeAendern, sql)
		if loesen < letztes || loeschen < letztes {
			t.Errorf("%s: räumt nach dem letzten Löschen der Titel die Werke nicht auf "+
				"(UPDATE buecher_titel SET werk_id = NULL, DELETE FROM werke), wie raeumeWerkeAuf", pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Nicht-leer-Garantie: e2e_altlasten, entferne_demo_daten, seed_demo.
	if loeschende < 3 {
		t.Errorf("nur %d Skripte mit DELETE FROM buecher_titel gefunden, erwartet mindestens 3: liest der Test noch scripts/?", loeschende)
	}
}

// letzterTreffer ist der Anfang des letzten Treffers, -1 ohne Treffer.
func letzterTreffer(muster *regexp.Regexp, text string) int {
	treffer := muster.FindAllStringIndex(text, -1)
	if treffer == nil {
		return -1
	}
	return treffer[len(treffer)-1][0]
}

// auflagenSperrschluessel liest den Schlüssel der Sperre aus repository/auflagen.go, damit Skripte
// und Programm dieselbe Sperre nehmen.
func auflagenSperrschluessel(t *testing.T) string {
	t.Helper()
	datei, err := parser.ParseFile(token.NewFileSet(), "repository/auflagen.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range datei.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			wert, ok := spec.(*ast.ValueSpec)
			if !ok || len(wert.Names) != 1 || wert.Names[0].Name != "auflagenSperre" || len(wert.Values) != 1 {
				continue
			}
			if lit, ok := wert.Values[0].(*ast.BasicLit); ok {
				if n, err := strconv.ParseInt(lit.Value, 0, 64); err == nil {
					return strconv.FormatInt(n, 10)
				}
			}
		}
	}
	t.Fatal("repository/auflagen.go: Konstante auflagenSperre nicht gefunden")
	return ""
}
