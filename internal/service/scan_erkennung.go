package service

import (
	"context"
	"fmt"
)

// ScanTreffer ist die Antwort auf „was ist das?" für einen gescannten oder getippten Wert, ohne
// Wirkung. Die Theke (VerarbeiteEingabe) bucht; die Suche im Medienkatalog zeigt nur den Titel.
// Beide erkennen ein Exemplar gleich, auch über das Etikett von Littera. Typ "" heißt kein
// genauer Treffer, die Suche entscheidet.
type ScanTreffer struct {
	Typ     string `json:"typ"` // "exemplar" | "schueler"
	ID      string `json:"id"`
	TitelID string `json:"titel_id,omitempty"`
	Barcode string `json:"barcode"`
	// Ausgesondert: Der Katalog führt einen Titel nur mit einem Exemplar im Bestand. Ohne die
	// Angabe hieße ein ausgesondertes Exemplar dort „keins gefunden".
	Ausgesondert bool `json:"ausgesondert,omitempty"`
}

// ErkenneScan löst einen Wert exakt auf: erst Exemplar-Barcode, dann Littera-Etikett
// (EAN-13 → Mediennummer), dann Ausweis (Schüler oder Kollegium). Dieselbe Reihenfolge wie
// verarbeiteOhneVorsilbe in der Theke, nur ohne Buchung.
func ErkenneScan(ctx context.Context, katalogAbfragen exemplarErkennung, leserAbfragen ausweisErkennung, text string) (*ScanTreffer, error) {
	if treffer, err := ErkenneExemplar(ctx, katalogAbfragen, text); treffer != nil || err != nil {
		return treffer, err
	}
	// Ein Leser, nicht nur ein Schüler: Die Akte ist eine Maske für jeden, und die Theke
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

// ErkenneExemplar löst einen Wert zu einem Exemplar auf, über den Barcode selbst und über das
// Etikett von Littera; nil, wenn keines ihn trägt. Die Suche im Medienkatalog fragt nur danach:
// Sie zeigt Bücher, keine Leser.
func ErkenneExemplar(ctx context.Context, katalogAbfragen exemplarErkennung, text string) (*ScanTreffer, error) {
	for _, kandidat := range exemplarKandidaten(text) {
		exemplar, err := katalogAbfragen.FindeExemplar(ctx, kandidat)
		if err != nil {
			return nil, fmt.Errorf("exemplar auflösen: %w", err)
		}
		if exemplar != nil {
			return &ScanTreffer{
				Typ: "exemplar", ID: exemplar.ID, TitelID: exemplar.TitelID, Barcode: exemplar.BarcodeID,
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
