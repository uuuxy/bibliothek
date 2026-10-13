package repository

import (
	"fmt"
	"strings"
)

// InventurBereich beschreibt, welche Exemplare eine Inventur-Session erfasst: optional
// eingeschränkt nach Signatur, Fach (buecher_titel.subject) und/oder Klasse
// (jahrgang_von..jahrgang_bis enthält den Wert). Alle gesetzten Dimensionen sind
// UND-verknüpft; nil bedeutet "keine Einschränkung in dieser Dimension".
//
// Diese Struct + ihre Prädikat-Builder sind die EINZIGE Quelle für "was ist im Scope".
// Zählung, Scan-Warnung und Verlustbuchung leiten sich alle hieraus ab, damit die drei
// nie auseinanderlaufen (früher lag das Prädikat 3× dupliziert im Repo).
type InventurBereich struct {
	// Signatur ist ein PRÄFIX von buecher_titel.signatur, kein Fremdschlüssel
	// (Migration 060). Die frühere signature_id zeigte auf eine Tabelle, die
	// niemand pflegte — der Scope traf damit null Exemplare.
	Signatur *string
	Fach     *string
	Jahrgang *int
}

// dimensionen liefert die gesetzten Scope-Dimensionen als Einzelbedingungen (Aliasse
// e = buecher_exemplare, t = buecher_titel) plus ihre Argumente ab Platzhalter $start.
func (s InventurBereich) dimensionen(start int) ([]string, []any) {
	var bedingungen []string
	var args []any
	platz := start
	if s.Signatur != nil {
		bedingungen = append(bedingungen, SignaturPraefixBedingung("t.signatur", platz))
		args = append(args, *s.Signatur)
		platz++
	}
	if s.Fach != nil {
		bedingungen = append(bedingungen, fmt.Sprintf("t.subject = $%d", platz))
		args = append(args, *s.Fach)
		platz++
	}
	if s.Jahrgang != nil {
		// Klasse liegt im Jahrgangsbereich des Titels (z. B. Kl. 5 in "5–10").
		bedingungen = append(bedingungen, fmt.Sprintf("(t.jahrgang_von <= $%d AND t.jahrgang_bis >= $%d)", platz, platz))
		args = append(args, *s.Jahrgang)
	}
	return bedingungen, args
}

// bedingung liefert die volle "im Scope"-bedingung (ohne "WHERE"): physische Basis
// (vorhanden, ausleihbar, NICHT verliehen — ein beim Schüler befindliches Buch kann
// niemand scannen und darf beim Abschluss nicht als Verlust gelten) UND die Dimensionen.
// Argumente beginnen bei $start.
func (s InventurBereich) bedingung(start int) (string, []any) {
	teile, args := s.dimensionen(start)
	bedingungen := append([]string{
		"e.ist_ausgesondert = false",
		"e.ist_ausleihbar = true",
		"NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)",
	}, teile...)
	return strings.Join(bedingungen, "\n  AND "), args
}

// dimensionBedingung liefert NUR die Dimensionen (Signatur/Fach/Klasse) — für die
// nicht-blockierende Scan-Warnung "gehört nicht zum Scope". Ohne Dimensionen (globaler
// Scope) ist alles im Scope → "TRUE".
func (s InventurBereich) dimensionBedingung(start int) (string, []any) {
	teile, args := s.dimensionen(start)
	if len(teile) == 0 {
		return "TRUE", nil
	}
	return strings.Join(teile, " AND "), args
}
