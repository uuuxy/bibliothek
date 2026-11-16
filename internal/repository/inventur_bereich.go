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

// laufBeginn ist der Beginn des Laufs, dessen ID in $1 steht.
const laufBeginn = "(SELECT gestartet_am FROM inventur_sessions WHERE id = $1)"

// bedingung liefert die Bedingung „erwartet" des Laufs laufID (ohne WHERE) samt Argumenten: die
// ID des Laufs in $1, die Einschränkungen danach. Erwartet ist ein Exemplar im Bereich, das
// vorhanden und ausleihbar ist und seit dem Beginn des Laufs im Regal stehen musste. Ein Buch beim
// Schüler kann niemand scannen. Was seit dem Beginn verliehen oder zurückgegeben, angelegt,
// entsperrt oder wieder aufgenommen wurde, kam erst während der Inventur ins Regal oder ging
// hinaus; nicht gescannt bewiese es keinen Verlust.
func (s InventurBereich) bedingung(laufID string) (string, []any) {
	teile, args := s.dimensionen(2)
	bedingungen := append([]string{
		"e.ist_ausgesondert = false",
		"e.ist_ausleihbar = true",
		"NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND (a.rueckgabe_am IS NULL OR a.rueckgabe_am >= " + laufBeginn + "))",
		// Ausleihe, Rückgabe und jeder Wechsel von ist_ausleihbar oder ist_ausgesondert stempeln
		// letzte_bewegung_am (invarianten.md 1); ein neues Exemplar trägt keinen Stempel.
		"(e.letzte_bewegung_am IS NULL OR e.letzte_bewegung_am < " + laufBeginn + ")",
		"e.erstellt_am < " + laufBeginn,
	}, teile...)
	return strings.Join(bedingungen, "\n  AND "), append([]any{laufID}, args...)
}

// nichtErfasst ist die Bedingung „in diesem Lauf nicht gescannt" für das Exemplar e; die ID des
// Laufs steht in $1. Was erwartet ist und sie erfüllt, bucht der Abschluss als Verlust; die
// Zählung davor (ZaehleStand) nimmt sie von hier, damit beide dieselben Bücher meinen.
const nichtErfasst = `NOT EXISTS (
		SELECT 1 FROM inventur_erfassungen ie
		WHERE ie.session_id = $1 AND ie.exemplar_id = e.id
	)`

// dimensionBedingung liefert nur die Einschränkungen (Signatur, Fach, Jahrgang), für die
// Warnung beim Scan. Ohne Einschränkung ist alles im Bereich: "TRUE".
func (s InventurBereich) dimensionBedingung(start int) (string, []any) {
	teile, args := s.dimensionen(start)
	if len(teile) == 0 {
		return "TRUE", nil
	}
	return strings.Join(teile, " AND "), args
}
