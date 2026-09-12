package service

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// zaehleAktiveSchuelerAusleihen setzt einen Row-Level-Lock auf den Schüler
// (FOR UPDATE, gegen parallele Ausleihen) und zählt dessen reguläre (Nicht-LMF-)
// Ausleihen. Für Nicht-Schüler ist das Ergebnis 0.
func (s *defaultLoanService) zaehleAktiveSchuelerAusleihen(ctx context.Context, tx pgx.Tx, chkCtx *checkoutContext) (int, error) {
	if chkCtx.borrowerType != "student" {
		return 0, nil
	}
	if _, err := tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", chkCtx.borrowerID); err != nil {
		return 0, err
	}
	var count int
	// Lernmittel zählen nicht ins Limit (buecher_titel.ist_lernmittel, Migration 093).
	const query = `
		SELECT COUNT(*)
		FROM ausleihen a
		JOIN buecher_exemplare be ON a.exemplar_id = be.id
		JOIN buecher_titel bt ON be.titel_id = bt.id
		WHERE a.schueler_id = $1
		  AND a.rueckgabe_am IS NULL
		  AND NOT bt.ist_lernmittel
	`
	err := tx.QueryRow(ctx, query, chkCtx.borrowerID).Scan(&count)
	return count, err
}

// istEigeneRueckgabe erkennt, ob der aktive Ausleiher sein eigenes Buch scannt
// (also eine reguläre Rückgabe statt einer Fremdrückgabe).
func istEigeneRueckgabe(chkCtx *checkoutContext, activeLoan *repository.Loan) bool {
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

// pruefeSchuelerAusleihlimit erzwingt das Ausleihlimit für Schüler. Ausgenommen sind
// LMF-Bücher und jede Rückgabe — bei einer Rückgabe entsteht keine Ausleihe, die gegen
// das Limit zählen könnte.
func (s *defaultLoanService) pruefeSchuelerAusleihlimit(ctx context.Context, chkCtx *checkoutContext, copy *repository.BookCopy, activeLoansCount int, neueAusleihe bool) error {
	if chkCtx.borrowerType != "student" {
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
	var reservedSchuelerID, resVorname, resNachname string
	err := tx.QueryRow(ctx, `
		SELECT v.schueler_id, s.vorname, s.nachname
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.bereitgestellt_exemplar_id = $1
		  AND v.status = 'abholbereit'
		  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
	`, copyID).Scan(&reservedSchuelerID, &resVorname, &resNachname)
	if err == nil {
		if chkCtx.borrowerType != "student" || chkCtx.borrowerID != reservedSchuelerID {
			return fmt.Errorf("%w: Achtung: dieses Exemplar ist noch für %s %s reserviert", ErrConflict, resVorname, resNachname)
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}

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

	// 1. Ausleiher und Frist auflösen (loan_checkout_validation.go). Die Sperrprüfung folgt
	// erst nach dem Lesen der Ausleihe — sie hängt davon ab, ob das Buch diesem Ausleiher gehört.
	chkCtx, err := s.resolveBorrowerAndDueTime(ctx, copy, activeStudentID, activeTeacherID)
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
	if chkCtx.student != nil && neueAusleihe {
		if err := s.pruefeSchuelerAusleihbar(ctx, chkCtx.student, chkCtx.borrowerID, staffID, overrideBlock); err != nil {
			return nil, err
		}
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
