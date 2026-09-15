package repository

import (
	"context"
	"fmt"
	"time"
)

// Die Barcodes der Theke (Stufe 2 des Offline-Baus, Commit 12): alle Nummern nicht
// ausgesonderter Exemplare, damit der Theken-Rechner ohne Netz eine nackte Ziffernfolge
// einordnen kann — Buch oder Ausweis. Nur Nummern, keine Personendaten.

// BuchbarcodeStand ist die Kennzahl des Bestands: Anzahl und jüngste Änderung. Beides
// zusammen — die Anzahl allein übersähe eine Umetikettierung, der Zeitstempel allein ein
// gelöschtes Exemplar, das nicht das zuletzt geänderte war.
type BuchbarcodeStand struct {
	Anzahl   int
	Juengste *time.Time
}

// LiesBuchbarcodeStand liest die Kennzahl über den Index, ohne die Liste zu lesen — ein
// unveränderter Bestand spart damit die ganze Auslieferung.
func LiesBuchbarcodeStand(ctx context.Context, q DBQueryer) (BuchbarcodeStand, error) {
	var s BuchbarcodeStand
	err := q.QueryRow(ctx, `
		SELECT count(*), max(aktualisiert_am) FROM buecher_exemplare WHERE ist_ausgesondert = false
	`).Scan(&s.Anzahl, &s.Juengste)
	if err != nil {
		return s, fmt.Errorf("buch-barcodes zählen: %w", err)
	}
	return s, nil
}

// ListeBuchbarcodes liest alle Barcodes nicht ausgesonderter Exemplare, aufsteigend.
//
// Bewusst OHNE LIMIT — anders als jede andere Listen-Abfrage (vgl. api/audit_limit_pg_test.go).
// Eine gekappte Liste wäre schlimmer als keine: Die fehlenden Bücher gälten offline als
// „unklar", und niemand sähe, dass es an der Kappung liegt. Der Umfang ist durch den
// Bestand begrenzt und wächst nicht von selbst weiter; der Aufrufer misst die ausgelieferte
// Größe und warnt, wenn die Annahme kippt.
func ListeBuchbarcodes(ctx context.Context, q DBQueryer, erwartet int) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT barcode_id FROM buecher_exemplare
		WHERE ist_ausgesondert = false AND barcode_id IS NOT NULL AND barcode_id <> ''
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
