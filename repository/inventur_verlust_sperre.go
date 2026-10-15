package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrVerlustNochGebunden meldet, dass mindestens ein gewähltes Exemplar noch an einem laufenden
// Vorgang hängt. Die Tür macht daraus 409 mit dem Grund im Text; bei 500 ersetzte der Sanitizer
// ihn durch „interner Datenbankfehler".
var ErrVerlustNochGebunden = errors.New("exemplar hängt noch an einem laufenden Vorgang")

// pruefeVerlusteLoeschbar weist den ganzen Stapel ab, sobald ein Exemplar noch an einem
// laufenden Vorgang hängt, und nennt die betroffenen Barcodes. Das endgültige Löschen nimmt die
// Ausleihhistorie des Exemplars mit, und eine unbezahlte Forderung der Schule verschwände
// dabei. Dieselbe Regel wie beim Leser (pruefeKeineOffenenVorgaenge): Solange etwas offen ist,
// wird nicht gelöscht, und der Grund wird genannt.
//
// Alles oder nichts: Ein Teilerfolg („28 von 30 gelöscht") ließe die Frage, welche zwei übrig
// sind und warum. Mit den Barcodes im Text lassen sie sich in der Liste abwählen.
func (a *InventurAbfragen) pruefeVerlusteLoeschbar(ctx context.Context, ids []string, barcodeVon map[string]string) error {
	// Die offene Ausleihe ist ein Netz für einen Fall, den es nicht geben sollte: Der Abschluss
	// bucht ein verliehenes Exemplar nicht als Verlust (InventurBereich.bedingung schließt es
	// aus).
	for _, sperre := range []struct {
		grund string
		query string
	}{
		{
			"noch verliehen",
			`SELECT DISTINCT exemplar_id FROM ausleihen
			 WHERE exemplar_id = ANY($1) AND rueckgabe_am IS NULL`,
		},
		{
			// ist_bezahlt = false genügt: Das Storno setzt ist_bezahlt = true
			// (StorniereForderung); eine stornierte Forderung ist keine offene.
			"unbezahlte Gebühr",
			`SELECT DISTINCT exemplar_id FROM schadensfaelle
			 WHERE exemplar_id = ANY($1) AND ist_bezahlt = false`,
		},
	} {
		betroffen, err := a.ladeBarcodesZu(ctx, sperre.query, ids, barcodeVon)
		if err != nil {
			return err
		}
		if len(betroffen) > 0 {
			return mitGrund(ErrVerlustNochGebunden, "%s (%s)",
				strings.Join(betroffen, ", "), sperre.grund)
		}
	}
	return nil
}

// ladeBarcodesZu liefert die Barcodes der Exemplare, die die Abfrage nennt, sortiert, damit die
// Meldung bei gleicher Lage gleich lautet.
func (a *InventurAbfragen) ladeBarcodesZu(ctx context.Context, query string, ids []string, barcodeVon map[string]string) ([]string, error) {
	rows, err := a.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("prüfung auf laufende vorgänge fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	var barcodes []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("prüfung auf laufende vorgänge fehlgeschlagen: %w", err)
		}
		barcodes = append(barcodes, barcodeVon[id])
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("prüfung auf laufende vorgänge fehlgeschlagen: %w", err)
	}
	sort.Strings(barcodes)
	return barcodes, nil
}
