package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// Die Tür erkennt an ErrZeileUnlesbar, dass das Lesen scheiterte. Die Ursache bleibt in der
// Kette: Ein Abbruch der Anfrage mitten im Lesen ist kein Serverfehler.
func TestZeileUnlesbar_HaeltKennzeichenUndUrsache(t *testing.T) {
	err := zeileUnlesbar(fmt.Errorf("lesen: %w", context.Canceled))
	if !errors.Is(err, ErrZeileUnlesbar) {
		t.Errorf("ErrZeileUnlesbar fehlt in der Kette: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("die Ursache fehlt in der Kette: %v", err)
	}
}
