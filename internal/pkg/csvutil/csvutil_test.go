package csvutil

import (
	"testing"
)

func TestEntschaerfeZelle(t *testing.T) {
	faelle := map[string]string{
		"":                  "",
		"Harry Potter":      "Harry Potter",
		"=HYPERLINK(\"x\")": "'=HYPERLINK(\"x\")",
		"=cmd|'/c calc'!A1": "'=cmd|'/c calc'!A1",
		"+1+1":              "'+1+1",
		"-2+3":              "'-2+3",
		"@SUM(A1)":          "'@SUM(A1)",
		"\tTab":             "'\tTab",
		"\rCarriage":        "'\rCarriage",
		"978-3-16-148410-0": "978-3-16-148410-0", // beginnt mit einer Ziffer und bleibt
		"Normaler; Titel":   "Normaler; Titel",
	}
	for zelle, want := range faelle {
		if got := entschaerfeZelle(zelle); got != want {
			t.Errorf("entschaerfeZelle(%q) = %q, erwartet %q", zelle, got, want)
		}
	}
}

func TestEntschaerfeZeile(t *testing.T) {
	got := EntschaerfeZeile([]string{"=evil", "ok", "@bad"})
	want := []string{"'=evil", "ok", "'@bad"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Zelle %d: %q, erwartet %q", i, got[i], want[i])
		}
	}
}

func TestHaeufigererTrenner(t *testing.T) {
	faelle := map[string]rune{"a;b;c": ';', "a,b,c": ',', "a;b,c": ',', "abc": ',', "a;b;c,d": ';'}
	for text, want := range faelle {
		if got := HaeufigererTrenner(text); got != want {
			t.Errorf("HaeufigererTrenner(%q) = %q, erwartet %q", text, got, want)
		}
	}
}

// Eine Liste aus Excel mit drei Spalten und Semikolon trägt in den Daten mehr Kommas als
// Semikolons: Titel mit Zusatz und Namen wie „Schurf, Bernd". Entscheiden darf nur die Kopfzeile;
// über die ganze Datei gezählt wurde die Liste als Komma-Liste gelesen.
func TestHaeufigererTrenner_NachDerKopfzeile(t *testing.T) {
	faelle := map[string]rune{
		"Titel;Autor;ISBN\nDeutschbuch 7, Neue Ausgabe, Band 1;Schurf, Bernd;9783060626755\nMathe, Band 2, Teil A;Müller, Eva;9783060000000\n": ';',
		"\n\nTitel;Autor\nA, B, C;D, E\n":             ';',
		"\"Titel, Untertitel\";ISBN\nA;1\n":           ';',
		"Title,Author,ISBN\nBook; Part 1,Smith,978\n": ',',
		"ISBN\n978;3\n978;4\n":                        ';',
	}
	for text, want := range faelle {
		if got := HaeufigererTrenner(text); got != want {
			t.Errorf("HaeufigererTrenner(%q) = %q, erwartet %q", text, got, want)
		}
	}
}
