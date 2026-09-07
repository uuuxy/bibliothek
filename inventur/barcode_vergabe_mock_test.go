package inventur

import (
	"fmt"

	"github.com/pashagolub/pgxmock/v4"
)

// erwarteBarcodeVergabe stellt die zwei Abfragen von repository.ZieheFreieExemplarBarcodes
// in Erwartung (nextval aus barcode_seq, dann der Bestandsabgleich ohne Treffer) und
// liefert die Nummern, die der Schreibpfad danach einfügt. Seit dem 07.09.2026 ziehen
// Bestandskorrektur und Sammelimport aus barcode_seq statt aus einer eigenen SYS-Sequenz.
func erwarteBarcodeVergabe(mock pgxmock.PgxPoolIface, anzahl int) []string {
	codes := make([]string, anzahl)
	rows := pgxmock.NewRows([]string{"code"})
	for i := range codes {
		codes[i] = fmt.Sprintf("B-%05d", 10001+i)
		rows.AddRow(codes[i])
	}
	mock.ExpectQuery(`nextval\('barcode_seq'\)`).WithArgs(anzahl).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT barcode_id FROM buecher_exemplare WHERE barcode_id = ANY`).
		WithArgs(codes).WillReturnRows(pgxmock.NewRows([]string{"barcode_id"}))
	return codes
}
