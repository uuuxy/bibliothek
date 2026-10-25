package repository

import (
	"context"
	"fmt"
	"time"
)

// Die Barcodes der Theke: die Nummern aller Exemplare,
// damit der Theken-Rechner ohne Netz eine nackte Ziffernfolge einordnen kann — Buch oder
// Ausweis. Nur Nummern, keine Personendaten.
//
// Ausgesonderte gehören dazu: Kommt ein verloren
// gemeldetes Buch offline zurück, holt das Nachbuchen es in den Umlauf. Fehlte seine Nummer,
// gälte es an der Theke als unklar und sperrte die Zuordnung. barcode_id ist über alle
// Exemplare eindeutig; eine Nummer bleibt eine Buchnummer, auch wenn das Buch abgeschrieben ist.
//
// Littera-Etiketten stehen nicht als EAN-13 in der Liste, nur als Nummer: Der Rechner rechnet
// das Etikett selbst zurück (frontend/src/lib/utils/litteraEtikett.js, gleiche Prüffälle wie der
// Server) und schlägt die Nummer nach.

// BuchbarcodeStand ist die Kennzahl des Bestands: Anzahl und jüngste Änderung. Beides
// zusammen — die Anzahl allein übersähe eine Umetikettierung, der Zeitstempel allein ein
// gelöschtes Exemplar, das nicht das zuletzt geänderte war.
type BuchbarcodeStand struct {
	Anzahl   int
	Juengste *time.Time
}

// LadeBuchbarcodeStand liest die Kennzahl über den Index, ohne die Liste zu lesen — ein
// unveränderter Bestand spart damit die ganze Auslieferung.
func LadeBuchbarcodeStand(ctx context.Context, q Datenbank) (BuchbarcodeStand, error) {
	var s BuchbarcodeStand
	err := q.QueryRow(ctx, `
		SELECT count(*), max(aktualisiert_am) FROM buecher_exemplare
	`).Scan(&s.Anzahl, &s.Juengste)
	if err != nil {
		return s, fmt.Errorf("buch-barcodes zählen: %w", err)
	}
	return s, nil
}

// LadeBuchbarcodes liest die Barcodes aller Exemplare, aufsteigend.
//
// Bewusst ohne LIMIT — anders als jede andere Listen-Abfrage (vgl. api/system/protokoll_grenze_pg_test.go).
// Eine gekappte Liste wäre schlimmer als keine: Die fehlenden Bücher gälten offline als
// „unklar", und niemand sähe, dass es an der Kappung liegt. Der Umfang ist durch den
// Bestand begrenzt und wächst nicht von selbst weiter; der Aufrufer misst die ausgelieferte
// Größe und warnt, wenn die Annahme kippt.
func LadeBuchbarcodes(ctx context.Context, q Datenbank, erwartet int) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT barcode_id FROM buecher_exemplare
		WHERE barcode_id IS NOT NULL AND barcode_id <> ''
		ORDER BY barcode_id
	`)
	if err != nil {
		return nil, fmt.Errorf("buch-barcodes lesen: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0, erwartet)
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			return nil, fmt.Errorf("buch-barcode lesen: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
