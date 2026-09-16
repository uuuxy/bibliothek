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

// SperrGrundFehler trennt den Sperr-Freitext (schueler.block_reason) vom
// generischen Teil der Meldung. Der Freitext ist Verwaltungsinformation —
// er kann Zahlungsrückstände oder Familieninterna nennen (PII-Matrix Stufe 2)
// und darf Aufrufer ohne view_students nicht erreichen. Die HTTP-Schicht
// entscheidet anhand dieses Typs (errors.As), ob sie den Grund mitschickt;
// Error() liefert weiterhin die volle Meldung für berechtigte Aufrufer & Logs.
type SperrGrundFehler struct {
	Kern  error  // ErrBlocked-Kette mit generischem Text ("…: Manuelle Sperre")
	Grund string // Freitext aus schueler.block_reason
}

func (e *SperrGrundFehler) Error() string { return e.Kern.Error() + ": " + e.Grund }

// Unwrap hält errors.Is(err, ErrBlocked) am Leben — der HTTP-Status bleibt 403.
func (e *SperrGrundFehler) Unwrap() error { return e.Kern }

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
	// Vorbesitzer ist der LESER, der das Buch zuvor ausgeliehen hatte (bei Fremdrückgabe).
	// Ein zweites Feld für Lehrkräfte gab es bis Migration 125; es gibt nur noch Leser.
	Vorbesitzer *repository.Student
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
	// HandleUnifiedCheckout wickelt die Ausleihe eines Buchexemplars an den aktiven Leser ab.
	// Falls das Exemplar bereits von jemand anderem ausgeliehen war, wird dieses zuerst automatisch zurückgegeben
	// (Fremdrückgabe) und danach für den neuen Ausleiher verbucht.
	HandleUnifiedCheckout(ctx context.Context, copy *repository.BookCopy, activeLeserID *string, staffID string, overrideBlock bool) (*LoanResult, error)

	// HandleSimpleReturn wickelt die direkte Rückgabe eines Buchexemplars ab (ohne dass ein neuer Ausleiher aktiv ist).
	// Ein freies Exemplar ist hier ein Fehler: Ausgeliehen wird über den Ausweis, nicht
	// dadurch, dass ein angemeldetes Konto ein Buch in die Hand nimmt.
	HandleSimpleReturn(ctx context.Context, copy *repository.BookCopy, staffID string) (*LoanResult, error)
}

// defaultLoanService implementiert den LoanService unter Verwendung von Repositories.
type defaultLoanService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	bookRepo    repository.BookRepository
	loanRepo    repository.LoanRepository
	auditRepo   repository.AuditRepository
	// userRepo löst Lehrerausweise auf, die offline gescannt wurden (nur die Nachbuch-Tür
	// setzt es; der Online-Scan kennt die Lehrkraft schon).
	userRepo repository.UserRepository
	// jetzt ist die Uhr der Fristberechnung; nil heißt time.Now. Tests setzen einen festen
	// Tag, um die Frist am Tag vor, am und nach dem Rückgabetermin zu prüfen (Bugklasse
	// „Frist am Tag des Ereignisses", docs/sweeps.md).
	jetzt func() time.Time
}

// heute liefert den Zeitpunkt der Uhr in der Schulzeitzone.
func (s *defaultLoanService) heute() time.Time {
	if s.jetzt != nil {
		return s.jetzt().In(schoolLocation())
	}
	return time.Now().In(schoolLocation())
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
