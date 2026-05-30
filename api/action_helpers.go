package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// handleStudentAction loads student details by scanning card barcodes.
func (s *Server) handleStudentAction(ctx context.Context, query string, repo repository.StudentRepository, resp *ActionResponse) error {
	student, err := repo.GetByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if student == nil {
		return fmt.Errorf("%w: student barcode %s not registered", errNotFound, query)
	}
	resp.Type = "student"
	resp.Student = student
	return nil
}

// handleSearchAction queries book catalog using full-text search triggers.
func (s *Server) handleSearchAction(ctx context.Context, query string, repo repository.BookRepository, resp *ActionResponse) error {
	titles, err := repo.SearchTitles(ctx, query)
	if err != nil {
		return err
	}
	resp.Type = "search_results"
	resp.SearchResults = titles
	return nil
}

// handleTeacherAction loads teacher details by scanning card barcodes.
func (s *Server) handleTeacherAction(ctx context.Context, query string, resp *ActionResponse) error {
	q := `
		SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
		FROM benutzer b 
		JOIN benutzer_rollen br ON b.id = br.benutzer_id
		WHERE b.barcode_id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true
		LIMIT 1
	`
	var u repository.User
	err := s.DB.Pool.QueryRow(ctx, q, query).Scan(&u.ID, &u.BarcodeID, &u.Vorname, &u.Nachname, &u.Rolle)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: teacher barcode %s not registered or active", errNotFound, query)
		}
		return err
	}
	resp.Type = "teacher"
	resp.Teacher = &u
	return nil
}

