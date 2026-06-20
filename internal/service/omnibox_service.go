package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// OmniboxResult beschreibt die Antwortstruktur der Omnibox nach Verarbeitung einer Eingabe (Scan oder Suche).
type OmniboxResult struct {
	// Type definiert die Art der Antwort (z. B. "student", "teacher", "ausleihe", "rueckgabe", "search_results", "info").
	Type string
	// Message enthält eine optionale Benachrichtigung für das Frontend.
	Message string
	// Student enthält Schülerdaten, wenn ein Schülerausweis gescannt oder eine Aktion durchgeführt wurde.
	Student *repository.Student
	// Teacher enthält Lehrerdaten, wenn ein Lehrerausweis gescannt oder eine Aktion durchgeführt wurde.
	Teacher *repository.User
	// Book enthält die Daten des betroffenen Buchs (Ausleihe/Rückgabe).
	Book *repository.BookCopy
	// Geraet enthält die Daten des betroffenen Geräts (Hardware-Ausleihe/Rückgabe).
	Geraet *repository.Geraet
	// DueDate gibt das Rückgabedatum der aktuellen Ausleihe an.
	DueDate *time.Time
	// LoanID ist die ID des verknüpften Ausleihvorgangs.
	LoanID *string
	// Fremdrueckgabe zeigt an, ob die Rückgabe durch eine andere Person erfolgt ist.
	Fremdrueckgabe bool
	// Vorbesitzer enthält den Schüler, der das Buch/Gerät zuvor ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Student
	// VorbesitzerUser enthält den Lehrer, der das Buch/Gerät zuvor ausgeliehen hatte (bei Fremdrückgabe).
	VorbesitzerUser *repository.User
	// SearchResults enthält Suchergebnisse bei einer allgemeinen Buchtitel-Suche.
	SearchResults []repository.BookTitle
	// HasVormerkung zeigt an, ob für das zurückgegebene Buch eine Reservierung aktiv wurde.
	HasVormerkung bool
	// VormerkungTitel ist der Titel des reservierten Buchs.
	VormerkungTitel string
	// VormerkungUser ist der Name des Schülers, der die Reservierung ausgelöst hat.
	VormerkungUser string
}

// OmniboxService verarbeitet alle Eingaben aus der zentralen Suche/Scan-Leiste (Omnibox).
type OmniboxService interface {
	// ProcessQuery wertet eine Eingabe (Eingabestring) aus und steuert die passende Domänen-Aktion an.
	ProcessQuery(ctx context.Context, query string, activeStudentID *string, activeTeacherID *string, confirmedChecklist bool, staffID string, staffRole string) (*OmniboxResult, error)
}

// defaultOmniboxService ist die Standard-Implementierung des OmniboxService.
type defaultOmniboxService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	bookRepo    repository.BookRepository
	loanRepo    repository.LoanRepository
	loanSvc     LoanService
	deviceSvc   DeviceService
}

// NewOmniboxService erzeugt eine neue Instanz des standardmäßigen OmniboxService.
func NewOmniboxService(
	pool db.PgxPoolIface,
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	loanRepo repository.LoanRepository,
	loanSvc LoanService,
	deviceSvc DeviceService,
) OmniboxService {
	return &defaultOmniboxService{
		pool:        pool,
		studentRepo: studentRepo,
		bookRepo:    bookRepo,
		loanRepo:    loanRepo,
		loanSvc:     loanSvc,
		deviceSvc:   deviceSvc,
	}
}

