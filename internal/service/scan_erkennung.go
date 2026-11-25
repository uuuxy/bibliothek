package service

import (
	"context"
	"fmt"

	"bibliothek/internal/pkg/code39"
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

// ErkenneExemplar löst einen Wert zu einem Exemplar auf, über ExemplarKandidaten und je Wert auch
// ein altes LMF-Etikett (FindeExemplar); nil, wenn keines ihn trägt.
func ErkenneExemplar(ctx context.Context, katalogAbfragen exemplarErkennung, text string) (*ExemplarTreffer, error) {
	for _, kandidat := range ExemplarKandidaten(text) {
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

// ExemplarKandidaten sind die Werte, unter denen ein gescanntes Buch im Bestand stehen kann, in
// der Reihenfolge der Theke: der Scan selbst, beim Etikett von Littera die zurückgerechnete
// Mediennummer, zuletzt der Scan ohne ein Prüfzeichen von Code 39 (code39.OhnePruefzeichen, nur
// als späterer Versuch). Wer zu einem Scan ein Buch sucht, nimmt die Liste von hier, damit jeder
// alte Aufdruck überall gilt.
func ExemplarKandidaten(text string) []string {
	kandidaten := []string{text}
	if nummer, istEtikett := dekodiereLitteraEtikett(text); istEtikett {
		kandidaten = append(kandidaten, nummer)
	}
	if kern, hatPruefzeichen := code39.OhnePruefzeichen(text); hatPruefzeichen {
		kandidaten = append(kandidaten, kern)
	}
	return kandidaten
}
