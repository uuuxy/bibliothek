// Package csvutil liest hochgeladene CSV-Dateien und entschärft Zellen für den Export nach CSV.
package csvutil

import (
	"encoding/csv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// BaueLeser liefert den Leser für eine hochgeladene CSV-Datei; trenner wählt aus dem Text der
// Datei das Trennzeichen. Lose Anführungszeichen sind erlaubt: Titel und Namen aus Altdaten
// tragen sie.
func BaueLeser(inhalt []byte, trenner func(text string) rune) *csv.Reader {
	text := leseText(inhalt)
	leser := csv.NewReader(strings.NewReader(text))
	leser.Comma = trenner(text)
	leser.LazyQuotes = true
	return leser
}

// HaeufigererTrenner wählt Semikolon oder Komma danach, was in der ganzen Datei häufiger steht.
func HaeufigererTrenner(text string) rune {
	if strings.Count(text, ";") > strings.Count(text, ",") {
		return ';'
	}
	return ','
}

// leseText macht aus den Bytes einer Datei Text. Excel speichert „CSV" unter Windows in
// Windows-1252 und „CSV UTF-8" mit BOM, und in beiden Formen steht derselbe Name. Was kein gültiges
// UTF-8 ist, gilt deshalb als Windows-1252. Nullzeichen fallen weg: Die Datenbank lehnte sie und
// ungewandelte Bytes ab, und die Übernahme endete mit einem Serverfehler.
func leseText(inhalt []byte) string {
	if utf8.Valid(inhalt) {
		return strings.ReplaceAll(strings.TrimPrefix(string(inhalt), "\uFEFF"), "\x00", "")
	}
	var text strings.Builder
	text.Grow(len(inhalt))
	for _, zeichen := range inhalt {
		if zeichen != 0 {
			text.WriteRune(charmap.Windows1252.DecodeByte(zeichen))
		}
	}
	return text.String()
}

// entschaerfeZelle stellt einer Zelle ein Apostroph voran, wenn sie mit =, +, -, @ oder einem
// Steuerzeichen beginnt. Eine Tabellenkalkulation rechnet eine solche Zelle sonst als Formel
// aus, und ein Titel aus einem importierten Katalog könnte beim Öffnen der Datei Befehle
// ausführen oder Daten verschicken.
func entschaerfeZelle(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return "'" + s
	}
	return s
}

// EntschaerfeZeile liefert eine neue Zeile, in der jede Zelle entschärft ist.
func EntschaerfeZeile(zeile []string) []string {
	entschaerft := make([]string, len(zeile))
	for i, zelle := range zeile {
		entschaerft[i] = entschaerfeZelle(zelle)
	}
	return entschaerft
}
