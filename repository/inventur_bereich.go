package repository

import (
	"fmt"
	"strings"
)

// InventurBereich beschreibt, welche Exemplare ein Lauf erfasst: wahlweise eingeschränkt nach
// Signatur, Fach (buecher_titel.subject) und Jahrgang (jahrgang_von bis jahrgang_bis enthält
// den Wert). Gesetzte Einschränkungen gelten zusammen; nil heißt keine Einschränkung.
//
// Der Typ und seine Bedingungen sind die eine Quelle für „im Bereich": Zählung, Scan-Warnung
// und Verlustbuchung leiten sich daraus ab, damit die drei nicht auseinanderlaufen.
type InventurBereich struct {
	// Signatur ist ein Präfix von buecher_titel.signatur, kein Fremdschlüssel.
	Signatur *string
	Fach     *string
	Jahrgang *int
}

// dimensionen liefert die gesetzten Einschränkungen als einzelne Bedingungen (Aliasse e =
// buecher_exemplare, t = buecher_titel) und ihre Argumente ab Platzhalter $start.
func (s InventurBereich) dimensionen(start int) ([]string, []any) {
	var bedingungen []string
	var args []any
	platz := start
	if s.Signatur != nil {
		bedingungen = append(bedingungen, signaturPraefixBedingung("t.signatur", platz))
		args = append(args, *s.Signatur)
		platz++
	}
	if s.Fach != nil {
		bedingungen = append(bedingungen, fmt.Sprintf("t.subject = $%d", platz))
		args = append(args, *s.Fach)
		platz++
	}
	if s.Jahrgang != nil {
		// Der Jahrgang liegt im Bereich des Titels (etwa 5 in „5–10").
		bedingungen = append(bedingungen, fmt.Sprintf("(t.jahrgang_von <= $%d AND t.jahrgang_bis >= $%d)", platz, platz))
		args = append(args, *s.Jahrgang)
	}
	return bedingungen, args
}

// bedingung liefert die ganze Bedingung „im Bereich" (ohne WHERE): vorhanden, ausleihbar und
// nicht verliehen, dazu die Einschränkungen. Ein Buch beim Schüler kann niemand scannen, und es
// darf beim Abschluss nicht als Verlust gelten. Die Argumente beginnen bei $start.
func (s InventurBereich) bedingung(start int) (string, []any) {
	teile, args := s.dimensionen(start)
	bedingungen := append([]string{
		"e.ist_ausgesondert = false",
		"e.ist_ausleihbar = true",
		"NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)",
	}, teile...)
	return strings.Join(bedingungen, "\n  AND "), args
}

// nichtErfasst liefert die Bedingung „in diesem Lauf nicht gescannt" für das Exemplar e, mit der
// ID des Laufs im Platzhalter $platz. Was im Bereich liegt und sie erfüllt, bucht der Abschluss
// als Verlust; die Zählung davor (ZaehleStand) nimmt sie von hier, damit beide dieselben Bücher
// meinen.
func nichtErfasst(platz int) string {
	return fmt.Sprintf(`NOT EXISTS (
		SELECT 1 FROM inventur_erfassungen ie
		WHERE ie.session_id = $%d AND ie.exemplar_id = e.id
	)`, platz)
}

// dimensionBedingung liefert nur die Einschränkungen (Signatur, Fach, Jahrgang), für die
// Warnung beim Scan. Ohne Einschränkung ist alles im Bereich: "TRUE".
func (s InventurBereich) dimensionBedingung(start int) (string, []any) {
	teile, args := s.dimensionen(start)
	if len(teile) == 0 {
		return "TRUE", nil
	}
	return strings.Join(teile, " AND "), args
}
