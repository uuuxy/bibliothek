package theke

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/repository"
)

// nichtLesbar ist ein Abruf der Einstellungen, der immer scheitert.
type nichtLesbar struct{}

func (nichtLesbar) Lade(context.Context) (*repository.SystemEinstellungen, error) {
	return nil, errors.New("einstellungen nicht lesbar")
}

// Sind die Einstellungen nicht lesbar, verlängert die Theke um die Vorgaben der Leihfristen, mit
// denen dann auch die Ausleihe rechnet (internal/service/frist.go, geraet.go), und nicht um eine
// eigene Zahl.
func TestVerlaengerungsRegel_VorgabeWieDieAusleihe(t *testing.T) {
	tage, _ := verlaengerungsRegel(context.Background(), nichtLesbar{})
	if tage.buch != repository.FristBuchTageVorgabe || tage.geraet != repository.FristGeraeteTageVorgabe {
		t.Errorf("verlaengerungsRegel ohne lesbare Einstellungen = %+v, erwartet Buch %d und Gerät %d Tage",
			tage, repository.FristBuchTageVorgabe, repository.FristGeraeteTageVorgabe)
	}
}
