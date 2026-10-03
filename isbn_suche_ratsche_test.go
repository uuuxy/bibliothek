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

// Eine Suche, die die ISBN als Teilstring vergleicht, findet einen Titel nicht über die andere
// Länge seiner ISBN: Die Datenbank führt eine zehnstellige mit richtigem Prüfzeichen
// dreizehnstellig (isbn_normalform, Migration 157), und beide Längen enden auf verschiedene
// Prüfzeichen. Sechs Suchabfragen hatten nur den Teilstring-Vergleich.
//
// Die Regel: Wo SQL die Spalte isbn mit LIKE oder ILIKE gegen einen Suchtext vergleicht, steht
// in derselben Deklaration auch repository.SQLSuchtextIstISBN — der Vergleich mit der
// Normalform des Suchtexts.
//
// BLINDHEIT: Liest Zeichenketten-Literale; SQL, das erst aus Sprintf-Teilen entsteht, sieht sie
// nur, soweit Spalte und LIKE in einem Literal auf einer Zeile stehen. Ob der Baustein in
// derselben Abfrage und an der richtigen Stelle steht, prüft sie nicht — das messen die
// Suchtests je Paket (isbn_suche_pg_test.go, order_search_pg_test.go,
// opac_isbn_laenge_pg_test.go). Andere Teilstring-Vergleiche (~*, position, strpos, SIMILAR TO)
// fasst sie nicht; über die ISBN gibt es im Baum keinen. Suchen, die die ISBN nur über den
// Volltext (search_vector) treffen, und Filter im Browser sieht sie nicht.
func TestISBNSuche_VergleichtDieNormalform(t *testing.T) {
	const baustein = "SQLSuchtextIstISBN"
	var verstoesse []string
	suchen := 0

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
			sucht := false
			for _, sql := range literaleIn(decl) {
				sucht = sucht || suchtUeberISBN(sql)
			}
			if !sucht {
				continue
			}
			suchen++
			if !ruft(decl, baustein) {
				ort := pfad
				if fn, ok := decl.(*ast.FuncDecl); ok {
					ort = pfad + ":" + fn.Name.Name
				}
				verstoesse = append(verstoesse, ort+": vergleicht die ISBN als Teilstring, ohne die "+
					"Normalform des Suchtexts — dazu OR repository."+baustein+"(alias, suchtext)")
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

	// Nicht-leer-Garantie, von Hand gezählt: Suchleiste und Thekensuche (repository), die
	// Titel-Verwaltung und die Lernmittel (inventur), die Bestellsuche (service), der Katalog
	// für Leser (api). Findet der Detektor weniger, misst er nichts mehr.
	if suchen < 6 {
		t.Errorf("nur %d Deklarationen mit einer Suche über die ISBN gefunden — erwartet mindestens 6: misst der Detektor noch?", suchen)
	}
	for _, form := range []string{
		"OR regexp_replace(coalesce(b.isbn, ''), '[- ]', '', 'g') ILIKE '%' || regexp_replace($1::text, '[- ]', '', 'g') || '%'",
		"OR lower(b.isbn)        LIKE '%' || lower($4::text) || '%'",
		"OR lower(coalesce(b.isbn, ''))         LIKE '%' || tokens.roh || '%'",
		"WHERE isbn ILIKE $1",
		"WHERE t.isbn like lower($2)",
		"WHERE NOT (b.isbn ILIKE $1)",
		"OR bt.isbn::text ILIKE '%' || $2 || '%'",
		"or ISBN like $1",
	} {
		if !suchtUeberISBN(form) {
			t.Errorf("Selbstprobe: Detektor fasst %q nicht", form)
		}
	}
	for _, form := range []string{
		"SELECT id FROM buecher_titel WHERE isbn LIKE 'ISBN-%' ORDER BY isbn LIMIT $1", // festes Muster, keine Suche
		"WHERE bt.titel ILIKE '%' || $3 || '%'",
		"WHERE isbn_normalform(isbn) = isbn_normalform($1)",
		"SELECT coalesce(isbn, ''), titel FROM buecher_titel WHERE titel ILIKE $1",
	} {
		if suchtUeberISBN(form) {
			t.Errorf("Selbstprobe: %q gilt als Suche über die ISBN", form)
		}
	}
}

// isbnDannLike fasst die Spalte isbn und ein LIKE oder ILIKE dahinter, im selben Ausdruck:
// dazwischen nur, was eine Funktion um die Spalte schreibt, kein OR, AND oder Zeilenende.
var isbnDannLike = regexp.MustCompile(`(?i)\bisbn\b([^\n]{0,60}?)\bI?LIKE\s+`)

// suchtUeberISBN: Vergleicht dieses SQL die Spalte isbn per LIKE oder ILIKE mit einem
// Suchtext? Ein festes Muster in Anführungszeichen ist keine Suche.
func suchtUeberISBN(sql string) bool {
	for _, m := range isbnDannLike.FindAllStringSubmatchIndex(sql, -1) {
		dazwischen := strings.ToUpper(sql[m[2]:m[3]])
		if strings.Contains(dazwischen, " OR ") || strings.Contains(dazwischen, " AND ") ||
			strings.Contains(dazwischen, " WHERE ") || strings.Contains(dazwischen, " FROM ") {
			continue
		}
		rest := sql[m[1]:]
		if !strings.HasPrefix(rest, "'") {
			return true
		}
		ende := strings.Index(rest[1:], "'")
		if ende < 0 {
			return true
		}
		if strings.HasPrefix(strings.TrimSpace(rest[ende+2:]), "||") {
			return true
		}
	}
	return false
}
