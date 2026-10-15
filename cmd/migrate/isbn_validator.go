package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

var barcodeMuster = regexp.MustCompile(`^B-(\d+)$`)

// hoechsteBarcodeNummer liest die höchste schon vergebene Nummer B-XXXXX aus PostgreSQL.
//
// Die Nummer wird in SQL als Zahl bestimmt, nicht über MAX(barcode_id): barcode_id ist VARCHAR,
// dort gilt die Ordnung der Zeichen. Ab 100.000 Exemplaren ist 'B-99999' größer als 'B-100000'
// (weil '9' > '1'), der Zähler spränge auf 99999 zurück und vergäbe danach Barcodes, die es
// längst gibt: 23505 auf buecher_exemplare.barcode_id, für jeden weiteren Datensatz. Für eine
// Schulbibliothek mit rund 80.000 Titeln ist diese Schwelle erreichbar.
//
// Der reguläre Ausdruck schützt zugleich die Umwandlung: Nur reine Ziffernfolgen bis neun
// Stellen werden überhaupt gelesen. Ein von Hand vergebener Barcode wie 'B-2024/A' bringt so
// weder den CAST zum Scheitern, noch setzt er den Zähler unbemerkt auf 0 zurück, was jede
// folgende Nummer kollidieren ließe.
func hoechsteBarcodeNummer(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var nummer sql.NullInt64
	err := pool.QueryRow(ctx, `
		SELECT MAX(CAST(substring(barcode_id FROM 3) AS BIGINT))
		FROM buecher_exemplare
		WHERE barcode_id ~ '^B-[0-9]{1,9}$'
	`).Scan(&nummer)
	if err != nil {
		return 0, fmt.Errorf("höchsten Barcode ermitteln: %w", err)
	}
	if !nummer.Valid {
		return 0, nil // noch keine generierten Barcodes vorhanden
	}
	return int(nummer.Int64), nil
}

// naechsteBarcodes liefert anzahl fortlaufende Barcodes nach nummer.
func naechsteBarcodes(nummer, anzahl int) []string {
	liste := make([]string, anzahl)
	for i := range liste {
		liste[i] = fmt.Sprintf("B-%05d", nummer+i+1)
	}
	return liste
}

// istBarcodeGueltig meldet, ob ein Barcode dem Muster B-XXXXX folgt.
func istBarcodeGueltig(barcode string) bool {
	return barcodeMuster.MatchString(barcode)
}
