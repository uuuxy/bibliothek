// Package csvutil liest hochgeladene CSV-Dateien und entschärft Zellen für den Export nach CSV.
package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// HoechstensZeilen ist die Zahl der Datenzeilen, die ein Import ohne eigene Grenze höchstens
// liest: weit über dem Bestand der Schule (Littera 2010: 61.580 Exemplare) und doch fest. Eine
// Datei von 100 MB aus kurzen Zeilen stünde sonst als 50 Millionen Zeilen im Speicher.
const HoechstensZeilen = 1_000_000

// ZuVieleZeilen heißt: Die Datei hat mehr Datenzeilen als Hoechstens.
type ZuVieleZeilen struct {
	Hoechstens int
}

func (e *ZuVieleZeilen) Error() string {
	return fmt.Sprintf("csv: mehr als %d datenzeilen", e.Hoechstens)
}

// Meldung ist der Satz für die Person, die die Datei hochlädt.
func (e *ZuVieleZeilen) Meldung() string {
	return fmt.Sprintf("Die Datei hat mehr als %d Zeilen, höchstens %d sind möglich", e.Hoechstens, e.Hoechstens)
}

// LeseZeilen liest die Kopfzeile und höchstens hoechstens Datenzeilen, sonst *ZuVieleZeilen.
// Gezählt wird beim Lesen, damit eine Datei aus kurzen Zeilen nicht erst ganz im Speicher steht.
func LeseZeilen(leser *csv.Reader, hoechstens int) ([][]string, error) {
	var zeilen [][]string
	for {
		zeile, err := leser.Read()
		if err == io.EOF {
			return zeilen, nil
		}
		if err != nil {
			return nil, err
		}
		if len(zeilen) > hoechstens {
			return nil, &ZuVieleZeilen{Hoechstens: hoechstens}
		}
		zeilen = append(zeilen, zeile)
	}
}

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
