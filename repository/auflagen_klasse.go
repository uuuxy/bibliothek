package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Gemischte Auflagen in einer Klasse (docs/OFFEN.md 4.18, Stufe 5). Entschieden am 17.09.2026:
// „Die Ausgabe warnt, wenn eine Klasse gemischte Auflagen bekommt. Still darf das nicht
// passieren: verschiedene Auflagen heißen verschiedene Seitenzahlen." Am 25.09.2026: als
// Hinweiszeile an der Theke wie bei der Fremdrückgabe, kein Dialog.
//
// Die Frage ist nur lesend und hängt an einer Ausleihe, die schon gebucht ist: Welche ANDEREN
// Auflagen desselben Buchs haben Kinder aus derselben Klasse gerade? „Dieselbe Klasse" wie in
// der Klassensatz-Übersicht (GetClassGroups): klassen_normkey über die Sicht schueler, nur
// aktive Kinder.

// AuflageInKlasse ist eine andere Auflage desselben Buchs und wie viele Kinder der Klasse
// sie gerade haben.
type AuflageInKlasse struct {
	Auflage          string `json:"auflage"`
	Erscheinungsjahr int    `json:"erscheinungsjahr"`
	Kinder           int    `json:"kinder"`
}

// AuflagenMischung ist der Hinweis an der Theke: Diese Auflage ging an ein Kind der Klasse,
// in der andere Auflagen desselben Buchs schon ausgegeben sind. Keine Namen — die Klasse und
// Zahlen reichen, um das Buch zurückzulegen und eine andere Auflage zu holen.
type AuflagenMischung struct {
	Klasse           string            `json:"klasse"`
	Auflage          string            `json:"auflage"`
	Erscheinungsjahr int               `json:"erscheinungsjahr"`
	Andere           []AuflageInKlasse `json:"andere"`
}

// AuflagenMischungInKlasse liefert den Hinweis für die Ausleihe eines Exemplars von titelID
// an schuelerID — oder nil, wenn es keinen gibt: Der Titel gehört zu keinem Werk, der Leser
// ist kein Schüler oder hat keine Klasse, oder in der Klasse hat niemand eine andere Auflage.
// Die Ausleihe selbst zählt nicht mit (sie ist diese Auflage).
func AuflagenMischungInKlasse(ctx context.Context, q DBQueryer, titelID, schuelerID string) (*AuflagenMischung, error) {
	var m AuflagenMischung
	var werkID *string
	err := q.QueryRow(ctx, `
		SELECT coalesce(s.klasse, ''), coalesce(t.auflage, ''), coalesce(t.erscheinungsjahr, 0), t.werk_id::text
		FROM buecher_titel t
		JOIN schueler s ON s.id = $2
		WHERE t.id = $1`, titelID, schuelerID).
		Scan(&m.Klasse, &m.Auflage, &m.Erscheinungsjahr, &werkID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (werkID == nil || m.Klasse == "")) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auflagen in der klasse: %w", err)
	}

	rows, err := q.Query(ctx, `
		SELECT coalesce(t.auflage, ''), coalesce(t.erscheinungsjahr, 0), count(DISTINCT s.id)::int
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN buecher_titel t ON t.id = e.titel_id
		JOIN schueler s ON s.id = a.schueler_id
		WHERE a.rueckgabe_am IS NULL
		  AND t.werk_id = $1::uuid AND t.id <> $2::uuid
		  AND s.deleted_at IS NULL AND s.ist_abgaenger = false
		  AND klassen_normkey(s.klasse) = klassen_normkey($3)
		GROUP BY t.id, t.auflage, t.erscheinungsjahr
		ORDER BY count(DISTINCT s.id) DESC, `+SQLNeuesteAuflageZuerst("t"), *werkID, titelID, m.Klasse)
	if err != nil {
		return nil, fmt.Errorf("auflagen in der klasse: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a AuflageInKlasse
		if err := rows.Scan(&a.Auflage, &a.Erscheinungsjahr, &a.Kinder); err != nil {
			return nil, fmt.Errorf("auflagen in der klasse: %w", err)
		}
		m.Andere = append(m.Andere, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auflagen in der klasse: %w", err)
	}
	if len(m.Andere) == 0 {
		return nil, nil
	}
	return &m, nil
}
