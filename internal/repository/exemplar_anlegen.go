package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErzeugeBarcodes zieht anzahl fortlaufende Barcodes.
// Die Vergabe selbst steht in barcode_vergabe.go — sie ist die einzige Quelle für
// B-Nummern und liefert nur Nummern, die im Bestand noch nicht vorkommen.
func (a *KatalogAbfragen) ErzeugeBarcodes(ctx context.Context, anzahl int) ([]string, error) {
	barcodes, err := zieheFreieExemplarBarcodes(ctx, a.db, anzahl)
	if err != nil {
		return nil, err
	}
	if len(barcodes) != anzahl {
		return nil, errors.New("barcode sequence generation count mismatch")
	}
	return barcodes, nil
}

// LegeExemplareAn legt die Exemplare in der Transaktion des Rufers an, alle in einem Zug
// (CopyFrom). Den Standort
// erben sie von den Exemplaren ihres Titels im Bestand (sqlGeerbterStandort), gelesen vor dem
// Einfügen: Die neuen Zeilen zählen dabei nicht mit.
func (a *KatalogAbfragen) LegeExemplareAn(ctx context.Context, tx pgx.Tx, exemplare []ExemplarNeu) error {
	if len(exemplare) == 0 {
		return nil
	}

	titelIDs := make([]string, 0, len(exemplare))
	for _, c := range exemplare {
		titelIDs = append(titelIDs, c.TitelID)
	}
	standorte, err := ladeGeerbteStandorte(ctx, tx, titelIDs)
	if err != nil {
		return err
	}

	var zeilen [][]any
	for _, c := range exemplare {
		zeilen = append(zeilen, []any{
			c.TitelID, c.BarcodeID, c.ZustandNotiz, c.IstAusleihbar, c.EtikettGedruckt, c.Einkaufspreis,
			leerAlsNull(c.BestellungID), leerAlsNull(c.Bestellstatus),
			leerAlsNull(standorte[c.TitelID]),
		})
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"buecher_exemplare"},
		[]string{"titel_id", "barcode_id", "zustand_notiz", "ist_ausleihbar", "etikett_gedruckt", "einkaufspreis", "bestellung_id", "bestellstatus", "standort"},
		pgx.CopyFromRows(zeilen),
	)
	return err
}
