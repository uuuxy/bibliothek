package service

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// zaehleAktiveSchuelerAusleihen setzt einen Row-Level-Lock auf den Schüler
// (FOR UPDATE, gegen parallele Ausleihen) und zählt dessen reguläre (Nicht-LMF-)
// Ausleihen. Für Nicht-Schüler ist das Ergebnis 0.
func (s *defaultLoanService) zaehleAktiveSchuelerAusleihen(ctx context.Context, tx pgx.Tx, chkCtx *checkoutContext) (int, error) {
	if !chkCtx.istSchueler() {
		return 0, nil
	}
	if err := repository.SperreLeserzeile(ctx, tx, chkCtx.borrowerID); err != nil {
		return 0, err
	}
	// Lernmittel zählen nicht ins Limit.
	return repository.ZaehleOffeneBuechereiAusleihen(ctx, tx, chkCtx.borrowerID)
}

// istEigeneRueckgabe erkennt, ob der aktive Ausleiher sein eigenes Buch scannt
// (also eine reguläre Rückgabe statt einer Fremdrückgabe). Eine Spalte, ein Vergleich:
// Vorher waren es zwei, und ob der richtige griff, entschied der borrowerType.
func istEigeneRueckgabe(chkCtx *checkoutContext, activeLoan *repository.Loan) bool {
	if activeLoan == nil {
		return false
	}
	return activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == chkCtx.borrowerID
}

// pruefeSchuelerAusleihlimit erzwingt das Ausleihlimit für Schüler. Ausgenommen sind
// LMF-Bücher und jede Rückgabe — bei einer Rückgabe entsteht keine Ausleihe, die gegen
// das Limit zählen könnte.
func (s *defaultLoanService) pruefeSchuelerAusleihlimit(ctx context.Context, chkCtx *checkoutContext, copy *repository.BookCopy, activeLoansCount int, neueAusleihe bool) error {
	if !chkCtx.istSchueler() {
		return nil
	}
	if !neueAusleihe {
		return nil
	}
	settings, err := s.querySettings(ctx)
	if err != nil {
		return err
	}
	if !copy.IstLernmittel && activeLoansCount >= settings.MaxAusleihenSchueler {
		return fmt.Errorf("%w: Ausleihlimit von %d Büchern überschritten (aktuell: %d)", ErrBlocked, settings.MaxAusleihenSchueler, activeLoansCount)
	}
	return nil
}

// pruefeVormerkungKonflikt blockiert die Ausleihe, wenn das Exemplar für einen
// anderen Schüler abholbereit reserviert ist. Bei jeder Rückgabe entfällt die Prüfung —
// auch bei der Fremdrückgabe, die das Exemplar nur zurücknimmt (die Vormerkung bedient
// danach processReturnVormerkungTx).
func (s *defaultLoanService) pruefeVormerkungKonflikt(ctx context.Context, tx pgx.Tx, copyID string, chkCtx *checkoutContext, neueAusleihe bool) error {
	if !neueAusleihe {
		return nil
	}
	reserviert, bereit, err := repository.ReservierungAmExemplar(ctx, tx, copyID)
	if err != nil {
		return err
	}
	if bereit && (!chkCtx.istSchueler() || chkCtx.borrowerID != reserviert.LeserID) {
		return meldung(ErrConflict, "Achtung: dieses Exemplar ist noch für %s %s reserviert", reserviert.Vorname, reserviert.Nachname)
	}
	return nil
}

// HandleUnifiedCheckout wickelt die Ausleihe eines Buchexemplars an den aktiven LESER ab.
// Wenn das Buch bereits ausgeliehen ist, entscheidet die Methode, ob es sich um eine reguläre Rückgabe handelt
// (Ausleiher scannt sein eigenes Buch) oder um eine Fremdrückgabe mit anschließender Neuausleihe.
func (s *defaultLoanService) HandleUnifiedCheckout(
	ctx context.Context,
	copy *repository.BookCopy,
	activeLeserID *string,
	staffID string,
	overrideBlock bool,
) (*LoanResult, error) {
	resp := &LoanResult{}

	// Sicherheitsschranke: Nur ausleihbare Exemplare dürfen verarbeitet werden
	if !copy.IstAusleihbar {
		return nil, meldung(ErrInvalidState, "Dieses Buchexemplar ist nicht ausleihbar")
	}

	// 1. Ausleiher und Frist auflösen (loan_checkout_validation.go). Die Sperrprüfung folgt
	// erst nach dem Lesen der Ausleihe — sie hängt davon ab, ob das Buch diesem Ausleiher gehört.
	chkCtx, err := s.resolveBorrowerAndDueTime(ctx, copy, activeLeserID)
	if err != nil {
		return nil, err
	}

	// 2. Transaktion gegen Race Conditions
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	activeLoansCount, err := s.zaehleAktiveSchuelerAusleihen(ctx, tx, chkCtx)
	if err != nil {
		return nil, err
	}

	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	isReturningThis := istEigeneRueckgabe(chkCtx, activeLoan)
	// Eine Ausleihe entsteht nur am freien Exemplar. Beide Rückgabe-Fälle nehmen bloß
	// zurück — die Fremdrückgabe kennt den Ausleiher der Sitzung nicht einmal
	// (handleForeignReturn bekommt keinen checkoutContext).
	neueAusleihe := activeLoan == nil

	// 3. Die Schranken der Ausleihe — Sperrgründe, Ausleihlimit, fremde Vormerkung — nur,
	// wenn wirklich eine Ausleihe entsteht. Sie liefen bis zum 11.09.2026 beim Auflösen des
	// Ausleihers, bevor feststand, wem das Buch gehört (ein wegen Überfälligkeit gesperrtes
	// Kind wurde mit genau diesen Büchern abgewiesen), und danach bis zum 12.09.2026 an
	// `!isReturningThis` — was die Fremdrückgabe einschloss: Wer gesperrt oder am Limit war,
	// konnte das Buch eines Mitschülers nicht abgeben, während dieselbe Rückgabe ohne offene
	// Sitzung (HandleSimpleReturn) durchging. Die Ausleihe liegt hier unter FOR UPDATE, der
	// Fall kann sich bis zum Commit nicht mehr ändern.
	//
	// Die Sperren gelten der Art nach (pruefeAusleihSperren: ein Kollege wird nie gesperrt);
	// bis zum 24.09.2026 stand dafür hier ein istSchueler() davor. Gezählt wird über den
	// Pool wie bisher; ein einfaches SELECT wartet auf keine Zeilensperre dieser Transaktion.
	if neueAusleihe {
		lage, err := pruefeAusleihSperren(ctx, s.pool, chkCtx.leser, copy.IstLernmittel, overrideBlock)
		if err != nil {
			return nil, err
		}
		chkCtx.uebergangen = lage.uebergangen
	}

	if err := s.pruefeSchuelerAusleihlimit(ctx, chkCtx, copy, activeLoansCount, neueAusleihe); err != nil {
		return nil, err
	}

	if err := s.pruefeVormerkungKonflikt(ctx, tx, copy.ID, chkCtx, neueAusleihe); err != nil {
		return nil, err
	}

	// 5. Fall-Logik ausführen (loan_checkout_cases.go)
	if activeLoan == nil {
		return s.handleNewLoan(ctx, tx, copy, chkCtx, staffID, resp)
	}
	if isReturningThis {
		return s.handleReturn(ctx, tx, copy, chkCtx, activeLoan, staffID, resp)
	}
	return s.handleForeignReturn(ctx, tx, copy, activeLoan, staffID, resp)
}
