package repository

import (
	"fmt"
	"strings"
)

// InventurScope beschreibt, welche Exemplare eine Inventur-Session erfasst: optional
// eingeschränkt nach Signatur, Fach (buecher_titel.subject) und/oder Klasse
// (jahrgang_von..jahrgang_bis enthält den Wert). Alle gesetzten Dimensionen sind
// UND-verknüpft; nil bedeutet "keine Einschränkung in dieser Dimension".
//
// Diese Struct + ihre Prädikat-Builder sind die EINZIGE Quelle für "was ist im Scope".
// Zählung, Scan-Warnung und Verlustbuchung leiten sich alle hieraus ab, damit die drei
// nie auseinanderlaufen (früher lag das Prädikat 3× dupliziert im Repo).
type InventurScope struct {
	SignatureID *int
	Subject     *string
	Grade       *int
}

// dimensionen liefert die gesetzten Scope-Dimensionen als Einzelbedingungen (Aliasse
// e = buecher_exemplare, t = buecher_titel) plus ihre Argumente ab Platzhalter $start.
func (s InventurScope) dimensionen(start int) ([]string, []any) {
	var conds []string
	var args []any
	idx := start
	if s.SignatureID != nil {
		conds = append(conds, fmt.Sprintf("t.signature_id = $%d", idx))
		args = append(args, *s.SignatureID)
		idx++
	}
	if s.Subject != nil {
		conds = append(conds, fmt.Sprintf("t.subject = $%d", idx))
		args = append(args, *s.Subject)
		idx++
	}
	if s.Grade != nil {
		// Klasse liegt im Jahrgangsbereich des Titels (z. B. Kl. 5 in "5–10").
		conds = append(conds, fmt.Sprintf("(t.jahrgang_von <= $%d AND t.jahrgang_bis >= $%d)", idx, idx))
		args = append(args, *s.Grade)
	}
	return conds, args
}

// Bedingung liefert die volle "im Scope"-Bedingung (ohne "WHERE"): physische Basis
// (vorhanden, ausleihbar, NICHT verliehen — ein beim Schüler befindliches Buch kann
// niemand scannen und darf beim Abschluss nicht als Verlust gelten) UND die Dimensionen.
// Argumente beginnen bei $start.
func (s InventurScope) Bedingung(start int) (string, []any) {
	dims, args := s.dimensionen(start)
	conds := append([]string{
		"e.ist_ausgesondert = false",
		"e.ist_ausleihbar = true",
		"NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)",
	}, dims...)
	return strings.Join(conds, "\n  AND "), args
}

// DimensionBedingung liefert NUR die Dimensionen (Signatur/Fach/Klasse) — für die
// nicht-blockierende Scan-Warnung "gehört nicht zum Scope". Ohne Dimensionen (globaler
// Scope) ist alles im Scope → "TRUE".
func (s InventurScope) DimensionBedingung(start int) (string, []any) {
	dims, args := s.dimensionen(start)
	if len(dims) == 0 {
		return "TRUE", nil
	}
	return strings.Join(dims, " AND "), args
}
