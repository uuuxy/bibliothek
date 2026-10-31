package csvutil

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// Die Bytes, die Excel 16.114 für Mac unter „CSV“ schreibt, aus einer echten Datei; unter den
// Formaten „Windows“ und „MS-DOS“ schreibt es dieselben. „CSV (Macintosh)“ trennt nur mit CR.
const (
	ausExcelFuerMac = "nachname;vorname;titel\r\nM\x9fller;J\x9frgen;Gr\x9a\xa7e der Stra\xa7e\r\n" +
		"B\x8acker;\x80neas \x85l \x86bung;\xe3Fr\x9fh\xd2 kostet 5 \xdb"
	ausExcelFuerMacNurCR = "nachname;vorname;titel\rM\x9fller;J\x9frgen;Gr\x9a\xa7e der Stra\xa7e\r" +
		"B\x8acker;\x80neas \x85l \x86bung;\xe3Fr\x9fh\xd2 kostet 5 \xdb"
)

func TestLeseText(t *testing.T) {
	faelle := map[string]struct{ inhalt, want string }{
		"UTF-8 bleibt":                           {"Jürgen Müller", "Jürgen Müller"},
		"UTF-8 mit BOM":                          {"\xef\xbb\xbfJürgen", "Jürgen"},
		"Windows-1252":                           {"J\xfcrgen M\xfcller, Stra\xdfe", "Jürgen Müller, Straße"},
		"Windows-1252, Sonderfall":               {"\x84Fr\xfch\x93 \x80 5", "„Früh“ € 5"},
		"Windows-1252, Umlaut am Wortrand klärt": {"\xdcbung, Kre\x9aimir", "Übung, Krešimir"},
		"Windows-1252, § am Wortrand":            {"M\xfcller, Abs\xa7 5", "Müller, Abs§ 5"},
		"Windows-1252, § allein":                 {"\xa7 7", "§ 7"},
		"Windows-1252, Umlaut am Wortrand":       {"\xdcbung, Spa\xdf", "Übung, Spaß"},
		"Windows-1252, Umlaut klärt die Zeichen": {"M\xfcller, Kre\x9aimir, S\xe1nchez, \x85und",
			"Müller, Krešimir, Sánchez, …und"},
		"Windows-1252 ohne Umlaut": {"\x84Harry Potter\x94 \x96 O\x92Connor, Ren\xe9, 5 \x80",
			"„Harry Potter” – O’Connor, René, 5 €"},
		"Excel für Mac":                {ausExcelFuerMac, "nachname;vorname;titel\r\nMüller;Jürgen;Größe der Straße\r\nBäcker;Äneas Öl Übung;„Früh“ kostet 5 €"},
		"Excel für Mac, CR":            {ausExcelFuerMacNurCR, "nachname;vorname;titel\nMüller;Jürgen;Größe der Straße\nBäcker;Äneas Öl Übung;„Früh“ kostet 5 €"},
		"Excel für Mac, nur ü":         {"M\x9fller", "Müller"},
		"Excel für Mac, nur ß":         {"Stra\xa7e", "Straße"},
		"Excel für Mac, ü am Wortrand": {"\x9fber", "über"},
		"MS-DOS":                       {"M\x81ller;B\x84cker;K\x94ln;Gr\x94\xe1e;\x8epfel;\x99l;\x9abung", "Müller;Bäcker;Köln;Größe;Äpfel;Öl;Übung"},
		"MS-DOS, ü am Wortrand":        {"Men\x81", "Menü"},
		"MS-DOS, nur ä":                {"M\x84rz", "März"},
		"MS-DOS, nur ö":                {"K\x94ln", "Köln"},
		"nur ASCII":                    {"Max;Mustermann", "Max;Mustermann"},
		"nur CR":                       {"a;b\rc;d\r", "a;b\nc;d\n"},
		"CR und LF bleiben":            {"a;b\r\nc;d\r\n", "a;b\r\nc;d\r\n"},
		"Nullzeichen, UTF-8":           {"Mü\x00ller", "Müller"},
		"Nullzeichen, 1252":            {"M\xfc\x00ller", "Müller"},
		"leer":                         {"", ""},
	}
	for name, fall := range faelle {
		got, err := leseText([]byte(fall.inhalt))
		if err != nil {
			t.Errorf("%s: leseText(%q): %v", name, fall.inhalt, err)
			continue
		}
		if got != fall.want {
			t.Errorf("%s: leseText(%q) = %q, erwartet %q", name, fall.inhalt, got, fall.want)
		}
	}
}

