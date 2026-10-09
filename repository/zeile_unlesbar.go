package repository

import (
	"errors"
	"fmt"
)

// ErrZeileUnlesbar meldet, dass die Abfrage lief, eine Zeile der Antwort sich aber nicht in
// die Felder lesen ließ. Die Tür unterscheidet daran ihre Meldung von der einer gescheiterten
// Abfrage.
var ErrZeileUnlesbar = errors.New("zeile unlesbar")

// zeileUnlesbar hängt den Fehler des Lesens an ErrZeileUnlesbar; die Ursache bleibt in der
// Kette, ein Abbruch der Anfrage bleibt als solcher erkennbar.
func zeileUnlesbar(err error) error {
	return fmt.Errorf("%w: %w", ErrZeileUnlesbar, err)
}
