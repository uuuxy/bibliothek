package repository

// Was die Bestandsbücher von einem gelöschten Exemplar behalten.
//
// Zugangs- und Abgangsbuch lesen buecher_exemplare. Ein Exemplar, das körperlich gelöscht wird,
// fiele aus beiden heraus, auch aus Halbjahren, deren Ausdruck schon abgeheftet ist. Deshalb
// löscht jeder Weg, der Exemplare entfernt, über loescheExemplareMitBuchzeile: Dieselbe
// Anweisung, die die Zeile löscht, legt ihre Zeile in geloeschte_exemplare an (Migration 171).

import (
	"context"
	"fmt"
)

// exemplareNach sagt, über welche Kennung loescheExemplareMitBuchzeile die Exemplare wählt.
type exemplareNach string

const (
	// exemplareNachKennung wählt Exemplare über ihre eigene Kennung.
	exemplareNachKennung exemplareNach = "e.id"
	// exemplareNachTitel wählt alle Exemplare der genannten Titel.
	exemplareNachTitel exemplareNach = "e.titel_id"
)

// loescheExemplareMitBuchzeile löscht die gewählten Exemplare und legt für jedes, das im Bestand
// war, die Zeile der Bestandsbücher an. Löschen und Anlegen sind eine Anweisung (DELETE …
// RETURNING in einer schreibenden CTE): Genau die Zeilen, die fallen, bekommen ihre Buchzeile,
// auch ein Exemplar, das zwischen Lesen und Löschen dazukam.
//
// Ein schon ausgesondertes Exemplar behält Tag und Grund seiner Aussonderung; es bleibt in dem
// Halbjahr, in dem es abging. Für jedes andere ist das Löschen der Abgang, mit dem Grund, den der
// Rufer übergibt. Ausleihen und Schadensfälle muss der Rufer vorher entfernen (ON DELETE
// RESTRICT). Die Antwort ist die Zahl der gelöschten Exemplare.
func loescheExemplareMitBuchzeile(ctx context.Context, q Datenbank, nach exemplareNach, ids []string, grund string) (int64, error) {
	var geloescht int64
	err := q.QueryRow(ctx, `
		WITH weg AS (
			DELETE FROM buecher_exemplare AS e WHERE `+string(nach)+` = ANY($1::uuid[])
			RETURNING e.*
		), eingetragen AS (
			INSERT INTO geloeschte_exemplare
				(exemplar_id, barcode, titel, signatur, zugang_am, lieferant, topf_belegt, topf, abgang_am, grund)
			SELECT e.id, e.barcode_id, t.titel, COALESCE(t.signatur, ''), e.zugang_am,
			       COALESCE(bv_topf.lieferant_name, ''), `+exemplarTopfBelegtSQL+`, `+ExemplarTopfSQL+`,
			       CASE WHEN e.ist_ausgesondert THEN e.ausgesondert_am ELSE now() END,
			       CASE WHEN e.ist_ausgesondert THEN e.aussonderung_grund ELSE $2 END
			FROM weg e
			JOIN buecher_titel t ON t.id = e.titel_id
			`+ExemplarTopfJoin+`
			WHERE `+sqlWarImBestand+`
			RETURNING 1
		)
		SELECT count(*) FROM weg
	`, ids, grund).Scan(&geloescht)
	if err != nil {
		return 0, fmt.Errorf("exemplare mit buchzeile löschen: %w", err)
	}
	return geloescht, nil
}
