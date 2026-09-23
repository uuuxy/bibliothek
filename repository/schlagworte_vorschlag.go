package repository

import (
	"context"
	"fmt"
	"strings"
)

// schlagwortStichwoerterMax kappt die Kandidaten eines Satzes. Die DNB liefert bis zu 98
// Verlagswörter je Titel (gemessen am 23.09.2026); die Kappung hält nur eine unerwartet
// lange Antwort von der Datenbank fern.
const schlagwortStichwoerterMax = 300

// SchlagworteAusStichwoertern gleicht die Stichwörter eines DNB-Satzes (Gattungsbegriffe und
// Verlagswörter, inventur.MetadatenErgebnis.Stichwoerter) gegen die eigene Liste ab —
// der Schlagwort-Vorschlag beim Bestellen per ISBN (docs/OFFEN.md 4.20). Vorgeschlagen wird
// nur, was es schon gibt: ein Schlagwort, das mindestens ein Titel trägt, oder ein Verweis
// darauf, aufgelöst zu seinem Ziel („Science Fiction" → „Science-Fiction"). Dieselbe Regel
// wie die Vorschlagsliste beim Tippen (SchlagwortVorschlaege: ein Wort ohne Titel ist meist
// ein Tippfehler, der nicht weiterleben soll) und dieselbe Auflösung wie der Schreibpfad
// (coalesce(verweis_auf, id)).
//
// Verglichen wird das ganze Wort ohne Rücksicht auf Groß- und Kleinschreibung, nach
// derselben Normalform wie beim Speichern (Leerraum zusammengezogen) — kein Teilstring:
// „Krieg" soll nicht „Kriegsende" vorschlagen. Die Antwort ist alphabetisch und ohne
// Doppelte; leer, wenn nichts passt. Geschrieben wird nichts: Das Ja gibt ein Mensch.
func SchlagworteAusStichwoertern(ctx context.Context, q DBQueryer, stichwoerter []string) ([]string, error) {
	kandidaten := make([]string, 0, len(stichwoerter))
	for _, roh := range stichwoerter {
		if wort := strings.Join(strings.Fields(roh), " "); wort != "" {
			kandidaten = append(kandidaten, wort)
		}
		if len(kandidaten) == schlagwortStichwoerterMax {
			break
		}
	}
	woerter := []string{}
	if len(kandidaten) == 0 {
		return woerter, nil
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT ziel.wort, lower(ziel.wort)
		FROM schlagworte s
		JOIN schlagworte ziel ON ziel.id = coalesce(s.verweis_auf, s.id)
		WHERE lower(s.wort) = ANY (SELECT lower(k) FROM unnest($1::text[]) AS k)
		  AND EXISTS (SELECT 1 FROM titel_schlagworte ts WHERE ts.schlagwort_id = ziel.id)
		ORDER BY lower(ziel.wort)`, kandidaten)
	if err != nil {
		return nil, fmt.Errorf("schlagwort-vorschlag aus stichwörtern: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wort, sortierung string
		if err := rows.Scan(&wort, &sortierung); err != nil {
			return nil, fmt.Errorf("schlagwort-vorschlag aus stichwörtern: %w", err)
		}
		woerter = append(woerter, wort)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schlagwort-vorschlag aus stichwörtern: %w", err)
	}
	return woerter, nil
}
