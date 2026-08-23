package repository

// inventur_verlust_sperre.go — was dem endgültigen Löschen eines Verlust-Exemplars
// entgegensteht.
//
// Das endgültige Löschen entfernt die Ausleihhistorie des Exemplars mit. Das ist beim
// abgeschriebenen Buch richtig — die Zeilen zeigen auf ein Exemplar, das es nicht mehr
// gibt, und der Fehlbestandsbericht behält seine Abschrift. Für einen NOCH LAUFENDEN
// Vorgang wäre es falsch: Eine unbezahlte Gebühr ist eine Forderung der Schule, und die
// verschwände hier still und unwiederbringlich.
//
// Deshalb dieselbe Regel wie überall sonst, wo dieses System Daten endgültig entfernt
// (siehe blockiereBeiOffenenVorgaengen für Schüler): Solange etwas offen ist, wird
// nicht gelöscht — und der Grund wird benannt, nicht verschluckt.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrVerlustNochGebunden meldet, dass mindestens ein ausgewähltes Exemplar noch an einem
// laufenden Vorgang hängt. Der Handler macht daraus eine 409 mit Klartext — als 500
// ersetzte der Sanitizer die Begründung durch „interner Datenbankfehler".
var ErrVerlustNochGebunden = errors.New("exemplar hängt noch an einem laufenden Vorgang")

// pruefeVerlusteLoeschbar weist den GANZEN Stapel ab, sobald ein Exemplar gebunden ist,
// und nennt die betroffenen Barcodes.
//
// Bewusst alles-oder-nichts: Ein Teilerfolg („28 von 30 gelöscht") lässt die Bedienung
// mit der Frage zurück, welche zwei übrig sind und warum. Mit den Barcodes im Text kann
// man sie in der Liste abwählen und den Rest durchlaufen lassen.
func (r *InventoryRepository) pruefeVerlusteLoeschbar(ctx context.Context, ids []string, barcodeVon map[string]string) error {
	// Die offene Ausleihe ist ein Sicherheitsnetz, kein erwarteter Fall: Der
	// Inventurabschluss bucht ein verliehenes Exemplar gar nicht erst als Verlust
	// (InventurScope.Bedingung schließt es aus). Sie kostet eine Abfrage und deckt den
	// Weg ab, den heute niemand kennt.
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
			// ist_bezahlt = false genügt: Der Storno setzt ist_bezahlt = true
			// (StornierungGebuehr), eine stornierte Gebühr ist also keine offene.
			"unbezahlte Gebühr",
			`SELECT DISTINCT exemplar_id FROM schadensfaelle
			 WHERE exemplar_id = ANY($1) AND ist_bezahlt = false`,
		},
	} {
		betroffen, err := r.barcodesZu(ctx, sperre.query, ids, barcodeVon)
		if err != nil {
			return err
		}
		if len(betroffen) > 0 {
			return fmt.Errorf("%w: %s (%s)", ErrVerlustNochGebunden,
				strings.Join(betroffen, ", "), sperre.grund)
		}
	}
	return nil
}

// barcodesZu liefert die Barcodes der Exemplare, die die Abfrage zurückgibt — sortiert,
// damit die Meldung bei gleicher Lage gleich lautet.
func (r *InventoryRepository) barcodesZu(ctx context.Context, query string, ids []string, barcodeVon map[string]string) ([]string, error) {
	rows, err := r.db.Query(ctx, query, ids)
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
