package repository

import (
	"context"
	"fmt"
)

// Suche über Schlagworte: Katalog und
// Portal finden einen Titel auch über seine Schlagworte und über die Verweise darauf. Wer
// „Tierfantasy" sucht, findet die Titel mit „Fantasy", sobald „Tierfantasy" ein Verweis auf
// „Fantasy" ist.
//
// Die Regel steht hier für beide Formen der Suche:
//   - am Server als Bedingung, sqlTitelUeberSchlagwort, in der Titelsuche des Programms
//     (sqlTitelTrifft);
//   - im Browser (Medienkatalog „Suche & Filter" lädt die ganze Liste und filtert selbst)
//     als Wörter je Titel, ladeSuchwoerterDerTitel.
//
// Beide lösen ein Wort so auf wie der Schreibpfad SetzeSchlagworte: coalesce(verweis_auf, id).
// Ein Schlagwort steht für sich, ein Verweis für sein Ziel.

// sqlWortZumTitel verbindet jedes Wort — Schlagwort oder Verweis — mit den Titeln, die sein
// Ziel tragen. Ein Titel hängt nie an einem Verweis; jede Zeile von
// titel_schlagworte trifft deshalb das Wort selbst und jeden Verweis darauf genau einmal.
const sqlWortZumTitel = `schlagworte sw
		JOIN titel_schlagworte tsw ON tsw.schlagwort_id = coalesce(sw.verweis_auf, sw.id)`

// sqlTitelUeberSchlagwort ist die Bedingung „ein Schlagwort des Titels oder ein Verweis
// darauf enthält das Muster". Verglichen wird wie bei Titel und Autor: Teilstring nach
// suchnorm. muster ist der SQL-Ausdruck eines Suchworts nach suchnorm, als wörtlicher
// LIKE-Teilstring (sqlLikeMuster) — derselbe, den sqlTitelTrifft für Titel und Autor einsetzt.
//
// Die Unterabfrage hängt nicht vom Titel ab; Postgres rechnet sie einmal je Suche und
// prüft jeden Titel gegen die gehashte Menge.
func sqlTitelUeberSchlagwort(titelAlias, muster string) string {
	return titelAlias + `.id IN (SELECT tsw.titel_id FROM ` + sqlWortZumTitel +
		` WHERE suchnorm(sw.wort) LIKE '%' || ` + muster + ` || '%')`
}

// sqlTitelMitSchlagwort ist die Bedingung „der Titel trägt das Wort mit dieser Kennung" —
// der Filter im Portal. kennung ist der SQL-Ausdruck eines Parameters; die Unterabfrage
// hängt dann nicht vom Titel ab, und Postgres rechnet sie einmal je Suche. Sie geht über
// dieselbe Zuordnung wie die Suche und die Liste der Filter (sqlWortZumTitel). Ein
// Filterwort ist nie ein Verweis (chk_schlagwort_verweis_kein_filter), die Auflösung
// ändert für es also nichts; Liste und gefilterte Suche lesen aber dieselbe Zuordnung und
// können nicht auseinanderlaufen.
func sqlTitelMitSchlagwort(titelAlias, kennung string) string {
	return titelAlias + `.id IN (
		SELECT tsw.titel_id FROM ` + sqlWortZumTitel + `
		WHERE sw.id = ` + kennung + `)`
}

// SchlagwortFilter ist ein Wort, das im Portal als Filter steht.
type SchlagwortFilter struct {
	ID   string `json:"id"`
	Wort string `json:"wort"`
}

// ladeSuchwoerterDerTitel liefert je Titel die Wörter, über die er gefunden wird: seine
// Schlagworte und die Verweise darauf, alphabetisch. Ein Titel ohne Schlagworte fehlt in
// der Antwort.
//
// Das ist ein Suchindex, keine Anzeige: Ein Verweis ist kein Schlagwort des Titels. Was am
// Titel steht, liefert LadeSchlagworteDesTitels.
func ladeSuchwoerterDerTitel(ctx context.Context, q Datenbank, titelIDs []string) (map[string][]string, error) {
	woerter := make(map[string][]string)
	if len(titelIDs) == 0 {
		return woerter, nil
	}
	rows, err := q.Query(ctx, `
		SELECT tsw.titel_id::text, array_agg(sw.wort ORDER BY lower(sw.wort) COLLATE deutsch)
		FROM `+sqlWortZumTitel+`
		WHERE tsw.titel_id = ANY($1::uuid[])
		GROUP BY tsw.titel_id`, titelIDs)
	if err != nil {
		return nil, fmt.Errorf("suchwörter der titel lesen: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var titelID string
		var liste []string
		if err := rows.Scan(&titelID, &liste); err != nil {
			return nil, fmt.Errorf("suchwörter der titel lesen: %w", err)
		}
		woerter[titelID] = liste
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("suchwörter der titel lesen: %w", err)
	}
	return woerter, nil
}
