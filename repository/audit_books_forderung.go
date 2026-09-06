package repository

// audit_books_forderung.go — die Spur, die eine gelöschte Forderung hinterlässt.
//
// Gegenstück zu inventur/db_books_delete_spur.go für den Einzel-Titel-Weg. Beide
// Löschwege räumen `schadensfaelle` ohne Rücksicht auf `ist_bezahlt` ab; das ist gewollt
// (sonst hielte der FK den Titel fest), aber es darf nicht spurlos geschehen: Ein
// unbezahlter Schadensfall ist Geld, das ein Schüler der Schule schuldet.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// protokolliereOffeneForderungen hält jede unbezahlte, nicht stornierte Forderung der
// Exemplare dieses Titels fest — in derselben Transaktion, direkt vor dem DELETE.
//
// `schueler_id` ist nicht nur Information, sondern der Schlüssel, an dem die
// Lesehistorie-Befristung die Zeile findet (repository/loeschfristen.go verlangt
// `details ? 'schueler_id'`); ohne ihn stünde der Klarname 24 Monate hier statt 90 Tage.
func protokolliereOffeneForderungen(ctx context.Context, tx pgx.Tx, titelID string) error {
	rows, err := tx.Query(ctx, `
		SELECT sf.id, e.id, e.barcode_id,
		       coalesce(nullif(trim(coalesce(s.vorname,'') || ' ' || coalesce(s.nachname,'')), ''),
		                nullif(trim(coalesce(b.vorname,'') || ' ' || coalesce(b.nachname,'')), ''),
		                '(unbekannt)'),
		       sf.schueler_id, to_char(sf.betrag, 'FM9999990.00'), sf.beschreibung,
		       to_char(sf.erstellt_am, 'YYYY-MM-DD')
		FROM schadensfaelle sf
		JOIN buecher_exemplare e ON sf.exemplar_id = e.id
		LEFT JOIN schueler s     ON sf.schueler_id = s.id
		LEFT JOIN benutzer b     ON sf.benutzer_id = b.id
		WHERE e.titel_id = $1 AND sf.ist_bezahlt = false AND sf.storniert_am IS NULL
		ORDER BY e.barcode_id`, titelID)
	if err != nil {
		return fmt.Errorf("offene forderungen konnten nicht gelesen werden: %w", err)
	}
	type forderung struct {
		id, exemplarID, barcode, schuldner, betrag, grund, seit string
		schuelerID                                              *string
	}
	var alle []forderung
	for rows.Next() {
		var f forderung
		if err := rows.Scan(&f.id, &f.exemplarID, &f.barcode, &f.schuldner, &f.schuelerID,
			&f.betrag, &f.grund, &f.seit); err != nil {
			rows.Close()
			return fmt.Errorf("offene forderungen konnten nicht gelesen werden: %w", err)
		}
		alle = append(alle, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("offene forderungen konnten nicht gelesen werden: %w", err)
	}

	for _, f := range alle {
		inhalt := map[string]any{
			"schadensfall_id": f.id,
			"barcode_id":      f.barcode,
			"schuldner":       f.schuldner,
			"betrag":          f.betrag,
			"beschreibung":    f.grund,
			"erstellt_am":     f.seit,
			"action":          "titel_geloescht_mit_offener_forderung",
		}
		if f.schuelerID != nil {
			inhalt["schueler_id"] = *f.schuelerID
		}
		details, err := json.Marshal(inhalt)
		if err != nil {
			return fmt.Errorf("protokoll der offenen forderung: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_log (tabelle, aktion, datensatz_id, akteur, kontext, details)
			VALUES ('schadensfaelle', 'DELETE', $1, 'SYSTEM', $2, $3::jsonb)`,
			f.exemplarID,
			"Titel gelöscht, es stand noch eine unbezahlte Forderung offen",
			string(details)); err != nil {
			return fmt.Errorf("protokoll der offenen forderung: %w", err)
		}
	}
	return nil
}
