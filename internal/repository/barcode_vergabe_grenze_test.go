package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// Eine Vergabe über der Grenze zieht keine einzige Nummer: Eine ISBN im Feld „Bestand" brachte
// sonst den Server um seinen Speicher, und barcode_seq rückte um Millionen vor.
func TestZieheFreieExemplarBarcodes_ZiehtUeberDerGrenzeNichts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() = %v", err)
	}
	defer mock.Close()

	_, err = zieheFreieExemplarBarcodes(context.Background(), mock, HoechstensNummernJeVergabe+1)
	if !errors.Is(err, errZuVieleNummern) {
		t.Errorf("zieheFreieExemplarBarcodes(Grenze+1) = %v, erwartet errZuVieleNummern", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Abfragen an der Datenbank: %v", err)
	}
}
