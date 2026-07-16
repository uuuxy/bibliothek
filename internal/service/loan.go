package service

import (
	"context"
	"errors"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

var (
	// ErrNotFound wird zurückgegeben, wenn ein angeforderter Datensatz (z. B. Schüler oder Buch) nicht existiert.
	ErrNotFound = errors.New("eintrag nicht gefunden")
	// ErrBlocked wird zurückgegeben, wenn eine Ausleihe aufgrund von Sperren (z. B. blockierter Schüler) verweigert wird.
	ErrBlocked = errors.New("ausleihe für diese/n Schüler/in ist gesperrt")
	// ErrConflict wird zurückgegeben, wenn eine Aktion mit bestehenden Reservierungen oder Sperren kollidiert.
	ErrConflict = errors.New("conflict")
	// ErrInvalidState wird zurückgegeben, wenn sich ein Objekt oder eine Transaktion in einem ungültigen Zustand befindet.
	ErrInvalidState = errors.New("ungültiger Transaktionszustand")
)

// LoanResult beschreibt das Ergebnis einer Ausleih- oder Rückgabeoperation eines Buches.
type LoanResult struct {
	// Type spezifiziert die Art des Ergebnisses (z. B. "ausleihe", "rueckgabe", "info").
	Type string
	// Book enthält die Daten des betroffenen Buchexemplars.
	Book *repository.BookCopy
	// Student ist das Schülerprofil, falls die Aktion für einen Schüler durchgeführt wurde.
	Student *repository.Student
	// Teacher ist das Benutzerprofil, falls die Aktion für eine Lehrkraft durchgeführt wurde.
	Teacher *repository.User
	// DueDate gibt das berechnete Rückgabedatum an (nur bei erfolgreicher Ausleihe).
	DueDate *time.Time
	// LoanID ist die eindeutige ID des Ausleihvorgangs.
	LoanID *string
	// Fremdrueckgabe gibt an, ob das Buch von jemand anderem als dem Entleiher zurückgegeben wurde.
	Fremdrueckgabe bool
	// Vorbesitzer ist der Schüler, der das Buch zuvor ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Student
	// VorbesitzerUser ist der Lehrer, der das Buch zuvor ausgeliehen hatte (bei Fremdrückgabe).
	VorbesitzerUser *repository.User
	// HasVormerkung ist wahr, wenn für das Buch eine Vormerkung vorliegt und es nun für den nächsten Schüler bereitgestellt wurde.
	HasVormerkung bool
	// VormerkungTitel ist der Titel des vorgemerkten Buchs.
	VormerkungTitel string
	// VormerkungUser ist der Name (und ggf. Klasse) des Schülers, für den das Buch reserviert wurde.
	VormerkungUser string
	// RegalfreigabeBarcode ist gesetzt, wenn für diesen Schüler ein ANDERES Exemplar
	// desselben Titels im Reservierungsfach lag, er sich aber ein Freihand-Exemplar
	// genommen hat. Das reservierte Exemplar muss zurück ins normale Regal — sonst
	// bleibt es als "Geisterbuch" im Fach liegen, obwohl es laut DB verfügbar ist.
	RegalfreigabeBarcode string
}

// LoanService steuert die Geschäftsregeln und Transaktionen rund um das Ausleihen und Zurückgeben von Büchern.
type LoanService interface {
	// HandleUnifiedCheckout wickelt die Ausleihe eines Buchexemplars an einen Schüler oder Lehrer ab.
	// Falls das Exemplar bereits von jemand anderem ausgeliehen war, wird dieses zuerst automatisch zurückgegeben
	// (Fremdrückgabe) und danach für den neuen Ausleiher verbucht.
	HandleUnifiedCheckout(ctx context.Context, copy *repository.BookCopy, activeStudentID *string, activeTeacherID *string, staffID string, overrideBlock bool) (*LoanResult, error)

	// HandleSimpleReturn wickelt die direkte Rückgabe eines Buchexemplars ab (ohne dass ein neuer Ausleiher aktiv ist).
	// Wenn eine Lehrkraft das Buch scannt und es frei ist, wird eine Ausleihe an diese Lehrkraft als Handapparat initiiert.
	HandleSimpleReturn(ctx context.Context, copy *repository.BookCopy, staffID string, staffRole string) (*LoanResult, error)
}

// defaultLoanService implementiert den LoanService unter Verwendung von Repositories.
type defaultLoanService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	bookRepo    repository.BookRepository
	loanRepo    repository.LoanRepository
	auditRepo   repository.AuditRepository
}

// NewLoanService erzeugt eine neue Instanz des standardmäßigen LoanService.
func NewLoanService(pool db.PgxPoolIface, studentRepo repository.StudentRepository, bookRepo repository.BookRepository, loanRepo repository.LoanRepository, auditRepo repository.AuditRepository) LoanService {
	return &defaultLoanService{
		pool:        pool,
		studentRepo: studentRepo,
		bookRepo:    bookRepo,
		loanRepo:    loanRepo,
		auditRepo:   auditRepo,
	}
}
