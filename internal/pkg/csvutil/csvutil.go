// Package csvutil liest hochgeladene CSV-Dateien und entschärft Zellen für den Export nach CSV.
package csvutil

import (
	"encoding/csv"
	"strings"
)

// BaueLeser liefert den Leser für eine hochgeladene CSV-Datei; trenner wählt aus dem Text der
// Datei das Trennzeichen. Lose Anführungszeichen sind erlaubt: Titel und Namen aus Altdaten
// tragen sie. Lässt sich die Kodierung nicht sicher bestimmen, kommt ErrKodierung zurück.
func BaueLeser(inhalt []byte, trenner func(text string) rune) (*csv.Reader, error) {
	text, err := leseText(inhalt)
	if err != nil {
		return nil, err
	}
	leser := csv.NewReader(strings.NewReader(text))
	leser.Comma = trenner(text)
	leser.LazyQuotes = true
	return leser, nil
}

// HaeufigererTrenner wählt Semikolon oder Komma nach der ersten Zeile mit Inhalt, meist der
// Kopfzeile, gezählt außerhalb von Anführungszeichen. In den Daten stehen Kommas in Titeln und
// Namen („Schurf, Bernd"); gezählt über die ganze Datei, läsen sie eine Liste mit drei Spalten und
// Semikolon als Komma-Liste. Steht in der ersten Zeile keins von beiden, zählt die ganze Datei.
func HaeufigererTrenner(text string) rune {
	for _, zeile := range strings.Split(text, "\n") {
		if strings.TrimSpace(zeile) == "" {
			continue
		}
		semikolon, komma := zaehleTrennerAusserhalbVonAnfuehrung(zeile)
		if semikolon+komma > 0 {
			return mehrSemikolonAlsKomma(semikolon, komma)
		}
		break
	}
	return mehrSemikolonAlsKomma(strings.Count(text, ";"), strings.Count(text, ","))
}

// mehrSemikolonAlsKomma liefert ';' nur, wenn Semikolons überwiegen; sonst ','.
func mehrSemikolonAlsKomma(semikolon, komma int) rune {
	if semikolon > komma {
		return ';'
	}
	return ','
}

// zaehleTrennerAusserhalbVonAnfuehrung zählt Semikolons und Kommas einer Zeile, die nicht in
// Anführungszeichen stehen.
func zaehleTrennerAusserhalbVonAnfuehrung(zeile string) (semikolon, komma int) {
	inAnfuehrung := false
	for _, zeichen := range zeile {
		switch {
		case zeichen == '"':
			inAnfuehrung = !inAnfuehrung
		case inAnfuehrung:
		case zeichen == ';':
			semikolon++
		case zeichen == ',':
			komma++
		}
	}
	return semikolon, komma
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
