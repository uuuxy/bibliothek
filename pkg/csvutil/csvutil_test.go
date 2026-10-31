package csvutil

import (
	"strings"
	"testing"
	"unicode/utf8"
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

func TestLeseText(t *testing.T) {
	faelle := map[string]struct{ inhalt, want string }{
		"UTF-8 bleibt":             {"Jürgen Müller", "Jürgen Müller"},
		"UTF-8 mit BOM":            {"\xef\xbb\xbfJürgen", "Jürgen"},
		"Windows-1252":             {"J\xfcrgen M\xfcller, Stra\xdfe", "Jürgen Müller, Straße"},
		"Windows-1252, Sonderfall": {"\x84Fr\xfch\x93 \x80 5", "„Früh“ € 5"},
		"nur ASCII":                {"Max;Mustermann", "Max;Mustermann"},
		"Nullzeichen, UTF-8":       {"Mü\x00ller", "Müller"},
		"Nullzeichen, 1252":        {"M\xfc\x00ller", "Müller"},
		"leer":                     {"", ""},
	}
	for name, fall := range faelle {
		if got := leseText([]byte(fall.inhalt)); got != fall.want {
			t.Errorf("%s: leseText(%q) = %q, erwartet %q", name, fall.inhalt, got, fall.want)
		}
	}
}

// Die Datenbank lehnt jede Zeile mit ungültigem UTF-8 oder einem Nullzeichen ab; in einer Datei
// kann jedes Byte stehen.
func TestLeseTextLiefertWasDieDatenbankAnnimmt(t *testing.T) {
	for b := 0; b < 256; b++ {
		got := leseText([]byte{'a', byte(b), 'z'})
		if !utf8.ValidString(got) || strings.ContainsRune(got, 0) {
			t.Errorf("Byte %#x ergibt %q", b, got)
		}
	}
}

func TestBaueLeser(t *testing.T) {
	zeile, err := BaueLeser([]byte("M\xfcller;Das \"gro\xdfe\" Buch, Teil 1;7a"), HaeufigererTrenner).Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(zeile) != 3 || zeile[0] != "Müller" || zeile[1] != "Das \"große\" Buch, Teil 1" || zeile[2] != "7a" {
		t.Errorf("Zeile = %q", zeile)
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