// handleTeacherCheckoutFlow handles checkout/return/fremdrueckgabe flows for a teacher session.
// Every path is wrapped in a Read Committed transaction with SELECT ... FOR UPDATE on the loan
// row to serialise concurrent scans of the same book copy across 8 parallel scanning clients.
func (s *Server) handleTeacherCheckoutFlow(
	ctx context.Context,
	copy *repository.BookCopy,
	_ *repository.Loan, // pre-fetched activeLoan is discarded; we re-read inside the tx with FOR UPDATE
	teacherID string,
	staffID string,
	studentRepo repository.StudentRepository,
	loanRepo repository.LoanRepository,
	resp *ActionResponse,
) error {
	var teacher repository.User
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
		FROM benutzer b 
		JOIN benutzer_rollen br ON b.id = br.benutzer_id
		WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true 
		LIMIT 1
	`, teacherID).Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
	if err != nil {
		return fmt.Errorf("%w: active teacher profile not found", errNotFound)
	}

	// Open a Read Committed transaction for the entire checkout/return decision.
	// SELECT ... FOR UPDATE inside GetActiveLoanByCopyIDTx prevents concurrent scanners
	// from racing on the same exemplar within a WLAN-lag window.
	tx, err := loanRepo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Re-read the active loan with a row-level lock to prevent race conditions.
	activeLoan, err := loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return err
	}

	// Subcase A: Book is available -> Checkout
	if activeLoan == nil {
		dueTime := time.Now().AddDate(1, 0, 0) // 1 year checkout
		loan, err := loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, teacher.ID, staffID, dueTime, true)
		if err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		resp.Type = "ausleihe"
		resp.Book = copy
		resp.Teacher = &teacher
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		return nil
	}

	// Subcase B: Book currently borrowed by this teacher -> Return
	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == teacher.ID {
		if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
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
		resp.Teacher = &teacher
		return nil
	}

	// Subcase C: Book borrowed by a DIFFERENT borrower -> Fremdrückgabe & Checkout
	var prevStudent *repository.Student
	var prevTeacher *repository.User

	if activeLoan.SchuelerID != nil {
		prevStudent, _ = studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		prevTeacher = &repository.User{}
		_ = tx.QueryRow(ctx, `
			SELECT b.id, b.barcode_id, b.vorname, b.nachname, COALESCE(br.rolle, 'HELFER') 
			FROM benutzer b 
			LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id 
			WHERE b.id = $1 
			LIMIT 1
		`, *activeLoan.AusleiherBenutzerID).Scan(&prevTeacher.ID, &prevTeacher.BarcodeID, &prevTeacher.Vorname, &prevTeacher.Nachname, &prevTeacher.Rolle)
	}

	if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
		return err
	}

	dueTime := time.Now().AddDate(1, 0, 0)
	loan, err := loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, teacher.ID, staffID, dueTime, true)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
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
	resp.Teacher = &teacher
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	resp.Fremdrueckgabe = true
	resp.Vorbesitzer = prevStudent
	resp.VorbesitzerUser = prevTeacher
	return nil
}

// handleStudentCheckoutFlow handles checkout/return/fremdrueckgabe flows for a student session.
// Every path is wrapped in a Read Committed transaction with SELECT ... FOR UPDATE on the loan
// row to serialise concurrent scans of the same book copy across 8 parallel scanning clients.
func (s *Server) handleStudentCheckoutFlow(
	ctx context.Context,
	copy *repository.BookCopy,
	_ *repository.Loan, // pre-fetched activeLoan is discarded; we re-read inside the tx with FOR UPDATE
	studentID string,
	staffID string,
	studentRepo repository.StudentRepository,
	loanRepo repository.LoanRepository,
	resp *ActionResponse,
) error {
	student, err := studentRepo.GetByID(ctx, studentID)
	if err != nil {
		return err
	}
	if student == nil {
		return fmt.Errorf("%w: active student profile not found", errNotFound)
	}
	if student.IstGesperrt {
		return fmt.Errorf("%w: borrowing suspended for this student", errBlocked)
	}

	// Open a Read Committed transaction for the entire checkout/return decision.
	// SELECT ... FOR UPDATE inside GetActiveLoanByCopyIDTx prevents concurrent scanners
	// from racing on the same exemplar within a WLAN-lag window.
	tx, err := loanRepo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Re-read the active loan with a row-level lock inside the transaction.
	activeLoan, err := loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return err
	}

	// Subcase A: Book is available -> Checkout
	if activeLoan == nil {
		dueTime, err := s.resolveCheckoutDueDate(ctx, copy)
		if err != nil {
			return err
		}
		loan, err := loanRepo.CreateLoanTx(ctx, tx, copy.ID, student.ID, staffID, dueTime)
		if err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		resp.Type = "ausleihe"
		resp.Book = copy
		resp.Student = student
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		return nil
	}

	// Subcase B: Book currently borrowed by this student -> Return
	if activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == student.ID {
		if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
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
		resp.Student = student
		resp.LoanID = &activeLoan.ID
		return nil
	}

	// Subcase C: Book borrowed by a DIFFERENT student -> Fremdrückgabe & Checkout
	if activeLoan.SchuelerID != nil && *activeLoan.SchuelerID != student.ID {
		prevStudent, err := studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return err
		}

		if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
			return err
		}

		dueTime, err := s.resolveCheckoutDueDate(ctx, copy)
		if err != nil {
			return err
		}
		loan, err := loanRepo.CreateLoanTx(ctx, tx, copy.ID, student.ID, staffID, dueTime)
		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
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
		resp.Student = student
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		resp.Fremdrueckgabe = true
		resp.Vorbesitzer = prevStudent
		return nil
	}

	if activeLoan.SchuelerID == nil && activeLoan.AusleiherBenutzerID != nil {
		if err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
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
		return nil
	}

	return fmt.Errorf("%w: book is currently borrowed by a staff member", errInvalidState)
}

// calculateDueDate calculates the return deadline based on media type and title prefix.
// - Titles prefixed with "lmf-" (case-insensitive) return the end of the school year (lmfStichtag, MM-DD format).
// - Media of type CD, DVD, or Audio get 7 days.
// - All other media default to 21 days.
func calculateDueDate(titel, medientyp, lmfStichtag string) time.Time {
	now := time.Now()
	if strings.HasPrefix(strings.ToLower(titel), "lmf-") {
		year := now.Year()
		if now.Month() >= time.August {
			year++
		}
		month := time.July
		day := 31
		parts := strings.SplitN(lmfStichtag, "-", 2)
		if len(parts) == 2 {
			m, err1 := strconv.Atoi(parts[0])
			d, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && m >= 1 && m <= 12 && d >= 1 && d <= 31 {
				month = time.Month(m)
				day = d
			}
		}
		return time.Date(year, month, day, 23, 59, 59, 0, now.Location())
	}
	lower := strings.ToLower(medientyp)
	if strings.Contains(lower, "cd") || strings.Contains(lower, "dvd") || strings.Contains(lower, "audio") {
		return now.AddDate(0, 0, 7)
	}
	return now.AddDate(0, 0, 21)
}

// resolveCheckoutDueDate calculates the due date for a student checkout, honouring the
// Ferien-Leseclub override from system settings when active.
func (s *Server) resolveCheckoutDueDate(ctx context.Context, copy *repository.BookCopy) (time.Time, error) {
	settings, err := s.querySettings(ctx)
	if err != nil {
		// Fall back to default calculation rather than blocking every checkout.
		return calculateDueDate(copy.Titel, copy.Medientyp, "07-31"), nil
	}
	if settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
			return end, nil
		}
	}
	return calculateDueDate(copy.Titel, copy.Medientyp, settings.LmfStichtag), nil
}
