// Package ordnung ordnet Texte wie ein deutsches Register: Umlaute stehen bei ihrem
// Grundbuchstaben, Zahlen nach ihrer Größe („Teil 2“ vor „Teil 10“), Groß- und Kleinschreibung
// trennt keine Buchstaben. Dieselbe Reihenfolge gibt die Kollation deutsch der Datenbank
// (de-u-kn) und der Browser mit Intl.Collator('de', {numeric: true}).
package ordnung

import (
	"bytes"
	"sort"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Sortiere ordnet die Liste nach dem Text, den text zu jedem Element liefert; gleiche Texte
// behalten ihre Reihenfolge.
//
// Der Collator entsteht je Aufruf: Er hält Zustand und darf nicht zwei Anfragen zugleich
// dienen.
func Sortiere[T any](liste []T, text func(T) string) {
	if len(liste) <= 1 {
		return
	}

	regel := collate.New(language.German)
	var puffer collate.Buffer
	schluessel := make([][]byte, len(liste))
	reihenfolge := make([]int, len(liste))
	for i := range liste {
		schluessel[i] = regel.KeyFromString(&puffer, zahlenNachGroesse(strings.TrimSpace(text(liste[i]))))
		reihenfolge[i] = i
	}
	sort.SliceStable(reihenfolge, func(a, b int) bool {
		return bytes.Compare(schluessel[reihenfolge[a]], schluessel[reihenfolge[b]]) < 0
	})

	sortiert := make([]T, len(liste))
	for ziel, quelle := range reihenfolge {
		sortiert[ziel] = liste[quelle]
	}
	copy(liste, sortiert)
}

// zahlenNachGroesse schreibt jede Ziffernfolge so um, dass die Buchstabenfolge sie nach ihrer
// Größe ordnet: Führende Nullen fallen weg, davor steht zweistellig die Zahl der Stellen
// („5“ wird „015“, „10“ wird „0210“).
//
// collate.Numeric leistet das nicht: Es ordnet die Zahl 0 hinter jede andere, sobald Text
// folgt („Heft 5 …“ vor „Heft 0 …“).
func zahlenNachGroesse(text string) string {
	var umgeschrieben strings.Builder
	umgeschrieben.Grow(len(text) + 8)
	for i := 0; i < len(text); {
		if !istZiffer(text[i]) {
			umgeschrieben.WriteByte(text[i])
			i++
			continue
		}
		ende := i
		for ende < len(text) && istZiffer(text[ende]) {
			ende++
		}
		ziffern := strings.TrimLeft(text[i:ende], "0")
		if ziffern == "" {
			ziffern = "0"
		}
		stellen := min(len(ziffern), 99)
		umgeschrieben.WriteByte(byte('0' + stellen/10))
		umgeschrieben.WriteByte(byte('0' + stellen%10))
		umgeschrieben.WriteString(ziffern)
		i = ende
	}
	return umgeschrieben.String()
}

func istZiffer(zeichen byte) bool { return zeichen >= '0' && zeichen <= '9' }
