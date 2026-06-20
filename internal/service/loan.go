package service

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

var (
	ErrNotFound     = errors.New("eintrag nicht gefunden")
	ErrBlocked      = errors.New("ausleihe für diese/n Schüler/in ist gesperrt")
	ErrConflict     = errors.New("conflict")
	ErrInvalidState = errors.New("ungültiger Transaktionszustand")
)

// LoanResult is the response from the LoanService.
type LoanResult struct {
	Type            string
	Book            *repository.BookCopy
	Student         *repository.Student
	Teacher         *repository.User
	DueDate         *time.Time
	LoanID          *string
	Fremdrueckgabe  bool
	Vorbesitzer     *repository.Student
	VorbesitzerUser *repository.User
	HasVormerkung   bool
	VormerkungTitel string
	VormerkungUser  string
}

type LoanService interface {
	HandleUnifiedCheckout(ctx context.Context, copy *repository.BookCopy, activeStudentID *string, activeTeacherID *string, staffID string) (*LoanResult, error)
	HandleSimpleReturn(ctx context.Context, copy *repository.BookCopy, staffID string, staffRole string) (*LoanResult, error)
}

type defaultLoanService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	bookRepo    repository.BookRepository
	loanRepo    repository.LoanRepository
	auditRepo   repository.AuditRepository
}

func NewLoanService(pool db.PgxPoolIface, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, loanRepo repository.LoanRepository, auditRepo repository.AuditRepository) LoanService {
	return &defaultLoanService{
		pool:        pool,
		studentRepo: studentRepo,
		bookRepo:    bookRepo,
		loanRepo:    loanRepo,
		auditRepo:   auditRepo,
	}
}
