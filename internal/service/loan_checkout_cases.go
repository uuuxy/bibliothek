package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// mapLoanCreateErr übersetzt eine Unique-Verletzung (Migration 033: höchstens eine aktive
// Ausleihe je Exemplar/Gerät) in einen sauberen Konflikt. Das tritt auf, wenn ein zweiter
// zeitgleicher Checkout dasselbe Exemplar greifen will — dann ist 409 (statt 500) korrekt.
func mapLoanCreateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: dieses Exemplar wurde soeben bereits ausgeliehen", ErrConflict)
	}
	return err
}

// handleNewLoan handles the case where the book is currently available (not checked out).
func (s *defaultLoanService) handleNewLoan(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
	staffID string,
	resp *LoanResult,
) (*LoanResult, error) {
	var loan *repository.Loan
	var err error

	if chkCtx.borrowerType == "student" {
		loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, chkCtx.borrowerID, staffID, chkCtx.dueTime)
	} else {
		loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, chkCtx.borrowerID, staffID, chkCtx.dueTime, true)
	}
	if err != nil {
		return nil, mapLoanCreateErr(err)
	}

	if chkCtx.borrowerType == "student" {
		if _, err := tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, chkCtx.borrowerID); err != nil {
			log.Printf("ausleihe: Vormerkung für Titel %s konnte nicht entfernt werden: %v", copy.TitelID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if chkCtx.borrowerType == "student" {
		logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, copy.ID, chkCtx.borrowerID, "", staffID))
		resp.Student = chkCtx.student
	} else {
		logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, copy.ID, "", chkCtx.borrowerID, staffID))
		resp.Teacher = chkCtx.teacher
	}

	resp.Type = "ausleihe"
	resp.Book = copy
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	return resp, nil
}

// handleReturn handles the case where the active user returns their own book.
func (s *defaultLoanService) handleReturn(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
	activeLoan *repository.Loan,
	staffID string,
	resp *LoanResult,
) (*LoanResult, error) {
	if err := s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	s.processReturnVormerkungTx(ctx, tx, copy, resp)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if chkCtx.borrowerType == "student" {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, copy.ID, chkCtx.borrowerID, "", staffID))
		resp.Student = chkCtx.student
	} else {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, copy.ID, "", chkCtx.borrowerID, staffID))
		resp.Teacher = chkCtx.teacher
	}

	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       copy.ID,
		BarcodeID:    copy.BarcodeID,
		Titel:        copy.Titel,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: staffID,
	})

	resp.Type = "rueckgabe"
	resp.Book = copy
	resp.LoanID = &activeLoan.ID
	return resp, nil
}

// handleForeignReturn handles the complex case where user B checks out a book currently checked out by user A.
func (s *defaultLoanService) handleForeignReturn(
	ctx context.Context,
	tx pgx.Tx,
	copy *repository.BookCopy,
	chkCtx *checkoutContext,
	activeLoan *repository.Loan,
	staffID string,
	resp *LoanResult,
) (*LoanResult, error) {
	var prevStudent *repository.Student
	var prevTeacher *repository.User
	var err error

	if activeLoan.SchuelerID != nil {
		prevStudent, err = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			log.Printf("fremdrückgabe: Vorbesitzer (Schüler) konnte nicht geladen werden: %v", err)
		}
	} else if activeLoan.AusleiherBenutzerID != nil {
		prevTeacher = &repository.User{}
		err = tx.QueryRow(ctx, "SELECT b.id, b.vorname, b.nachname, COALESCE(br.rolle, 'HELFER') FROM benutzer b LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id WHERE b.id = $1 LIMIT 1", *activeLoan.AusleiherBenutzerID).Scan(&prevTeacher.ID, &prevTeacher.Vorname, &prevTeacher.Nachname, &prevTeacher.Rolle)
		if errors.Is(err, pgx.ErrNoRows) {
			prevTeacher = nil
		}
	}

	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
		return nil, err
	}

	var loan *repository.Loan
	if chkCtx.borrowerType == "student" {
		loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, chkCtx.borrowerID, staffID, chkCtx.dueTime)
	} else {
		loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, chkCtx.borrowerID, staffID, chkCtx.dueTime, true)
	}
	if err != nil {
		return nil, mapLoanCreateErr(err)
	}

	if chkCtx.borrowerType == "student" {
		if _, err := tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, chkCtx.borrowerID); err != nil {
			log.Printf("ausleihe: Vormerkung für Titel %s konnte nicht entfernt werden: %v", copy.TitelID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if activeLoan.SchuelerID != nil {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID))
	} else if activeLoan.AusleiherBenutzerID != nil {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID))
	}

	if chkCtx.borrowerType == "student" {
		logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, copy.ID, chkCtx.borrowerID, "", staffID))
		resp.Student = chkCtx.student
	} else {
		logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, copy.ID, "", chkCtx.borrowerID, staffID))
		resp.Teacher = chkCtx.teacher
	}

	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       copy.ID,
		BarcodeID:    copy.BarcodeID,
		Titel:        copy.Titel,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: staffID,
	})

	resp.Type = "ausleihe"
	resp.Book = copy
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	resp.Fremdrueckgabe = true
	resp.Vorbesitzer = prevStudent
	resp.VorbesitzerUser = prevTeacher
	return resp, nil
}
