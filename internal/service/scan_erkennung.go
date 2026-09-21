package service

import (
	"context"
	"fmt"

	"bibliothek/repository"
)

// ScanTreffer ist die Antwort auf „was ist das?" für einen gescannten oder getippten
// Wert — OHNE Wirkung. Die Theke (ProcessQuery) bucht; die globale Suchleiste springt
// nur hin. Beide erkennen dasselbe: Exemplar-Barcode (auch als Littera-EAN-13), dann
// Schülerausweis. Typ "" = kein exakter Treffer, die Volltextsuche entscheidet.
type ScanTreffer struct {
	Typ     string `json:"typ"` // "exemplar" | "schueler"
	ID      string `json:"id"`
	TitelID string `json:"titel_id,omitempty"`
	Barcode string `json:"barcode"`
}

// ErkenneScan löst einen Wert exakt auf: erst Exemplar-Barcode, dann Littera-Etikett
// (EAN-13 → Mediennummer), dann Ausweis (Schüler oder Kollegium). Dieselbe Reihenfolge wie
// resolveOhnePraefix in der Theke, nur ohne Buchung.
func ErkenneScan(ctx context.Context, bookRepo repository.BookRepository, studentRepo repository.StudentRepository, q string) (*ScanTreffer, error) {
	for _, kandidat := range exemplarKandidaten(q) {
		copy, err := bookRepo.GetCopyByBarcode(ctx, kandidat)
		if err != nil {
			return nil, fmt.Errorf("exemplar auflösen: %w", err)
		}
		if copy != nil {
			return &ScanTreffer{Typ: "exemplar", ID: copy.ID, TitelID: copy.TitelID, Barcode: copy.BarcodeID}, nil
		}
	}
	// Ein LESER, nicht nur ein Schüler: Die Akte ist eine Maske für jeden, und die Theke
	// löst denselben Ausweis über dieselbe Funktion auf. Der Typ heißt weiter "schueler" —
	// er benennt das Sprungziel (die Akte), nicht die Art der Person.
	student, err := studentRepo.GetLeserByBarcode(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("ausweis auflösen: %w", err)
	}
	if student != nil {
		return &ScanTreffer{Typ: "schueler", ID: student.ID, Barcode: student.BarcodeID}, nil
	}
	return nil, nil
}

// exemplarKandidaten: der Wert selbst und — falls er ein Littera-Etikett ist — die
// rückgerechnete Mediennummer.
func exemplarKandidaten(q string) []string {
	kandidaten := []string{q}
	if nummer, istEtikett := dekodiereLitteraEtikett(q); istEtikett {
		kandidaten = append(kandidaten, nummer)
	}
	return kandidaten
}
