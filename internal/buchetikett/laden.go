package buchetikett

import (
	"context"
	"errors"
	"log"

	"bibliothek/repository"
)

// LadeDesTitels lädt alle Exemplare (Barcode, Titel, Autor, Anschaffungsjahr, Signatur) eines Titels.
func LadeDesTitels(ctx context.Context, datenbank repository.Datenbank, titelID string) ([]Etikett, error) {
	zeilen, err := repository.LadeEtikettDatenDesTitels(ctx, datenbank, titelID)
	if err != nil {
		return nil, err
	}
	return etikettenAus(zeilen), nil
}

// LadeDerBestellung holt die Exemplare EINER Bestellung, beschränkt auf die
// Positionen, die auch auf dem Barcodebogen der Bestellmail standen.
//
// Ohne die Einschränkung über mit_vorab_barcode bekäme der Lieferant Etiketten für
// Exemplare, die bewusst ohne Vorab-Barcode bestellt wurden — die beklebt dann die
// Bibliothek selbst, und beide würden dasselbe Buch bekleben.
func LadeDerBestellung(ctx context.Context, datenbank repository.Datenbank, bestellungID string) ([]Etikett, error) {
	zeilen, err := repository.LadeEtikettDatenDerBestellung(ctx, datenbank, bestellungID)
	if err != nil {
		return nil, err
	}
	return etikettenAus(zeilen), nil
}

// ErgaenzeServerfelder füllt die Etikettenfelder nach, die NICHT aus der Anfrage
// stammen: Anschaffungsjahr und Signatur.
//
// Der Nachdruck-Dialog schickt nur Barcode, Titel und Autor — beides stünde sonst nie
// auf einem nachgedruckten Etikett. Es dem Client mitzugeben wäre der falsche Weg: Es
// ist Serverwissen, und ein Feld, das die Oberfläche mitschicken MUSS, wird irgendwann
// vergessen (dieselbe Klasse Fehler hat hier schon Buchdaten still verschluckt).
//
// Genau das ist der Signatur passiert: Diese Nachfüllung gab es zuerst nur fürs Jahr,
// also trug dasselbe Buch je nach Druckweg einen anderen Aufkleber — über
// GET /api/buecher/titel/{id}/etiketten mit Signatur, über POST /api/print/labels ohne.
// Wer hier ein Feld ergänzt, ergänzt es auch in LadeDesTitels und LadeDerBestellung;
// TestEtikettenWegeDruckenDasselbe hält die Wege am fertigen PDF zusammen.
//
// Eine Abfrage für alle Barcodes, kein N+1. Unbekannte Barcodes bleiben unangetastet —
// beim Vorab-Druck für eine Bestellung existieren die Exemplare noch gar nicht, dort
// ist das der richtige Zustand und kein Fehler.
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
