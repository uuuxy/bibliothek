package repository

import (
	"errors"
	"fmt"
)

// ErrZeileUnlesbar meldet, dass die Abfrage lief, das Lesen ihrer Zeilen aber scheiterte: Eine
// Zeile passt nicht zu den Feldern, oder das Lesen brach ab. Die Tür unterscheidet daran ihre
// Meldung von der einer gescheiterten Abfrage.
var ErrZeileUnlesbar = errors.New("zeile unlesbar")

// zeileUnlesbar hängt den Fehler des Lesens an ErrZeileUnlesbar; die Ursache bleibt in der
// Kette, ein Abbruch der Anfrage bleibt als solcher erkennbar.
func zeileUnlesbar(err error) error {
	return fmt.Errorf("%w: %w", ErrZeileUnlesbar, err)
}
