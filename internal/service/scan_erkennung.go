package service

import (
	"context"
	"fmt"

	"bibliothek/repository"
)

// ScanTreffer ist die Antwort auf „was ist das?" für einen gescannten oder getippten
// Wert — OHNE Wirkung. Die Theke (VerarbeiteEingabe) bucht; die globale Suchleiste springt
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
// verarbeiteOhneVorsilbe in der Theke, nur ohne Buchung.
func ErkenneScan(ctx context.Context, katalogAbfragen repository.KatalogAbfragen, leserAbfragen repository.LeserAbfragen, text string) (*ScanTreffer, error) {
	for _, kandidat := range exemplarKandidaten(text) {
		exemplar, err := katalogAbfragen.FindeExemplar(ctx, kandidat)
		if err != nil {
			return nil, fmt.Errorf("exemplar auflösen: %w", err)
		}
		if exemplar != nil {
			return &ScanTreffer{Typ: "exemplar", ID: exemplar.ID, TitelID: exemplar.TitelID, Barcode: exemplar.BarcodeID}, nil
		}
	}
	// Ein LESER, nicht nur ein Schüler: Die Akte ist eine Maske für jeden, und die Theke
	// löst denselben Ausweis über dieselbe Funktion auf. Der Typ heißt weiter "schueler" —
	// er benennt das Sprungziel (die Akte), nicht die Art der Person.
	leser, err := leserAbfragen.FindeZuBarcode(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("ausweis auflösen: %w", err)
	}
	if leser != nil {
		return &ScanTreffer{Typ: "schueler", ID: leser.ID, Barcode: leser.BarcodeID}, nil
	}
	return nil, nil
}

// exemplarKandidaten: der Wert selbst und — falls er ein Littera-Etikett ist — die
// rückgerechnete Mediennummer.
func exemplarKandidaten(text string) []string {
	kandidaten := []string{text}
	if nummer, istEtikett := dekodiereLitteraEtikett(text); istEtikett {
		kandidaten = append(kandidaten, nummer)
	}
	return kandidaten
}