// Passt nicht genau eine Kodierung, käme mit jeder Wahl ein Teil der Datei falsch an.
func TestLeseTextLehntUnklareDateiAb(t *testing.T) {
	faelle := map[string]string{
		"š, Windows-1252 oder ein ö vom Mac":         "Kre\x9aimir",
		"Š, Windows-1252 oder ein ä vom Mac":         "B\x8acker",
		"á, Windows-1252 oder ein ß aus MS-DOS":      "Stra\xe1e",
		"Ž, Windows-1252 oder ein Ä aus MS-DOS":      "\x8epfel",
		"…, Windows-1252 oder ein Ö vom Mac":         "\x85l",
		"€, Windows-1252 oder ein Ä vom Mac":         "\x80pfel",
		"†, Windows-1252 oder ein Ü vom Mac":         "\x86bung",
		"™, Windows-1252 oder ein Ö aus MS-DOS":      "\x99l",
		"§ am Wort, Windows-1252 oder ein ß vom Mac": "Spa\xa7 5",
		"in Windows-1252 nicht belegt":               "a \x8d b",
		"nicht belegt, daneben ein Umlaut":           "M\xfcller a \x8d b",
		"UTF-8 und Windows-1252 gemischt":            "M\xc3\xbcller, J\xfcrgen",
		"UTF-8 mit einem fremden Byte":               "M\xc3\xbcller \xff",
		"Windows-1252 und Macintosh":                 "M\xfcller, M\x9fller",
		"Windows-1252 und MS-DOS":                    "M\xfcller, M\x84rz",
		"Macintosh und MS-DOS":                       "M\x9fller, M\x84rz",
	}
	for name, inhalt := range faelle {
		if got, err := leseText([]byte(inhalt)); !errors.Is(err, ErrKodierung) {
			t.Errorf("%s: leseText(%q) = %q, %v; erwartet ErrKodierung", name, inhalt, got, err)
		}
	}
}

// Die Datenbank lehnt jede Zeile mit ungültigem UTF-8 oder einem Nullzeichen ab, und ein Steuerzeichen
// aus einem unbelegten Byte fiele niemandem auf; in einer Datei kann jedes Byte stehen.
func TestLeseTextLiefertWasDieDatenbankAnnimmt(t *testing.T) {
	abgelehnt := 0
	for b := 0; b < 256; b++ {
		got, err := leseText([]byte{'a', byte(b), 'z'})
		if errors.Is(err, ErrKodierung) {
			abgelehnt++
			continue
		}
		if err != nil {
			t.Fatalf("Byte %#x: %v", b, err)
		}
		if !utf8.ValidString(got) || strings.ContainsRune(got, 0) || strings.ContainsFunc(got, istC1) {
			t.Errorf("Byte %#x ergibt %q", b, got)
		}
	}
	// Die acht mehrdeutigen und die vier unbelegten Bytes; mehr hieße, die Prüfung lehnt zu viel ab.
	if abgelehnt != 12 {
		t.Errorf("%d von 256 Bytes abgelehnt, erwartet 12", abgelehnt)
	}
}

func istC1(zeichen rune) bool {
	return zeichen >= 0x80 && zeichen <= 0x9f
}

func TestBaueLeser(t *testing.T) {
	leser, err := BaueLeser([]byte("M\xfcller;Das \"gro\xdfe\" Buch, Teil 1;7a"), HaeufigererTrenner)
	if err != nil {
		t.Fatalf("BaueLeser: %v", err)
	}
	zeile, err := leser.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(zeile) != 3 || zeile[0] != "Müller" || zeile[1] != "Das \"große\" Buch, Teil 1" || zeile[2] != "7a" {
		t.Errorf("Zeile = %q", zeile)
	}
}

func TestBaueLeserTrenntZeilenNurMitCR(t *testing.T) {
	leser, err := BaueLeser([]byte(ausExcelFuerMacNurCR), HaeufigererTrenner)
	if err != nil {
		t.Fatalf("BaueLeser: %v", err)
	}
	zeilen, err := leser.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(zeilen) != 3 || zeilen[1][0] != "Müller" || zeilen[2][0] != "Bäcker" {
		t.Errorf("Zeilen = %q", zeilen)
	}
}

func TestBaueLeserMeldetUnklareKodierung(t *testing.T) {
	if _, err := BaueLeser([]byte("Kre\x9aimir;7a"), HaeufigererTrenner); !errors.Is(err, ErrKodierung) {
		t.Errorf("Fehler = %v, erwartet ErrKodierung", err)
	}
}
