package repository

import (
	"context"
	"time"
)

// Die Stufe zwischen Mahnliste und Bescheid (15.09.2026).
//
// „Verlust melden" beendet die Ausleihe (ReportDamage). Damit fällt das Kind aus der
// Mahnliste, die nur offene Ausleihen kennt — und der Bescheid, der genau diese
// Forderung braucht, war nur erreichbar, solange die Liste noch nicht neu geladen war.
// Diese Abfrage macht die Stufe sichtbar: je Kind eine Zeile über seine offenen
// Forderungen, die noch auf keinem Brief stehen.

// ForderungOhneBescheid ist eine Zeile des Reiters „Schadensersatz": ein Kind mit
// offenen Forderungen ohne Brief.
type ForderungOhneBescheid struct {
	SchuelerID   string  `json:"schueler_id"`
	SchuelerName string  `json:"schueler_name"`
	Klasse       string  `json:"klasse"`
	Anzahl       int     `json:"anzahl"`
	Summe        float64 `json:"summe"`
	// Seit: die älteste dieser Forderungen — so lange wartet der Fall schon.
	Seit time.Time `json:"seit"`
	// Lernmittel: mindestens eine der Forderungen betrifft ein Lernmittel und kann
	// damit auf den Bescheid des Landes. Die Rechnung der Schülerbücherei ist noch
	// nicht gebaut (mittel_konzept.md 4.7, Etappe 3).
	Lernmittel bool `json:"lernmittel"`
}

// Ausstehend liefert je Kind die offenen Forderungen ohne Bescheid, älteste zuerst.
//
// Dieselbe Bedingung wie OffeneForderungen und ordnePositionenZu: offen, nicht
// storniert, noch auf keinem Brief. Gelöschte Schüler bleiben draußen (Papierkorb,
// DSGVO) — wie in der Mahnliste.
func (r *pgBescheidRepository) Ausstehend(ctx context.Context) ([]ForderungOhneBescheid, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.vorname || ' ' || s.nachname, coalesce(s.klasse, ''),
		       count(*)::int, coalesce(sum(f.betrag), 0)::float8, min(f.erstellt_am),
		       bool_or(coalesce(t.ist_lernmittel, false))
		FROM schadensfaelle f
		JOIN schueler s ON s.id = f.schueler_id
		LEFT JOIN buecher_exemplare e ON e.id = f.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		WHERE f.bescheid_id IS NULL
		  AND f.ist_bezahlt = false
		  AND f.storniert_am IS NULL
		  AND s.deleted_at IS NULL
		GROUP BY s.id, s.vorname, s.nachname, s.klasse
		ORDER BY min(f.erstellt_am), s.nachname, s.vorname
		LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ForderungOhneBescheid{}
	for rows.Next() {
		var z ForderungOhneBescheid
		if err := rows.Scan(&z.SchuelerID, &z.SchuelerName, &z.Klasse, &z.Anzahl, &z.Summe,
			&z.Seit, &z.Lernmittel); err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}
