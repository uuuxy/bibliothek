package inventur

import (
	"fmt"

	"github.com/pashagolub/pgxmock/v5"
)

// erwarteBarcodeVergabe erwartet die zwei Abfragen von repository.ZieheFreieExemplarBarcodes
// (nextval aus barcode_seq, dann der Abgleich mit dem Bestand ohne Treffer) und liefert die
// Nummern, die der Schreibweg danach einfügt. Korrektur des Bestands und Sammelimport ziehen aus
// barcode_seq.
func erwarteBarcodeVergabe(mock pgxmock.PgxPoolIface, anzahl int) []string {
	codes := make([]string, anzahl)
	zeilen := pgxmock.NewRows([]string{"nextval"})
	for i := range codes {
		codes[i] = fmt.Sprintf("B-%05d", 10001+i)
		zeilen.AddRow(int64(10001 + i))
	}
	mock.ExpectQuery(`nextval\('barcode_seq'\)`).WithArgs(anzahl).WillReturnRows(zeilen)
	mock.ExpectQuery(`SELECT barcode_id FROM buecher_exemplare WHERE barcode_id = ANY`).
		WithArgs(codes).WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}))
	return codes
}
