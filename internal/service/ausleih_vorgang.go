package service

import (
	"context"
	"time"

	"bibliothek/pkg/leserart"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// ausleihVorgang holds the resolved borrower information and the due time.
// ausleihVorgang ist der Ausleiher eines laufenden Vorgangs: EIN Leser (Migration 125).
// Bis dahin standen hier zwei Paare — borrowerType/borrowerID und student/teacher — und
// jede Fallunterscheidung im Ausleihpfad musste beide richtig bedienen.
type ausleihVorgang struct {
	leserID string
	// leser ist der Ausleiher. Über die Theke immer gesetzt; er trägt seine Art.
	leser *repository.Leser
	frist time.Time
	// uebergangen sind die Hinweise, die override_block übergangen hat
	// (pruefeAusleihSperren). verarbeiteAusleihe protokolliert sie, wenn die Ausleihe steht.
	uebergangen []string
}

// istSchueler sagt, ob die Schülerregeln greifen: Ausleihlimit, Vormerkungen und die
// Frist aus der Klasse. Sie hängen an der ART des Lesers, nicht mehr daran, aus welcher
// Tabelle er kam — das war vorher dasselbe und ist es seit Migration 125 nicht mehr.
func (v *ausleihVorgang) istSchueler() bool {
	return v.leser != nil && leserart.IstSchueler(v.leser.Art)
}

// ladeVorgang lädt den aktiven LESER und bestimmt seine Leihfrist.
//
// Die Sperrgründe prüft BucheAmLeser erst, wenn feststeht, dass eine Ausleihe
// entsteht (pruefeAusleihSperren) — wer sein Buch ZURÜCKgibt, darf gesperrt sein.
//
// Die Frist: Für einen Schüler aus der Klasse (Schuljahresende, Lernmittel abweichend),
// für alle anderen ein Jahr. Das ist unverändert die Regel von vor Migration 125 — dort
// hieß sie „Lehrerausleihe = Dauerleihgabe". Gespeichert wird das Jahr, aber die Ausleihe
// eines Kollegen ist eine Dauerleihe: Sie wird nie überfällig, und die Akte zeigt „ohne
// Frist" (entschieden am 16.09.2026).
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
