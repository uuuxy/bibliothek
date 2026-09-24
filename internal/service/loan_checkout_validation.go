package service

import (
	"context"
	"fmt"
	"time"

	"bibliothek/repository"
)

// checkoutContext holds the resolved borrower information and the due time.
// checkoutContext ist der Ausleiher eines laufenden Vorgangs: EIN Leser (Migration 125).
// Bis dahin standen hier zwei Paare — borrowerType/borrowerID und student/teacher — und
// jede Fallunterscheidung im Ausleihpfad musste beide richtig bedienen.
type checkoutContext struct {
	borrowerID string
	// leser ist der Ausleiher. Über die Theke immer gesetzt; er trägt seine Art.
	leser   *repository.Student
	dueTime time.Time
	// uebergangen sind die Hinweise, die override_block übergangen hat
	// (pruefeAusleihSperren). handleNewLoan protokolliert sie, wenn die Ausleihe steht.
	uebergangen []string
}

// istSchueler sagt, ob die Schülerregeln greifen: Ausleihlimit, Vormerkungen und die
// Frist aus der Klasse. Sie hängen an der ART des Lesers, nicht mehr daran, aus welcher
// Tabelle er kam — das war vorher dasselbe und ist es seit Migration 125 nicht mehr.
func (c *checkoutContext) istSchueler() bool {
	return c.leser != nil && c.leser.Art == "schueler"
}

// resolveBorrowerAndDueTime lädt den aktiven LESER und bestimmt seine Leihfrist.
//
// Die Sperrgründe prüft HandleUnifiedCheckout erst, wenn feststeht, dass eine Ausleihe
// entsteht (pruefeAusleihSperren) — wer sein Buch ZURÜCKgibt, darf gesperrt sein.
//
// Die Frist: Für einen Schüler aus der Klasse (Schuljahresende, Lernmittel abweichend),
// für alle anderen ein Jahr. Das ist unverändert die Regel von vor Migration 125 — dort
// hieß sie „Lehrerausleihe = Dauerleihgabe". Gespeichert wird das Jahr, aber die Ausleihe
// eines Kollegen ist eine Dauerleihe: Sie wird nie überfällig, und die Akte zeigt „ohne
// Frist" (entschieden am 16.09.2026).
func (s *defaultLoanService) resolveBorrowerAndDueTime(ctx context.Context, copy *repository.BookCopy, leserID *string) (*checkoutContext, error) {
	if leserID == nil || *leserID == "" {
		return nil, fmt.Errorf("%w: Kein Leser aktiv", ErrInvalidState)
	}

	leser, err := s.studentRepo.GetLeserByID(ctx, *leserID)
	if err != nil {
		return nil, err
	}
	if leser == nil {
		return nil, fmt.Errorf("%w: Aktiver Leser nicht gefunden", ErrNotFound)
	}

	result := &checkoutContext{borrowerID: *leserID, leser: leser}
	if !result.istSchueler() {
		// Tagesende in der Schul-Zeitzone — dieselbe Normalisierung wie alle Fristen
		// (TagesEndeInSchulzeitzone).
		result.dueTime = TagesEndeInSchulzeitzone(s.heute().AddDate(1, 0, 0))
		return result, nil
	}

	dt, err := s.resolveCheckoutDueDate(ctx, copy, leser.Klasse)
	if err != nil {
		return nil, err
	}
	result.dueTime = dt
	return result, nil
}
