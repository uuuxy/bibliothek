package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"bibliothek/db"
)

// SetzeExemplarBarcode ändert die Barcode-Zuordnung eines Exemplars.
func (a *pgKatalogAbfragen) SetzeExemplarBarcode(ctx context.Context, id string, barcode string) error {
	query := `
		UPDATE buecher_exemplare
		SET barcode_id = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	tag, err := a.db.Exec(ctx, query, barcode, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}
	return nil
}

// AendereExemplarStatus ändert den Verleihstatus und Zustand eines Exemplars.
//
// aussonderung_grund muss hier mitgeführt werden (chk_aussonderung_grund, Migration
// 043): ausgesondert verlangt genau einen Grund, im Umlauf verlangt NULL. Der einzige
// Aufrufer ist der Status-Editor, und dort setzt genau EIN Zustand aus — "Verloren".
// Deshalb ist VERLUST der fachliche Default (Betreiber-Entscheidung 19.08.2026): "Verloren"
// in der Oberfläche muss in der Verlustquote zählen (statistik.go: VERLUST/BESCHAEDIGUNG),
// nicht als AUSSORTIERT verschwinden. Ein bereits gesetzter, spezifischerer Grund — etwa
// BESCHAEDIGUNG aus der Schadensmeldung — bleibt via COALESCE beim blossen Bearbeiten der
// Notiz erhalten. Beim Reaktivieren wird der Grund gelöscht, sonst lehnt der CHECK ab.
// Guard seit 31.08.2026: „Ausgesondert" hatte DREI Türen, aber nur das Ausbuchen
// (ProtokollAbfragen.SondereExemplarAus) prüfte die offene Ausleihe. Über die beiden anderen ließ sich
// ein Buch aussondern, das ein Schüler gerade in der Tasche hatte — es verschwand aus
// Katalog, Kiosk und Inventur, während der Schüler in der Mahnstrecke blieb (Bugklasse
// „Zwei Türen zum selben Zustand"). Unbekannte IDs waren zudem stille „Erfolge"
// (RowsAffected prüfte niemand).
//
// Der Beschädigungsgrad (Migration 127) reist mit, weil er im selben Dialog steht — aber
// nur, wenn er mitgeschickt wird: COALESCE($5, alter Wert). Ohne das wäre jeder
// Statuswechsel eine stille 0, und der Ersatzbetrag stiege wieder auf den vollen Zeitwert.
//
// Wird ein Exemplar dabei ausgesondert, steht es mit der Person im Protokoll
// (ProtokolliereAussonderung), in derselben Transaktion wie der Wechsel. Wer ein schon
// ausgesondertes Exemplar noch einmal speichert, etwa mit geänderter Notiz, schreibt keinen
// zweiten Eintrag.
func (a *pgKatalogAbfragen) AendereExemplarStatus(ctx context.Context, id string, istAusleihbar bool, istAusgesondert bool, zustandNotiz string, zustandAbwertungProzent *int, bearbeiterID string) error {
	if istAusgesondert && bearbeiterID == "" {
		return ErrAussonderungOhneBearbeiter
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("status: transaktion öffnen: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Der Stand vor dem Wechsel, unter Sperre: Zwei Plätze, die dasselbe Exemplar zugleich
	// aussondern, schrieben sonst beide einen Eintrag.
	var warAusgesondert bool
	err = tx.QueryRow(ctx, `SELECT ist_ausgesondert FROM buecher_exemplare WHERE id = $1 FOR UPDATE`, id).
		Scan(&warAusgesondert)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExemplarNichtGefunden
	}
	if err != nil {
		return fmt.Errorf("status: alten stand lesen: %w", err)
	}

	query := `
		UPDATE buecher_exemplare
		SET ist_ausleihbar = $1,
		    ist_ausgesondert = $2,
		    aussonderung_grund = CASE
		        WHEN $2 THEN COALESCE(aussonderung_grund, 'VERLUST')
		        ELSE NULL
		    END,
		    zustand_notiz = $3, aktualisiert_am = CURRENT_TIMESTAMP,
		    zustand_abwertung_prozent = COALESCE($5::smallint, zustand_abwertung_prozent),
		    -- Freigeben oder Aussondern ist ein Ausgang aus dem Zulauf wie der Wareneingang:
		    -- bliebe bestellstatus stehen, zählten OPAC, Inventur und Katalog das Exemplar nie.
		    bestellstatus = CASE WHEN $1 OR $2 THEN NULL ELSE bestellstatus END
		WHERE id = $4
		  AND NOT ($2::boolean AND EXISTS (
		      SELECT 1 FROM ausleihen a WHERE a.exemplar_id = $4 AND a.rueckgabe_am IS NULL))
	`
	tag, err := tx.Exec(ctx, query, istAusleihbar, istAusgesondert, zustandNotiz, id, zustandAbwertungProzent)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if istAusgesondert {
			return deuteAussonderungsHindernis(ctx, tx, id)
		}
		return ErrExemplarNichtGefunden
	}
	if istAusgesondert && !warAusgesondert {
		if err := ProtokolliereAussonderung(ctx, tx, []string{id}, bearbeiterID, AussonderungsWegStatus, nil); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("status: commit: %w", err)
	}
	return nil
}

// IstExemplarImBestand sagt, ob ein Exemplar zum Bestand zählt: weder ausgesondert noch
// bestellt. Die Status-Tür nennt den Wert nach dem Speichern, damit die Oberfläche die Regel
// von AendereExemplarStatus nicht nachbaut.
func (a *pgKatalogAbfragen) IstExemplarImBestand(ctx context.Context, id string) (bool, error) {
	var imBestand bool
	err := a.db.QueryRow(ctx,
		`SELECT (`+SQLExemplarImBestand+`) FROM buecher_exemplare e WHERE e.id = $1`, id).Scan(&imBestand)
	return imBestand, err
}

// deuteAussonderungsHindernis unterscheidet, WARUM der Aussonderungs-Guard nichts traf:
// verliehen (400 mit Auskunft) oder unbekannt (404) — statt eines stillen „Erfolgs".
func deuteAussonderungsHindernis(ctx context.Context, q Datenbank, id string) error {
	var verliehen bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL)`,
		id).Scan(&verliehen); err != nil {
		return err
	}
	if verliehen {
		return ErrExemplarNochVerliehen
	}
	return ErrExemplarNichtGefunden
}
