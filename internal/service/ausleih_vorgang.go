package service

import (
	"context"
	"time"

	"bibliothek/pkg/leserart"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// ausleihVorgang ist der Leser einer Buchung mit seiner Frist und den übergangenen Hinweisen.
type ausleihVorgang struct {
	leserID string
	// leser ist der Ausleiher. Über die Theke immer gesetzt; er trägt seine Art.
	leser *repository.Leser
	frist time.Time
	// uebergangen sind die Hinweise, die override_block übergangen hat (pruefeAusleihSperren).
	// verarbeiteAusleihe protokolliert sie, wenn die Ausleihe steht.
	uebergangen []string
}

// istSchueler sagt, ob die Regeln für Schüler greifen: Ausleihlimit, Vormerkungen und die Frist
// aus der Klasse. Sie hängen an der Art des Lesers.
func (v *ausleihVorgang) istSchueler() bool {
	return v.leser != nil && leserart.IstSchueler(v.leser.Art)
}

// ladeVorgang lädt den geöffneten Leser und bestimmt seine Frist: für einen Schüler aus der
// Klasse, für alle anderen ein Jahr. Gespeichert wird das Jahr, aber die Ausleihe eines
// Kollegen ist eine Dauerleihe, die nie überfällig wird.
//
// Die Sperren prüft BucheAmLeser erst, wenn feststeht, dass eine Ausleihe entsteht: Wer sein
// Buch zurückgibt, darf gesperrt sein.
func (d *ausleihDienst) ladeVorgang(ctx context.Context, exemplar *repository.Exemplar, leserID *string) (*ausleihVorgang, error) {
	if leserID == nil || *leserID == "" {
		return nil, meldung(ErrUngueltigerZustand, "Kein Leser aktiv")
	}

	leser, err := d.leserAbfragen.Finde(ctx, *leserID)
	if err != nil {
		return nil, err
	}
	if leser == nil {
		return nil, meldung(ErrNichtGefunden, "Aktiver Leser nicht gefunden")
	}

	vorgang := &ausleihVorgang{leserID: *leserID, leser: leser}
	if !vorgang.istSchueler() {
		// Tagesende in der Schul-Zeitzone — dieselbe Normalisierung wie alle Fristen.
		vorgang.frist = schulzeit.TagesEnde(d.heute().AddDate(1, 0, 0))
		return vorgang, nil
	}

	frist, err := d.bestimmeFrist(ctx, exemplar, leser.Klasse)
	if err != nil {
		return nil, err
	}
	vorgang.frist = frist
	return vorgang, nil
}
