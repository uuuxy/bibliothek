package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LoescheAbgeschlossenenLauf löscht einen abgeschlossenen Lauf mit seinen Erfassungen und seinem
// Fehlbestandsbericht (beide fallen per ON DELETE CASCADE) und schreibt einen Eintrag ins
// Protokoll. Was der Lauf an den Büchern gebucht hat, bleibt: Ein als Verlust gebuchtes Exemplar
// bleibt ausgesondert und steht im Abgangsbuch, und findet es sich, holt die Theke es zurück.
// Einen offenen Lauf löscht die Funktion nicht, der wird verworfen; dann kommt wie bei einer
// unbekannten Kennung ErrInventurlaufNichtGefunden. Die Transaktion gehört dem Rufer.
func (a *InventurAbfragen) LoescheAbgeschlossenenLauf(ctx context.Context, laufID, bearbeiterID string) error {
	var bezeichnung, abgeschlossenAm string
	var erfasst, verluste int
	var verworfen bool
	err := a.db.QueryRow(ctx, `
		SELECT s.scope_label, s.abgeschlossen_am::text, coalesce(s.erfasst_gemeldet, 0), s.verworfen,
		       (SELECT count(*) FROM inventur_verluste v WHERE v.session_id = s.id)
		FROM inventur_sessions s
		WHERE s.id = $1 AND s.abgeschlossen_am IS NOT NULL
		FOR UPDATE
	`, laufID).Scan(&bezeichnung, &abgeschlossenAm, &erfasst, &verworfen, &verluste)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInventurlaufNichtGefunden
	}
	if err != nil {
		return fmt.Errorf("inventurlauf zum löschen lesen fehlgeschlagen: %w", err)
	}

	tag, err := a.db.Exec(ctx, `DELETE FROM inventur_sessions WHERE id = $1`, laufID)
	if err != nil {
		return fmt.Errorf("inventurlauf löschen fehlgeschlagen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrInventurlaufNichtGefunden
	}

	return schreibeProtokollEintrag(ctx, a.db, protokollEintrag{
		Tabelle: "inventur_sessions", Aktion: "DELETE", DatensatzID: laufID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Kontext: zeigerAuf("Frühere Inventur mit ihrem Fehlbestandsbericht gelöscht"),
		Details: map[string]any{
			"bezeichnung": bezeichnung, "abgeschlossen_am": abgeschlossenAm,
			"erfasst": erfasst, "verluste": verluste, "verworfen": verworfen,
		},
	})
}
