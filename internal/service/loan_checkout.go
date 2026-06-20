package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

func (s *defaultLoanService) HandleUnifiedCheckout(
	ctx context.Context,
	copy *repository.BookCopy,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
) (*LoanResult, error) {
	resp := &LoanResult{}

	if !copy.IstAusleihbar {
		return nil, fmt.Errorf("%w: dieses Buchexemplar ist nicht ausleihbar", ErrInvalidState)
	}

	var borrowerID string
	var borrowerType string
	var student *repository.Student
	var teacher *repository.User
	var dueTime time.Time

	if activeStudentID != nil && *activeStudentID != "" {
		borrowerType = "student"
		borrowerID = *activeStudentID
		sObj, err := s.studentRepo.GetByID(ctx, borrowerID)
		if err != nil {
			return nil, err
		}
		if sObj == nil {
			return nil, fmt.Errorf("%w: Aktives Schülerprofil nicht gefunden", ErrNotFound)
		}
		if sObj.IstGesperrt {
			return nil, fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
		}
		student = sObj
		dt, err := s.resolveCheckoutDueDate(ctx, copy)
		if err != nil {
			return nil, err
		}
		dueTime = dt
	} else if activeTeacherID != nil && *activeTeacherID != "" {
		borrowerType = "teacher"
		borrowerID = *activeTeacherID
		teacher = &repository.User{}
		err := s.pool.QueryRow(ctx, `
			SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
			FROM benutzer b JOIN benutzer_rollen br ON b.id = br.benutzer_id
			WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true LIMIT 1
		`, borrowerID).Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
		if err != nil {
			return nil, fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", ErrNotFound)
		}
		dueTime = time.Now().AddDate(1, 0, 0)
	} else {
		return nil, fmt.Errorf("%w: Weder Schüler noch Lehrer aktiv", ErrInvalidState)
	}

	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var activeLoansCount int
	if borrowerType == "student" {
		if _, err = tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", borrowerID); err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM ausleihen a
			JOIN buecher_exemplare be ON a.exemplar_id = be.id
			JOIN buecher_titel bt ON be.titel_id = bt.id
			WHERE a.schueler_id = $1 
			  AND a.rueckgabe_am IS NULL
			  AND LOWER(bt.titel) NOT LIKE 'lmf-%'
		`, borrowerID).Scan(&activeLoansCount)
		if err != nil {
			return nil, err
		}
	}

	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	isReturningThis := false
	if activeLoan != nil {
		if borrowerType == "student" && activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == borrowerID {
			isReturningThis = true
		} else if borrowerType == "teacher" && activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == borrowerID {
			isReturningThis = true
		}
	}

	if borrowerType == "student" {
		settings, err := s.querySettings(ctx)
		if err != nil {
			return nil, err
		}

		isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")

		if !isLMF && activeLoansCount >= settings.MaxAusleihenSchueler {
			if !isReturningThis {
				return nil, fmt.Errorf("%w: Ausleihlimit von %d Bibliotheks-Büchern überschritten (aktuell: %d)", ErrBlocked, settings.MaxAusleihenSchueler, activeLoansCount)
			}
		}
	}

	if !isReturningThis {
		var reservedSchuelerID string
		var resVorname, resNachname string
		err = tx.QueryRow(ctx, `
			SELECT v.schueler_id, s.vorname, s.nachname
			FROM vormerkungen v
			JOIN schueler s ON v.schueler_id = s.id
			WHERE v.bereitgestellt_exemplar_id = $1 
			  AND v.status = 'abholbereit' 
			  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
		`, copy.ID).Scan(&reservedSchuelerID, &resVorname, &resNachname)

		if err == nil {
			if borrowerType != "student" || borrowerID != reservedSchuelerID {
				return nil, fmt.Errorf("%w: Achtung: Dieses Exemplar ist noch für %s %s reserviert!", ErrConflict, resVorname, resNachname)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	if activeLoan == nil {
		var loan *repository.Loan
		if borrowerType == "student" {
			loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
		} else {
			loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
		}
		if err != nil {
			return nil, err
		}
		if borrowerType == "student" {
			_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		if borrowerType == "student" {
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
		}

		resp.Type = "ausleihe"
		resp.Book = copy
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		return resp, nil
	}

	if isReturningThis {
		if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return nil, err
		}

		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		if borrowerType == "student" {
			_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
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

	var prevStudent *repository.Student
	var prevTeacher *repository.User

	if activeLoan.SchuelerID != nil {
		prevStudent, _ = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
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
	if borrowerType == "student" {
		loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
	} else {
		loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
	}
	if err != nil {
		return nil, err
	}

	if borrowerType == "student" {
		_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if activeLoan.SchuelerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
	}

	if borrowerType == "student" {
		_ = s.auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
		resp.Student = student
	} else {
		_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
		resp.Teacher = teacher
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
