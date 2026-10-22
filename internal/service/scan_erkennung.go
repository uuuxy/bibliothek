package service

import (
	"context"
	"fmt"
)

// ExemplarTreffer ist das Exemplar zu einem gescannten oder getippten Wert, ohne Wirkung. Die
// Suche im Medienkatalog zeigt damit den Titel; gebucht wird nur an der Theke.
type ExemplarTreffer struct {
	ID      string `json:"id"`
	TitelID string `json:"titel_id"`
	Barcode string `json:"barcode"`
	// Ausgesondert: Der Katalog führt einen Titel nur mit einem Exemplar im Bestand. Ohne die
	// Angabe hieße ein ausgesondertes Exemplar dort „keins gefunden".
	Ausgesondert bool `json:"ausgesondert,omitempty"`
}

// ErkenneExemplar löst einen Wert zu einem Exemplar auf, über den Barcode selbst und über das
// Etikett von Littera (EAN-13 → Mediennummer), in der Reihenfolge der Theke
// (verarbeiteOhneVorsilbe); nil, wenn keines ihn trägt.
func ErkenneExemplar(ctx context.Context, katalogAbfragen exemplarErkennung, text string) (*ExemplarTreffer, error) {
	for _, kandidat := range exemplarKandidaten(text) {
		exemplar, err := katalogAbfragen.FindeExemplar(ctx, kandidat)
		if err != nil {
			return nil, fmt.Errorf("exemplar auflösen: %w", err)
		}
		if exemplar != nil {
			return &ExemplarTreffer{
				ID: exemplar.ID, TitelID: exemplar.TitelID, Barcode: exemplar.BarcodeID,
				Ausgesondert: exemplar.IstAusgesondert,
			}, nil
		}
	}
	return nil, nil
}

// exemplarKandidaten sind der Wert selbst und, wenn er ein Etikett von Littera ist, die
// zurückgerechnete Mediennummer.
func exemplarKandidaten(text string) []string {
	kandidaten := []string{text}
	if nummer, istEtikett := dekodiereLitteraEtikett(text); istEtikett {
		kandidaten = append(kandidaten, nummer)
	}
	return kandidaten
}
