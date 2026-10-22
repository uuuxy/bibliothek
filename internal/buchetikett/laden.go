package buchetikett

import (
	"context"
	"errors"
	"log"

	"bibliothek/repository"
)

// LadeDerBestellung lädt die Etiketten der Exemplare einer Bestellung, beschränkt auf die
// Positionen, die auch auf dem Barcodebogen der Bestellmail standen. Exemplare ohne
// Vorab-Barcode beklebt die Bibliothek selbst; bekäme der Lieferant auch ihre Etiketten,
// würden beide dasselbe Buch bekleben.
func LadeDerBestellung(ctx context.Context, datenbank repository.Datenbank, bestellungID string) ([]Etikett, error) {
	zeilen, err := repository.LadeEtikettDatenDerBestellung(ctx, datenbank, bestellungID)
	if err != nil {
		return nil, err
	}
	return etikettenAus(zeilen), nil
}

// ErgaenzeServerfelder füllt die Felder eines Druckauftrags nach, die nur der Server kennt:
// Anschaffungsjahr, Signatur und Topf. Das Druck-Center schickt nur Barcode, Titel und Autor;
// ein Feld, das die Oberfläche mitschicken muss, wird irgendwann vergessen.
//
// Wer hier ein Feld ergänzt, ergänzt es auch in LadeDerBestellung;
// TestEtikettenDatenSindAufBeidenWegenGleich hält die beiden Wege zusammen.
func ErgaenzeServerfelder(ctx context.Context, datenbank repository.Datenbank, etiketten []Etikett) {
	barcodes := make([]string, 0, len(etiketten))
	for i := range etiketten {
		if etiketten[i].BarcodeID != "" {
			barcodes = append(barcodes, etiketten[i].BarcodeID)
		}
	}
	if len(barcodes) == 0 {
		return
	}

	bekannt, err := repository.LadeEtikettServerfelderZuBarcodes(ctx, datenbank, barcodes)
	if err != nil {
		if errors.Is(err, repository.ErrZeileUnlesbar) {
			log.Printf("Etiketten: Serverfelder unvollständig gelesen: %v", err)
			return
		}
		log.Printf("Etiketten: Serverfelder nicht ermittelbar, drucke ohne: %v", err)
		return
	}

	for i := range etiketten {
		if felder, ok := bekannt[etiketten[i].BarcodeID]; ok {
			etiketten[i].AnschaffungsJahr = felder.Jahr
			etiketten[i].Signatur = felder.Signatur
			etiketten[i].Topf = felder.Topf
		}
	}
}
