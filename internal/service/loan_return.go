package service

import (
	"context"
	"fmt"
	"time"

	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

func (s *defaultLoanService) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *LoanResult) {
	var vID, sVorname, sNachname, sKlasse string
	err := tx.QueryRow(ctx, `
		SELECT v.id, s.vorname, s.nachname, COALESCE(s.klasse, '')
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		ORDER BY v.erstellt_am ASC LIMIT 1
		FOR UPDATE
	`, copy.TitelID).Scan(&vID, &sVorname, &sNachname, &sKlasse)

	if err == nil {
		schuelerName := sVorname + " " + sNachname
		if sKlasse != "" {
			schuelerName += ", " + sKlasse
		}

		_, _ = tx.Exec(ctx, "UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1, bereitgestellt_bis = CURRENT_TIMESTAMP + INTERVAL '3 days' WHERE id = $2", copy.ID, vID)

		resp.HasVormerkung = true
		resp.VormerkungTitel = copy.Titel
		resp.VormerkungUser = schuelerName
	}
}

func (s *defaultLoanService) HandleSimpleReturn(
	ctx context.Context,
	copy *repository.BookCopy,
	staffID string,
	staffRole string,
) (*LoanResult, error) {
	resp := &LoanResult{}

	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	if activeLoan == nil {
		if staffRole == "LEHRER" {
			dueTime := time.Now().AddDate(1, 0, 0) // 1 year
			loan, err := s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, staffID, staffID, dueTime, true)
			if err != nil {
				return nil, err
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", staffID, staffID)

			resp.Type = "ausleihe"
			resp.Book = copy
			if loan != nil {
				resp.DueDate = &loan.RueckgabeFrist
			}
			return resp, nil
		}
		return nil, fmt.Errorf("%w: Dieses Buchexemplar ist aktuell nicht ausgeliehen", ErrInvalidState)
	}

	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == staffID {
		if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return nil, err
		}

		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)

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

	var borrowerStudent *repository.Student
	if activeLoan.SchuelerID != nil {
		borrowerStudent, err = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return nil, err
		}
	}

	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	s.processReturnVormerkungTx(ctx, tx, copy, resp)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if activeLoan.SchuelerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
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
	resp.Student = borrowerStudent
	resp.LoanID = &activeLoan.ID
	return resp, nil
}
