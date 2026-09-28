package littera

import (
	"fmt"
	"sort"
)

// Gruppenbefund nennt eine Littera-Lesergruppe, deren Leser keiner Art zugeordnet sind — so,
// dass man sie in Littera wiederfindet, und ohne einen Namen.
type Gruppenbefund struct {
	Nummer      string // Leser.Lesergruppe, der Schlüssel in Leser_UG
	Bezeichnung string // Leser_UG.Untergruppe; leer, wenn die Gruppe dort fehlt
	Klasse      string // Leser_UG.KurzBez
	Personen    int
	Ausleihen   int
}

// OhneZuordnung zählt je Lesergruppe die Leser ohne Art (ArtUnbekannt) und ihre Ausleihen.
//
// Solange die Liste nicht leer ist, schreibt der Personenlauf nichts: cmd/littera-altbestand
// prüft sie vor dem Bestand und im Trockenlauf, SchreibePersonen noch einmal vor der ersten
// Person. Bis zum 28.09.2026 übersprang der Lauf solche Leser einzeln; ihre Bücher stünden
// danach als verfügbar im Regal (Sicherung von 2010: 42 Konten, 341 Ausleihen).
func OhneZuordnung(ab *Altbestand) []Gruppenbefund {
	jeGruppe := map[string]*Gruppenbefund{}
	gruppeDes := map[string]string{} // Leser.ID → Lesergruppe
	for _, l := range ab.Leser {
		if l.Art != ArtUnbekannt {
			continue
		}
		b := jeGruppe[l.GruppeNr]
		if b == nil {
			b = &Gruppenbefund{Nummer: l.GruppeNr, Bezeichnung: l.Gruppe, Klasse: l.Klasse}
			jeGruppe[l.GruppeNr] = b
		}
		b.Personen++
		gruppeDes[l.ID] = l.GruppeNr
	}
	for _, a := range ab.Ausleihen {
		if nr, ok := gruppeDes[a.LeserID]; ok {
			jeGruppe[nr].Ausleihen++
		}
	}

	befund := make([]Gruppenbefund, 0, len(jeGruppe))
	for _, b := range jeGruppe {
		befund = append(befund, *b)
	}
	// Die Schlüssel sind Zahlen als Text: erst nach Länge, dann nach Wert, damit 2 vor 10 steht.
	sort.Slice(befund, func(i, j int) bool {
		a, b := befund[i].Nummer, befund[j].Nummer
		if len(a) != len(b) {
			return len(a) < len(b)
		}
		return a < b
	})
	return befund
}

// String nennt die Gruppe, wie Littera sie führt, mit Personen- und Ausleihzahl.
func (b Gruppenbefund) String() string {
	gruppe := "Lesergruppe " + b.Nummer + " (fehlt in Leser_UG)"
	switch {
	case b.Nummer == "":
		gruppe = "ohne Lesergruppe"
	case b.Bezeichnung != "":
		gruppe = fmt.Sprintf("Lesergruppe %s „%s“ (%s)", b.Nummer, b.Bezeichnung, b.Klasse)
	}
	return fmt.Sprintf("%s: %s, %s", gruppe,
		stueck(b.Personen, "Person", "Personen"), stueck(b.Ausleihen, "Ausleihe", "Ausleihen"))
}

func stueck(n int, eins, mehr string) string {
	if n == 1 {
		return "1 " + eins
	}
	return fmt.Sprintf("%d %s", n, mehr)
}
