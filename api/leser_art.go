package api

import "strings"

// Die Arten eines Lesers: dieselben Werte wie chk_leser_art in der Datenbank (Migration 153)
// und dieselben Wörter wie frontend/src/lib/leserArt.js. Beide Seiten lesen die Prüffälle in
// frontend/src/lib/leserArt.faelle.json; der Server-Test hält sie zusätzlich gegen die
// Datenbank (leser_art_pg_test.go).
//
// Entschieden wird an der Art nur zweierlei: Schüler oder nicht (istSchuelerArt) und, im
// Kollegium, ob ein Zugang zu „Mein Portal" dazugehört (repository.ArtMitKonto). Alles andere
// ist Bezeichnung. Praktikum, Sekretariat, U-plus und Fachbereich sind die Sonderkonten aus
// Littera; bis zum 30.09.2026 standen sie als „Lehrkraft" da (docs/OFFEN.md 5.18).
var leserArtenListe = []struct{ art, wort string }{
	{"schueler", "Schüler"},
	{"lehrkraft", "Lehrkraft"},
	{"liv", "LiV"},
	{"praktikum", "Praktikum"},
	{"sekretariat", "Sekretariat"},
	{"uplus", "U-plus"},
	{"fachbereich", "Fachbereich"},
}

// leserArten ist die Menge aus leserArtenListe. Eine unbekannte Art ist ein Tippfehler und
// kein neuer Personenkreis: Die Datenbank wiese sie ab, aber als 500 „interner
// Datenbankfehler" statt mit einer Auskunft.
var leserArten = func() map[string]bool {
	m := make(map[string]bool, len(leserArtenListe))
	for _, a := range leserArtenListe {
		m[a.art] = true
	}
	return m
}()

// istSchuelerArt sagt, ob für diese Art die Schüler-Pflichten gelten.
func istSchuelerArt(art string) bool { return art == "schueler" }

// leserArtBezeichnung ist das Wort zur Art, wie Leserdatei und Theke es zeigen. Eine
// unbekannte Art bleibt stehen, wie sie gespeichert ist, statt still zu verschwinden.
func leserArtBezeichnung(art string) string {
	for _, a := range leserArtenListe {
		if a.art == art {
			return a.wort
		}
	}
	return art
}

// moeglicheArten zählt die Arten für eine Fehlermeldung auf: „Schüler, Lehrkraft, …".
func moeglicheArten() string {
	woerter := make([]string, 0, len(leserArtenListe))
	for _, a := range leserArtenListe {
		woerter = append(woerter, a.wort)
	}
	return strings.Join(woerter, ", ")
}
