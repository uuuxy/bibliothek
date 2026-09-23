package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Suche über Schlagworte (docs/OFFEN.md 4.20, freigegeben am 23.09.2026): Katalog und
// Portal finden einen Titel auch über seine Schlagworte und über die Verweise darauf. Wer
// „Tierfantasy" sucht, findet die Titel mit „Fantasy", sobald „Tierfantasy" ein Verweis auf
// „Fantasy" ist.
//
// Die Regel steht hier für beide Formen der Suche:
//   - am Server (Titel-Verwaltung, öffentlicher Katalog und Portal) als Bedingung,
//     SQLTitelUeberSchlagwort;
//   - im Browser (Medienkatalog „Suche & Filter" lädt die ganze Liste und filtert selbst)
//     als Wörter je Titel, SuchwoerterDerTitel.
//
// Beide lösen ein Wort so auf wie der Schreibpfad SetzeSchlagworte: coalesce(verweis_auf, id).
// Ein Schlagwort steht für sich, ein Verweis für sein Ziel.

// sqlWortZumTitel verbindet jedes Wort — Schlagwort oder Verweis — mit den Titeln, die sein
// Ziel tragen. Ein Titel hängt nie an einem Verweis (Migration 143); jede Zeile von
// titel_schlagworte trifft deshalb das Wort selbst und jeden Verweis darauf genau einmal.
const sqlWortZumTitel = `schlagworte sw
		JOIN titel_schlagworte tsw ON tsw.schlagwort_id = coalesce(sw.verweis_auf, sw.id)`

// SQLTitelUeberSchlagwort ist die Bedingung „ein Schlagwort des Titels oder ein Verweis
// darauf enthält den Suchtext". Verglichen wird wie bei Titel und Autor: Teilstring, ohne
// Rücksicht auf Groß- und Kleinschreibung.
//
// muster ist der SQL-Ausdruck des Suchtexts (ein Parameter) — derselbe, den der Aufrufer
// für Titel und Autor einsetzt. Maskiert der Aufrufer dort die LIKE-Joker (der öffentliche
// Katalog), maskiert er sie damit auch hier.
//
// Die Unterabfrage hängt nicht vom Titel ab; Postgres rechnet sie einmal je Suche und
// prüft jeden Titel gegen die gehashte Menge.
func SQLTitelUeberSchlagwort(titelAlias, muster string) string {
	return titelAlias + `.id IN (
		SELECT tsw.titel_id FROM ` + sqlWortZumTitel + `
		WHERE sw.wort ILIKE '%' || ` + muster + ` || '%')`
}

// SQLTitelMitSchlagwort ist die Bedingung „der Titel trägt das Wort mit dieser Kennung" —
// der Filter im Portal. kennung ist der SQL-Ausdruck eines Parameters; die Unterabfrage
// hängt dann nicht vom Titel ab, und Postgres rechnet sie einmal je Suche. Sie geht über
// dieselbe Zuordnung wie die Suche und die Liste der Filter (sqlWortZumTitel). Ein
// Filterwort ist nie ein Verweis (chk_schlagwort_verweis_kein_filter), die Auflösung
// ändert für es also nichts; Liste und gefilterte Suche lesen aber dieselbe Zuordnung und
// können nicht auseinanderlaufen.
func SQLTitelMitSchlagwort(titelAlias, kennung string) string {
	return titelAlias + `.id IN (
		SELECT tsw.titel_id FROM ` + sqlWortZumTitel + `
		WHERE sw.id = ` + kennung + `)`
}

// SchlagwortFilter ist ein Wort, das im Portal als Filter steht.
type SchlagwortFilter struct {
	ID   string `json:"id"`
	Wort string `json:"wort"`
}

// OeffentlicheSchlagwortFilter liefert die Wörter, die die Pflegeseite als Filter markiert
// hat (ist_filter), alphabetisch — aber nur die, zu denen der öffentliche Katalog
// mindestens einen Titel zeigt (OeffentlichSichtbar). Ein Filter, der nichts findet, stünde
// sonst als Sackgasse im Portal: ein Wort etwa, das nur Lernmittel tragen.
//
// Die Abfrage geht vom Wort zu seinen Titeln, nicht umgekehrt: Eine Bedingung je Wort über
// alle Titel (EXISTS … SQLTitelMitSchlagwort(„f.id")) lief an 13.000 Titeln und 20
// Filterwörtern 353 ms, diese Form 4 ms (gemessen am 23.09.2026).
func OeffentlicheSchlagwortFilter(ctx context.Context, q DBQueryer) ([]SchlagwortFilter, error) {
	rows, err := q.Query(ctx, `
		SELECT f.id::text, f.wort
		FROM schlagworte f
		WHERE f.id IN (
		      SELECT sw.id FROM `+sqlWortZumTitel+`
		      JOIN buecher_titel bt ON bt.id = tsw.titel_id
		      WHERE sw.ist_filter AND `+OeffentlichSichtbar("bt")+`)
		ORDER BY lower(f.wort)`)
	if err != nil {
		return nil, fmt.Errorf("schlagwort-filter lesen: %w", err)
	}
	filter, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SchlagwortFilter])
	if err != nil {
		return nil, fmt.Errorf("schlagwort-filter lesen: %w", err)
	}
	return filter, nil
}

// SuchwoerterDerTitel liefert je Titel die Wörter, über die er gefunden wird: seine
// Schlagworte und die Verweise darauf, alphabetisch. Ein Titel ohne Schlagworte fehlt in
// der Antwort.
//
// Das ist ein Suchindex, keine Anzeige: Ein Verweis ist kein Schlagwort des Titels. Was am
// Titel steht, liefert SchlagworteDesTitels.
func SuchwoerterDerTitel(ctx context.Context, q DBQueryer, titelIDs []string) (map[string][]string, error) {
	woerter := make(map[string][]string)
	if len(titelIDs) == 0 {
		return woerter, nil
	}
	rows, err := q.Query(ctx, `
		SELECT tsw.titel_id::text, array_agg(sw.wort ORDER BY lower(sw.wort))
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
