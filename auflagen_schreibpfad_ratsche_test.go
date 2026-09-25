package main

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

// Auflagen eines Schulbuchs: EIN Schreibpfad (Migration 148, docs/OFFEN.md 4.18).
//
// Die Regeln der Auflagen — nur Lernmittel, zwei Gruppen werden eine, ein Werk mit weniger
// als zwei Titeln fällt — stehen an einer Stelle, repository/auflagen.go. Das behaupteten
// die Migration, FACHKONZEPT, arc42 und zwei Kommentare; geprüft hat es nichts. Beim
// Rasterdurchgang über 4.18 (25.09.2026) lag die Lücke an einer Tür, die niemand im Blick
// hatte: Die beiden Löschwege eines Titels kannten die dritte Regel nicht (c54e368c). Zwei
// Regeln, mechanisch:
//  1. SQL, das werk_id schreibt (SET-Klausel, Spaltenliste eines INSERT) oder die Tabelle
//     werke ändert, steht nur in repository/auflagen.go.
//  2. Jede Funktion mit DELETE FROM buecher_titel ruft WerkeDerTitel und RaeumeWerkeAuf.
//
// BLINDHEIT: Liest Zeichenketten-Literale einzeln — SQL, das erst aus Variablen oder
// Sprintf-Teilen entsteht, sieht sie nur, soweit das Merkmal in einem Literal steht. Skripte
// (scripts/*.sql) und Migrationen liest sie nicht; die Skripte, die Titel löschen, stehen in
// docs/OFFEN.md 5.26. Ob RaeumeWerkeAuf NACH dem DELETE und in derselben Transaktion läuft,
// prüft sie nicht — das messen repository/ und inventur/auflagen_loeschen_pg_test.go.
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
				if !istFunktion || !ruft(fn.Body, "WerkeDerTitel") || !ruft(fn.Body, "RaeumeWerkeAuf") {
					verstoesse = append(verstoesse, ort+": löscht Titel, ohne die Werke zu halten — "+
						"zuerst repository.WerkeDerTitel, nach dem DELETE RaeumeWerkeAuf")
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

	// Nicht-leer-Garantie je Quelle, von Hand gezählt am 25.09.2026: In der Tür stehen
	// INSERT INTO werke, drei SET werk_id und DELETE FROM werke; Titel löschen DeleteTitle und
	// DeleteBooks. Findet der Detektor weniger, misst er nichts mehr.
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
// einer WHERE-Bedingung ist ein Lesezugriff (setKlauseln schneidet nur die SET-Klauseln).
func schreibtWerk(sql string) bool {
	if werkeAendern.MatchString(sql) {
		return true
	}
	for _, m := range titelSpalten.FindAllStringSubmatch(sql, -1) {
		if werkIDSpalte.MatchString(m[1]) {
			return true
		}
	}
	for _, set := range setKlauseln(sql) {
		if werkIDSetzen.MatchString(set) {
			return true
		}
	}
	return false
}

// literaleIn liefert den Inhalt aller Zeichenketten-Literale einer Deklaration — Rohstrings
// und doppelt quotierte (DeleteTitle schreibt sein DELETE in Anführungszeichen).
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

// ruft: Ruft der Rumpf eine Funktion dieses Namens auf, direkt oder über ein Paket?
func ruft(body *ast.BlockStmt, name string) bool {
	gefunden := false
	ast.Inspect(body, func(k ast.Node) bool {
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