// ProcessQuery leitet gescannte Barcodes oder Suchanfragen anhand von Präfixen an die jeweilige Fachlogik weiter.
func (s *defaultOmniboxService) ProcessQuery(
	ctx context.Context,
	query string,
	activeStudentID *string,
	activeTeacherID *string,
	confirmedChecklist bool,
	staffID string,
	staffRole string,
) (*OmniboxResult, error) {
	resp := &OmniboxResult{}
	var err error

	// Präfix-Erkennung (Scanner-Steuerung):
	// S- steht für Schüler (Student)
	// L- steht für Lehrer (Teacher)
	// B- steht für Buch (Book)
	// G- steht für Gerät (Hardware-Geräte)
	if strings.HasPrefix(query, "S-") {
		err = s.handleStudentAction(ctx, query, resp)
	} else if strings.HasPrefix(query, "L-") {
		err = s.handleTeacherAction(ctx, query, resp)
	} else if strings.HasPrefix(query, "B-") {
		err = s.handleBookAction(ctx, query, activeStudentID, activeTeacherID, staffID, staffRole, resp)
	} else if strings.HasPrefix(query, "G-") {
		dr, err := s.deviceSvc.HandleDeviceAction(ctx, query, activeStudentID, activeTeacherID, confirmedChecklist, staffID)
		if err == nil {
			s.mapDeviceResult(dr, resp)
		}
		return resp, err
	} else {
		// Fallback: Wenn kein Präfix vorhanden ist, prüfen wir zuerst, ob die Eingabe ein registrierter Buch-Barcode ist.
		// Ist dies der Fall, verarbeiten wir es als Buch-Aktion. Andernfalls führen wir eine Volltext-Titelsuche aus.
		if copy, _ := s.bookRepo.GetCopyByBarcode(ctx, query); copy != nil {
			err = s.handleBookAction(ctx, query, activeStudentID, activeTeacherID, staffID, staffRole, resp)
		} else {
			err = s.handleSearchAction(ctx, query, resp)
		}
	}

	return resp, err
}

// mapDeviceResult mappt die Felder aus DeviceResult in die flache OmniboxResult-Struktur.
func (s *defaultOmniboxService) mapDeviceResult(dr *DeviceResult, resp *OmniboxResult) {
	if dr == nil {
		return
	}
	resp.Type = dr.Type
	resp.Geraet = dr.Geraet
	resp.Student = dr.Student
	resp.Teacher = dr.Teacher
	resp.DueDate = dr.DueDate
	resp.LoanID = dr.LoanID
	resp.Fremdrueckgabe = dr.Fremdrueckgabe
	resp.Vorbesitzer = dr.Vorbesitzer
	resp.VorbesitzerUser = dr.VorbesitzerUser
}

// handleStudentAction lädt die Schülerdaten bei Scan eines Schüler-Barcodes.
func (s *defaultOmniboxService) handleStudentAction(ctx context.Context, query string, resp *OmniboxResult) error {
	student, err := s.studentRepo.GetByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if student == nil {
		return fmt.Errorf("%w: Schüler-Barcode %s ist nicht registriert", ErrNotFound, query)
	}
	resp.Type = "student"
	resp.Student = student
	return nil
}

// handleTeacherAction lädt die Lehrerdaten bei Scan eines Lehrer-Barcodes.
func (s *defaultOmniboxService) handleTeacherAction(ctx context.Context, query string, resp *OmniboxResult) error {
	var teacher repository.User
	err := s.pool.QueryRow(ctx, "SELECT id, barcode_id, vorname, nachname, rolle FROM benutzer WHERE barcode_id = $1 AND rolle = 'LEHRER' AND aktiv = true LIMIT 1", query).
		Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
	if err != nil {
		return fmt.Errorf("%w: Lehrer-Barcode %s nicht gefunden", ErrNotFound, query)
	}
	resp.Type = "teacher"
	resp.Teacher = &teacher
	return nil
}

// handleSearchAction führt eine Volltextsuche über Buchtitel, Autoren, ISBN und Systematik aus.
func (s *defaultOmniboxService) handleSearchAction(ctx context.Context, query string, resp *OmniboxResult) error {
	titles, err := s.bookRepo.SearchTitles(ctx, query)
	if err != nil {
		return err
	}
	resp.Type = "search_results"
	resp.SearchResults = titles
	return nil
}

type vormerkung struct {
	ID         string
	TitelID    string
	SchuelerID string
	Notiz      string
	Status     string
	ErstelltAm time.Time
}

