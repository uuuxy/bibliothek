package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// HandleUnifiedCheckout wickelt die Ausleihe eines Buchexemplars an einen aktiven Schüler oder Lehrer ab.
// Wenn das Buch bereits ausgeliehen ist, entscheidet die Methode, ob es sich um eine reguläre Rückgabe handelt
// (Ausleiher scannt sein eigenes Buch) oder um eine Fremdrückgabe mit anschließender Neuausleihe.
func (s *defaultLoanService) HandleUnifiedCheckout(
	ctx context.Context,
	copy *repository.BookCopy,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
	overrideBlock bool,
) (*LoanResult, error) {
	resp := &LoanResult{}

	// Sicherheitsschranke: Nur ausleihbare Exemplare dürfen verarbeitet werden
	if !copy.IstAusleihbar {
		return nil, fmt.Errorf("%w: dieses Buchexemplar ist nicht ausleihbar", ErrInvalidState)
	}

	// 1. Borrower-Validation auslagern in loan_checkout_validation.go
	chkCtx, err := s.resolveBorrowerAndDueTime(ctx, copy, activeStudentID, activeTeacherID, staffID, overrideBlock)
	if err != nil {
		return nil, err
	}

	// 2. Datenbanktransaktion starten, um Nebenläufigkeitsfehler (Race Conditions) zu verhindern
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	activeLoansCount, err := lockAndCountStudentLoans(ctx, tx, chkCtx)
	if err != nil {
		return nil, err
	}

	// Aktuellen Ausleihstatus dieses Buchexemplars in der Transaktion prüfen
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	isReturningThis := isReturningActiveLoan(chkCtx, activeLoan)

	// 3. Ausleihlimit für Schüler prüfen
	if err := s.checkStudentLoanLimit(ctx, chkCtx, copy, activeLoansCount, isReturningThis); err != nil {
		return nil, err
	}

	// 4. Reservierungsprüfung (Vormerkung)
	if err := checkReservationTx(ctx, tx, copy, chkCtx, isReturningThis); err != nil {
		return nil, err
	}

	// 5. Entsprechende Transaction-Fall-Logik ausführen (in loan_checkout_cases.go)
	if activeLoan == nil {
		return s.handleNewLoan(ctx, tx, copy, chkCtx, staffID, resp)
	}

	if isReturningThis {
		return s.handleReturn(ctx, tx, copy, chkCtx, activeLoan, staffID, resp)
	}

	return s.handleForeignReturn(ctx, tx, copy, chkCtx, activeLoan, staffID, resp)
}

func lockAndCountStudentLoans(ctx context.Context, tx pgx.Tx, chkCtx *checkoutContext) (int, error) {
	if chkCtx.borrowerType != "student" {
		return 0, nil
	}

	// Row-Level Lock auf den Schülereintrag setzen (`FOR UPDATE`), um zeitgleiche Ausleihen
	// auf denselben Schüler im parallelen Request zu synchronisieren.
	if _, err := tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", chkCtx.borrowerID); err != nil {
		return 0, err
	}

	var activeLoansCount int
	// Ermitteln, wie viele reguläre Bücher der Schüler aktuell besitzt
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen a
		JOIN buecher_exemplare be ON a.exemplar_id = be.id
		JOIN buecher_titel bt ON be.titel_id = bt.id
		WHERE a.schueler_id = $1
		  AND a.rueckgabe_am IS NULL
		  AND LOWER(bt.titel) NOT LIKE 'lmf-%'
	`, chkCtx.borrowerID).Scan(&activeLoansCount)

	return activeLoansCount, err
}

func isReturningActiveLoan(chkCtx *checkoutContext, activeLoan *repository.Loan) bool {
	if activeLoan == nil {
		return false
	}
	if chkCtx.borrowerType == "student" && activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == chkCtx.borrowerID {
		return true
	}
	if chkCtx.borrowerType == "teacher" && activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == chkCtx.borrowerID {
		return true
	}
	return false
}

func (s *defaultLoanService) checkStudentLoanLimit(ctx context.Context, chkCtx *checkoutContext, copy *repository.BookCopy, activeLoansCount int, isReturningThis bool) error {
	if chkCtx.borrowerType != "student" {
		return nil
	}
	settings, err := s.querySettings(ctx)
	if err != nil {
		return err
	}
	isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")
	if !isLMF && activeLoansCount >= settings.MaxAusleihenSchueler {
		if !isReturningThis {
			return fmt.Errorf("%w: Ausleihlimit von %d Büchern überschritten (aktuell: %d)", ErrBlocked, settings.MaxAusleihenSchueler, activeLoansCount)
		}
	}
	return nil
}

func checkReservationTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, chkCtx *checkoutContext, isReturningThis bool) error {
	if isReturningThis {
		return nil
	}
	var reservedSchuelerID string
	var resVorname, resNachname string
	err := tx.QueryRow(ctx, `
		SELECT v.schueler_id, s.vorname, s.nachname
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.bereitgestellt_exemplar_id = $1
		  AND v.status = 'abholbereit'
		  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
	`, copy.ID).Scan(&reservedSchuelerID, &resVorname, &resNachname)

	if err == nil {
		if chkCtx.borrowerType != "student" || chkCtx.borrowerID != reservedSchuelerID {
			return fmt.Errorf("%w: Achtung: dieses Exemplar ist noch für %s %s reserviert", ErrConflict, resVorname, resNachname)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}
