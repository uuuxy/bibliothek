package api

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

// handleUnifiedCheckoutFlow handles checkout/return/fremdrueckgabe flows.
// Every path is wrapped in a Read Committed transaction with SELECT ... FOR UPDATE on the loan
// row to serialise concurrent scans of the same book copy.
func (s *Server) handleUnifiedCheckoutFlow(
	ctx context.Context,
	copy *repository.BookCopy,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
	studentRepo repository.StudentRepository,
	loanRepo repository.LoanRepository,
	resp *ActionResponse,
) error {
	if !copy.IstAusleihbar {
		return fmt.Errorf("%w: dieses Buchexemplar ist nicht ausleihbar", errInvalidState)
	}

	var borrowerID string
	var borrowerType string
	var student *repository.Student
	var teacher *repository.User
	var dueTime time.Time

	// 1. Resolve active borrower and validate limits
	if activeStudentID != nil && *activeStudentID != "" {
		borrowerType = "student"
		borrowerID = *activeStudentID
		sObj, err := studentRepo.GetByID(ctx, borrowerID)
		if err != nil {
			return err
		}
		if sObj == nil {
			return fmt.Errorf("%w: Aktives Schülerprofil nicht gefunden", errNotFound)
		}
		if sObj.IstGesperrt {
			return fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", errBlocked)
		}
		student = sObj
		dt, err := s.resolveCheckoutDueDate(ctx, copy)
		if err != nil {
			return err
		}
		dueTime = dt
	} else if activeTeacherID != nil && *activeTeacherID != "" {
		borrowerType = "teacher"
		borrowerID = *activeTeacherID
		teacher = &repository.User{}
		err := s.DB.Pool.QueryRow(ctx, `
			SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
			FROM benutzer b JOIN benutzer_rollen br ON b.id = br.benutzer_id
			WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true LIMIT 1
		`, borrowerID).Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
		if err != nil {
			return fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", errNotFound)
		}
		dueTime = time.Now().AddDate(1, 0, 0)
	} else {
		return fmt.Errorf("%w: Weder Schüler noch Lehrer aktiv", errInvalidState)
	}

	// 2. Open Transaction
	tx, err := loanRepo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// If student, lock row to prevent concurrent limit bypass
	var activeLoansCount int
	if borrowerType == "student" {
		if _, err = tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", borrowerID); err != nil {
			return err
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
			return err
		}
	}

	// 3. Get Active Loan (FOR UPDATE)
	activeLoan, err := loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return err
	}

	isReturningThis := false
	if activeLoan != nil {
		if borrowerType == "student" && activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == borrowerID {
			isReturningThis = true
		} else if borrowerType == "teacher" && activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == borrowerID {
			isReturningThis = true
		}
	}

	// Check limit early for students
	if borrowerType == "student" {
		settings, err := s.querySettings(ctx)
		if err != nil {
			return err
		}
		
		isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")
		
		if !isLMF && activeLoansCount >= settings.MaxAusleihenSchueler {
			// Only allow return if they already borrowed THIS book
			if !isReturningThis {
				return fmt.Errorf("%w: Ausleihlimit von %d Bibliotheks-Büchern überschritten (aktuell: %d)", errBlocked, settings.MaxAusleihenSchueler, activeLoansCount)
			}
		}
	}

	// 4. Reservation Blocker (Checkout only)
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

		if err == nil { // Found an active reservation for this copy
			if borrowerType != "student" || borrowerID != reservedSchuelerID {
				return fmt.Errorf("%w: Achtung: Dieses Exemplar ist noch für %s %s reserviert!", errConflict, resVorname, resNachname)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}

	auditRepo := repository.NewAuditRepository(s.DB.Pool)

	// Subcase A: Available -> Checkout
	if activeLoan == nil {
		var loan *repository.Loan
		if borrowerType == "student" {
			loan, err = loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
		} else {
			loan, err = loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
		}
		if err != nil {
			return err
		}
		if borrowerType == "student" {
			_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		if borrowerType == "student" {
			_ = auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
		}

		resp.Type = "ausleihe"
		resp.Book = copy
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		return nil
	}

	// Subcase B: Borrowed by THIS user -> Return
	if isReturningThis {
		if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return err
		}

		// Process reservation BEFORE commit so the tx statements take effect
		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		if borrowerType == "student" {
			_ = auditRepo.LogRueckgabe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = auditRepo.LogRueckgabe(ctx, copy.ID, "", borrowerID, staffID)
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
		return nil
	}

	// Subcase C: Borrowed by DIFFERENT user -> Fremdrückgabe & Checkout
	var prevStudent *repository.Student
	var prevTeacher *repository.User

	if activeLoan.SchuelerID != nil {
		prevStudent, _ = studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		prevTeacher = &repository.User{}
		err = tx.QueryRow(ctx, "SELECT b.id, b.vorname, b.nachname, COALESCE(br.rolle, 'HELFER') FROM benutzer b LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id WHERE b.id = $1 LIMIT 1", *activeLoan.AusleiherBenutzerID).Scan(&prevTeacher.ID, &prevTeacher.Vorname, &prevTeacher.Nachname, &prevTeacher.Rolle)
		if errors.Is(err, pgx.ErrNoRows) {
			prevTeacher = nil
		}
	}

	if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
		return err
	}

	var loan *repository.Loan
	if borrowerType == "student" {
		loan, err = loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
	} else {
		loan, err = loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
	}
	if err != nil {
		return err
	}

	if borrowerType == "student" {
		_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if activeLoan.SchuelerID != nil {
		_ = auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
	}

	if borrowerType == "student" {
		_ = auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
		resp.Student = student
	} else {
		_ = auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
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
	return nil
}

// processReturnVormerkungTx checks if there's a pending reservation for the returned book copy.
// If so, it updates the reservation, sets the book as reserved, and populates the response.
func (s *Server) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *ActionResponse) {
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