// checkVormerkung prüft, ob für einen Buchtitel eine aktive Reservierung vorliegt.
func (s *defaultOmniboxService) checkVormerkung(ctx context.Context, titelID string) (*vormerkung, error) {
	var v vormerkung
	err := s.pool.QueryRow(ctx, "SELECT id, titel_id, schueler_id, notiz, status, erstellt_am FROM vormerkungen WHERE titel_id = $1 AND status = 'wartend' ORDER BY erstellt_am ASC LIMIT 1", titelID).
		Scan(&v.ID, &v.TitelID, &v.SchuelerID, &v.Notiz, &v.Status, &v.ErstelltAm)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// handleBookAction verarbeitet das Scannen eines Buch-Barcodes.
// Wenn kein aktiver Ausleiher vorhanden ist, wird das Buch zurückgegeben.
// Ist ein Schüler oder Lehrer aktiv, wird das Buch an diesen ausgeliehen.
func (s *defaultOmniboxService) handleBookAction(
	ctx context.Context,
	query string,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
	staffRole string,
	resp *OmniboxResult,
) error {
	copy, err := s.bookRepo.GetCopyByBarcode(ctx, query)
	if err != nil {
		return err
	}
	if copy == nil {
		return fmt.Errorf("%w: Buchexemplar-Barcode %s wurde nicht gefunden", ErrNotFound, query)
	}

	// Falls das Buch gesperrt oder ausgesondert ist, prüfen wir auf automatische Reaktivierungsmöglichkeiten.
	if !copy.IstAusleihbar || copy.IstAusgesondert {
		activeLoan, err := s.loanRepo.GetActiveLoanByCopyID(ctx, copy.ID)
		if err != nil {
			return err
		}

		isReserved := strings.HasPrefix(copy.ZustandNotiz, "Reserviert für:")
		reservedForThisStudent := false

		// Falls das Exemplar reserviert ist, prüfen wir, ob der aktive Schüler der berechtigte Reservierer ist.
		if isReserved && activeStudentID != nil && *activeStudentID != "" {
			v, checkErr := s.checkVormerkung(ctx, copy.TitelID)
			if checkErr == nil && v != nil && v.SchuelerID == *activeStudentID {
				reservedForThisStudent = true
			}
		}

		// Automatisches Reaktivieren:
		// Wenn kein aktiver Ausleihvorgang vorliegt und das Buch entweder unreserviert ist oder der
		// berechtigte Schüler es ausleiht, heben wir die Ausleihsperre automatisch auf.
		if activeLoan == nil && (!isReserved || reservedForThisStudent) {
			_, err = s.pool.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausleihbar = true, ist_ausgesondert = false, zustand_notiz = '' WHERE id = $1", copy.ID)
			if err != nil {
				return err
			}
			copy.IstAusleihbar = true
			copy.ZustandNotiz = ""

			if !reservedForThisStudent {
				resp.Type = "info"
				resp.Message = "Buch reaktiviert"
				return nil
			}
		} else if isReserved && !reservedForThisStudent {
			return fmt.Errorf("%w: Dieses Buchexemplar ist %s", ErrBlocked, copy.ZustandNotiz)
		} else if copy.IstAusgesondert {
			return fmt.Errorf("%w: Buchexemplar %s ist ausgesondert und kann nicht ausgeliehen werden", ErrInvalidState, query)
		} else {
			return fmt.Errorf("%w: Buchexemplar ist nicht ausleihbar", ErrInvalidState)
		}
	}

	// Ausleihe durchführen, falls ein aktiver Ausleiher vorhanden ist
	if (activeTeacherID != nil && *activeTeacherID != "") || (activeStudentID != nil && *activeStudentID != "") {
		lr, err := s.loanSvc.HandleUnifiedCheckout(ctx, copy, activeStudentID, activeTeacherID, staffID)
		if err != nil {
			return err
		}
		s.mapLoanResult(lr, resp)
		return nil
	}

	// Rückgabe durchführen, wenn kein aktiver Ausleiher vorhanden ist
	lr, err := s.loanSvc.HandleSimpleReturn(ctx, copy, staffID, staffRole)
	if err != nil {
		return err
	}
	s.mapLoanResult(lr, resp)
	return nil
}

// mapLoanResult mappt die Felder aus LoanResult in die OmniboxResult-Struktur.
func (s *defaultOmniboxService) mapLoanResult(lr *LoanResult, resp *OmniboxResult) {
	if lr == nil {
		return
	}
	resp.Type = lr.Type
	resp.Book = lr.Book
	resp.Student = lr.Student
	resp.Teacher = lr.Teacher
	resp.DueDate = lr.DueDate
	resp.LoanID = lr.LoanID
	resp.Fremdrueckgabe = lr.Fremdrueckgabe
	resp.Vorbesitzer = lr.Vorbesitzer
	resp.VorbesitzerUser = lr.VorbesitzerUser
	resp.HasVormerkung = lr.HasVormerkung
	resp.VormerkungTitel = lr.VormerkungTitel
	resp.VormerkungUser = lr.VormerkungUser
}
