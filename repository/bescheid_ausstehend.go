package repository

import (
	"context"
	"time"
)

// ForderungOhneBescheid ist eine Zeile des Reiters „Schadensersatz": ein Kind mit offenen
// Forderungen ohne Brief.
type ForderungOhneBescheid struct {
	SchuelerID   string  `json:"schueler_id"`
	SchuelerName string  `json:"schueler_name"`
	Klasse       string  `json:"klasse"`
	Anzahl       int     `json:"anzahl"`
	Summe        float64 `json:"summe"`
	// Seit ist das Datum der ältesten dieser Forderungen: So lange wartet der Fall schon.
	Seit time.Time `json:"seit"`
	// Land heißt: Mindestens eine der Forderungen betrifft ein Buch des Landes (Eigentum nach
	// ExemplarTopfSQL) und kann auf den Bescheid des Landes. Für die Schülerbücherei gibt es
	// noch keinen Brief (mittel_konzept.md 4.7).
	Land bool `json:"land"`
}

// LadeForderungenOhneBescheid liefert je Kind die offenen Forderungen ohne Bescheid, die
// ältesten zuerst. „Verlust melden" beendet die Ausleihe, und das Kind fällt aus der Mahnliste,
// die nur offene Ausleihen kennt; ohne diese Liste wäre der Fall bis zum Bescheid nirgends zu
// sehen.
//
// Dieselbe Bedingung wie LadeOffeneForderungen und ordnePositionenZu: offen, nicht storniert,
// auf keinem Brief. Leser im Papierkorb bleiben draußen, wie in der Mahnliste.
func (a *BescheidAbfragen) LadeForderungenOhneBescheid(ctx context.Context) ([]ForderungOhneBescheid, error) {
	rows, err := a.db.Query(ctx, `
		SELECT s.id, s.vorname || ' ' || s.nachname, coalesce(s.klasse, ''),
		       count(*)::int, coalesce(sum(f.betrag), 0)::float8, min(f.erstellt_am),
		       bool_or(e.id IS NOT NULL AND `+ExemplarTopfSQL+` = 'land')
		FROM schadensfaelle f
		JOIN schueler s ON s.id = f.schueler_id
		LEFT JOIN buecher_exemplare e ON e.id = f.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		`+ExemplarTopfJoin+`
		WHERE f.bescheid_id IS NULL
		  AND f.ist_bezahlt = false
		  AND f.storniert_am IS NULL
		  AND s.deleted_at IS NULL
		GROUP BY s.id, s.vorname, s.nachname, s.klasse
		ORDER BY min(f.erstellt_am), s.nachname COLLATE deutsch, s.vorname COLLATE deutsch
		LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ForderungOhneBescheid{}
	for rows.Next() {
		var z ForderungOhneBescheid
		if err := rows.Scan(&z.SchuelerID, &z.SchuelerName, &z.Klasse, &z.Anzahl, &z.Summe,
			&z.Seit, &z.Land); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}
