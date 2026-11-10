package theke

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"
)

// nichtLesbar ist ein Abruf der Einstellungen, der immer scheitert.
type nichtLesbar struct{}

func (nichtLesbar) Lade(context.Context) (*repository.SystemEinstellungen, error) {
	return nil, errors.New("einstellungen nicht lesbar")
}

// Sind die Einstellungen nicht lesbar, verlängert die Theke um die Vorgabe der Leihfrist, mit der
// dann auch die Ausleihe rechnet (internal/service/frist.go), und nicht um eine eigene Zahl.
func TestVerlaengerungsRegel_VorgabeWieDieAusleihe(t *testing.T) {
	if tage, _ := verlaengerungsRegel(context.Background(), nichtLesbar{}); tage != repository.FristBuchTageVorgabe {
		t.Errorf("verlaengerungsRegel ohne lesbare Einstellungen = %d Tage, erwartet %d", tage, repository.FristBuchTageVorgabe)
	}
}
