package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Die Fehler beim Aussondern eines Exemplars. Fehlerwerte und kein Vergleich am Text der
// Meldung: Der traf nie, und ein verliehenes Buch endete als Serverfehler statt als Auskunft.
var (
	ErrExemplarNochVerliehen = errors.New("exemplar ist aktuell noch verliehen")
	ErrExemplarNichtGefunden = errors.New("exemplar nicht gefunden oder bereits ausgebucht")
	// ErrExemplarInzwischenGeaendert heißt: Ein anderer Platz hat den Status geändert, seit der
	// Dialog geöffnet wurde.
	ErrExemplarInzwischenGeaendert = errors.New("status des exemplars inzwischen geändert")
)

// sqlIstVerliehen sagt, ob das Exemplar e gerade verliehen ist.
const sqlIstVerliehen = `EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)`

// sqlIstImHaus sind die Exemplare, die beim Löschen mit einem Grund aus dem Bestand gehen: im
// Bestand gewesen, nicht ausgesondert, nicht verliehen. Dieselbe Bedingung zählt die Vorschau des
// Löschdialogs (LadeLoeschfolgen).
const sqlIstImHaus = `e.ist_ausgesondert = false AND ` + sqlWarImBestand + ` AND NOT ` + sqlIstVerliehen

// sqlSetzeAusgesondert ist, was Aussondern an einem Exemplar ändert; $2 ist der Grund. Das Datum
// stempelt trg_exemplar_abgangsdatum.
const sqlSetzeAusgesondert = `ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = $2, ` +
	`bestellstatus = NULL, letzte_bewegung_am = ` + sqlStempelJetzt

// SondereExemplarAus sondert ein Exemplar mit dem genannten Grund aus („Exemplar löschen" in der
// Buchakte) und schreibt den Eintrag dazu. Die Zeile bleibt in der Tabelle und steht im
// Abgangsbuch; Ausleihen und Schadensfälle hängen an ihr. Ein verliehenes Exemplar lehnt es ab:
// Ein verlorenes Buch wird in der Leserakte als Verlust gemeldet. Die Zustandsnotiz bleibt, wie sie
// ist.
func (a *KatalogAbfragen) SondereExemplarAus(ctx context.Context, exemplarID, grund, bearbeiterID string) error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.Verwerfe(ctx, tx)

	// Was der Eintrag nennt, wird vor dem Ändern gelesen. Ein unbekanntes Exemplar ist ein
	// Fehler: Sonst entstünde ein Eintrag über ein Aussondern, das es nicht gab.
	var barcode, titel, titelID string
	err = tx.QueryRow(ctx,
		`SELECT e.barcode_id, e.titel_id, t.titel
		 FROM buecher_exemplare e
		 JOIN buecher_titel t ON e.titel_id = t.id
		 WHERE e.id = $1`,
		exemplarID,
	).Scan(&barcode, &titelID, &titel)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExemplarNichtGefunden
	}
	if err != nil {
		return fmt.Errorf("failed to snapshot copy for audit: %w", err)
	}

	var offeneAusleihen int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM ausleihen WHERE exemplar_id = $1 AND rueckgabe_am IS NULL", exemplarID).Scan(&offeneAusleihen)
	if err != nil {
		return fmt.Errorf("failed to check active loans for copy: %w", err)
	}
	if offeneAusleihen > 0 {
		return ErrExemplarNochVerliehen
	}

	// Die Bedingung ist_ausgesondert = false macht den zweiten Klick zu „nicht gefunden" statt zu
	// einem zweiten Erfolg, der Tag und Grund eines längst ausgesonderten Exemplars überschriebe.
	tag, err := tx.Exec(ctx, "UPDATE buecher_exemplare SET "+sqlSetzeAusgesondert+" WHERE id = $1 AND ist_ausgesondert = false", exemplarID, grund)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExemplarNichtGefunden
	}

	// Barcode und Titel stehen im Eintrag, damit die Theke das Buch auch dann findet, wenn es
	// später mit seinem Titel gelöscht wird.
	if err := protokolliereAussonderung(ctx, tx, []string{exemplarID}, bearbeiterID, AussonderungsWegExemplarLoeschen,
		map[string]any{"barcode_id": barcode, "titel_id": titelID, "titel": titel}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
