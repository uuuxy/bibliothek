package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
		SELECT id, barcode_id, vorname, nachname, rolle::text 
		FROM benutzer 
		WHERE barcode_id = $1 AND rolle = 'lehrer' AND aktiv = true
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
func (s *Server) handleTeacherCheckoutFlow(
	ctx context.Context,
	copy *repository.BookCopy,
	activeLoan *repository.Loan,
	teacherID string,
	staffID string,
	studentRepo repository.StudentRepository,
	loanRepo repository.LoanRepository,
	resp *ActionResponse,
) error {
	var teacher repository.User
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT id, barcode_id, vorname, nachname, rolle::text 
		FROM benutzer 
		WHERE id = $1 AND rolle = 'lehrer' AND aktiv = true 
		LIMIT 1
	`, teacherID).Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
	if err != nil {
		return fmt.Errorf("%w: active teacher profile not found", errNotFound)
	}

	// Subcase A: Book is available -> Checkout
	if activeLoan == nil {
		dueTime := time.Now().AddDate(1, 0, 0) // 1 year checkout
		loan, err := loanRepo.CreateUserLoan(ctx, copy.ID, teacher.ID, staffID, dueTime, true)
		if err != nil {
			return err
		}
		resp.Type = "ausleihe"
		resp.Book = copy
		resp.Teacher = &teacher
		resp.DueDate = &loan.RueckgabeFrist
		return nil
	}

	// Subcase B: Book currently borrowed by this teacher -> Return
	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == teacher.ID {
		err = loanRepo.ReturnLoan(ctx, activeLoan.ID, staffID, false)
		if err != nil {
			return err
		}
		resp.Type = "rueckgabe"
		resp.Book = copy
		resp.Teacher = &teacher
		return nil
	}

	// Subcase C: Book borrowed by a DIFFERENT borrower -> Fremdrückgabe & Checkout
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var prevStudent *repository.Student
	var prevTeacher *repository.User

	if activeLoan.SchuelerID != nil {
		prevStudent, _ = studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		prevTeacher = &repository.User{}
		_ = tx.QueryRow(ctx, "SELECT id, barcode_id, vorname, nachname, rolle::text FROM benutzer WHERE id = $1 LIMIT 1", *activeLoan.AusleiherBenutzerID).Scan(&prevTeacher.ID, &prevTeacher.BarcodeID, &prevTeacher.Vorname, &prevTeacher.Nachname, &prevTeacher.Rolle)
	}

	err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true)
	if err != nil {
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

	resp.Type = "ausleihe"
	resp.Book = copy
	resp.Teacher = &teacher
	resp.DueDate = &loan.RueckgabeFrist
	resp.Fremdrueckgabe = true
	resp.Vorbesitzer = prevStudent
	resp.VorbesitzerUser = prevTeacher
	return nil
}

// handleStudentCheckoutFlow handles checkout/return/fremdrueckgabe flows for a student session.
func (s *Server) handleStudentCheckoutFlow(
	ctx context.Context,
	copy *repository.BookCopy,
	activeLoan *repository.Loan,
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

	// Subcase A: Book is available -> Checkout
	if activeLoan == nil {
		dueTime := calculateDueDate(copy.Titel)
		loan, err := loanRepo.CreateLoan(ctx, copy.ID, student.ID, staffID, dueTime)
		if err != nil {
			return err
		}
		resp.Type = "ausleihe"
		resp.Book = copy
		resp.Student = student
		resp.DueDate = &loan.RueckgabeFrist
		return nil
	}

	// Subcase B: Book currently borrowed by this student -> Return
	if activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == student.ID {
		err = loanRepo.ReturnLoan(ctx, activeLoan.ID, staffID, false)
		if err != nil {
			return err
		}
		resp.Type = "rueckgabe"
		resp.Book = copy
		resp.Student = student
		return nil
	}

	// Subcase C: Book borrowed by a DIFFERENT student -> Fremdrückgabe & Checkout
	if activeLoan.SchuelerID != nil && *activeLoan.SchuelerID != student.ID {
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		prevStudent, err := studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return err
		}

		err = loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true)
		if err != nil {
			return err
		}

		dueTime := calculateDueDate(copy.Titel)
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
		resp.DueDate = &loan.RueckgabeFrist
		resp.Fremdrueckgabe = true
		resp.Vorbesitzer = prevStudent
		return nil
	}

	if activeLoan.SchuelerID == nil && activeLoan.AusleiherBenutzerID != nil {
		err = loanRepo.ReturnLoan(ctx, activeLoan.ID, staffID, false)
		if err != nil {
			return err
		}
		resp.Type = "rueckgabe"
		resp.Book = copy
		return nil
	}

	return fmt.Errorf("%w: book is currently borrowed by a staff member", errInvalidState)
}

// calculateDueDate calculates the return deadline based on the book category.
// If the book title starts with "lmf-" (case-insensitive), it returns the end of the school year (July 31st).
// Otherwise, it returns 4 weeks from now (+28 days).
func calculateDueDate(title string) time.Time {
	now := time.Now()
	if strings.HasPrefix(strings.ToLower(title), "lmf-") {
		year := now.Year()
		if now.Month() >= time.August {
			year++
		}
		return time.Date(year, time.July, 31, 23, 59, 59, 0, now.Location())
	}
	return now.AddDate(0, 0, 28)
}
