package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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
	// RegalfreigabeBarcode: reserviertes Exemplar, das zurück ins Regal muss (der
	// Schüler hat ein anderes Exemplar desselben Titels genommen).
	RegalfreigabeBarcode string
}

// OmniboxQuery bündelt die Eingabe und den Sitzungskontext eines Omnibox-Requests,
// damit ProcessQuery & Co. nicht acht Einzelargumente durchreichen.
type OmniboxQuery struct {
	Query              string
	ActiveStudentID    *string
	ActiveTeacherID    *string
	ConfirmedChecklist bool
	StaffID            string
	StaffRole          string
	OverrideBlock      bool
}

// OmniboxService verarbeitet alle Eingaben aus der zentralen Suche/Scan-Leiste (Omnibox).
type OmniboxService interface {
	// ProcessQuery wertet eine Eingabe (Eingabestring) aus und steuert die passende Domänen-Aktion an.
	ProcessQuery(ctx context.Context, q OmniboxQuery) (*OmniboxResult, error)
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
func (s *defaultOmniboxService) ProcessQuery(ctx context.Context, q OmniboxQuery) (*OmniboxResult, error) {
	resp := &OmniboxResult{}

	// Präfix-Erkennung (Scanner-Steuerung):
	// S- steht für Schüler (Student)
	// L- steht für Lehrer (Teacher)
	// B- steht für Buch (Book)
	// G- steht für Gerät (Hardware-Geräte)
	switch {
	case strings.HasPrefix(q.Query, "S-"):
		return resp, s.handleStudentAction(ctx, q.Query, resp)
	case strings.HasPrefix(q.Query, "L-"):
		return resp, s.handleTeacherAction(ctx, q.Query, resp)
	case strings.HasPrefix(q.Query, "B-"):
		return resp, s.handleBookAction(ctx, q, resp)
	case strings.HasPrefix(q.Query, "G-"):
		dr, err := s.deviceSvc.HandleDeviceAction(ctx, q.Query, q.ActiveStudentID, q.ActiveTeacherID, q.ConfirmedChecklist, q.StaffID)
		if err == nil {
			s.mapDeviceResult(dr, resp)
		}
		return resp, err
	default:
		return resp, s.resolveOhnePraefix(ctx, q, resp)
	}
}

// resolveOhnePraefix löst einen Barcode/eine Query ohne bekanntes Präfix auf.
// Auflösungsreihenfolge: Buch → Schülerausweis → Volltextsuche.
// Die Littera-Altbestand-Ausweise tragen nackte Nummern ohne "S-"-Präfix und dürfen
// nicht neu etikettiert werden; ihre Nummernkreise überschneiden sich nicht mit den
// (kürzeren) Littera-Mediennummern, daher ist die Reihenfolge deterministisch.
// GetCopyByBarcode/GetByBarcode liefern bei Nichttreffer (nil, nil); ein non-nil Fehler
// ist daher ein echter DB-Fehler und wird propagiert (→ HTTP 500), statt ihn als
// "nicht gefunden" zu verschlucken.
func (s *defaultOmniboxService) resolveOhnePraefix(ctx context.Context, q OmniboxQuery, resp *OmniboxResult) error {
	copy, lookupErr := s.bookRepo.GetCopyByBarcode(ctx, q.Query)
	if lookupErr != nil {
		return fmt.Errorf("datenbankfehler bei Barcode-Auflösung: %w", lookupErr)
	}
	if copy != nil {
		return s.handleBookAction(ctx, q, resp)
	}

	student, studentErr := s.studentRepo.GetByBarcode(ctx, q.Query)
	if studentErr != nil {
		return fmt.Errorf("datenbankfehler bei Ausweis-Auflösung: %w", studentErr)
	}
	if student != nil {
		return s.handleStudentAction(ctx, q.Query, resp)
	}
	return s.handleSearchAction(ctx, q.Query, resp)
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
	// benutzer.rolle ist das ENUM benutzer_rolle ('admin','lehrer','mitarbeiter') — kleingeschrieben.
	// rolle::text vergleicht cast-sicher und vermeidet "invalid input value for enum" bei Großschreibung.
	err := s.pool.QueryRow(ctx, "SELECT id, barcode_id, vorname, nachname, rolle FROM benutzer WHERE barcode_id = $1 AND LOWER(rolle::text) = 'lehrer' AND aktiv = true LIMIT 1", query).
		Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: Lehrer-Barcode %s nicht gefunden", ErrNotFound, query)
		}
		// Propagate real DB errors (timeout, connection loss, etc.) as-is → becomes HTTP 500
		return fmt.Errorf("datenbankfehler beim Laden des Lehrers: %w", err)
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
// Gibt nil, nil zurück, wenn keine wartende Vormerkung existiert.
func (s *defaultOmniboxService) checkVormerkung(ctx context.Context, titelID string) (*vormerkung, error) {
	var v vormerkung
	// Nur Vormerkungen abholberechtigter Schüler zählen als aktive Reservierung — sonst
	// würde eine Vormerkung eines gelöschten/gesperrten Schülers das Exemplar für andere
	// blockieren (schuelerAbholberechtigt, siehe loan_return.go).
	err := s.pool.QueryRow(ctx, `
		SELECT v.id, v.titel_id, v.schueler_id, v.notiz, v.status, v.erstellt_am
		FROM vormerkungen v
		JOIN schueler s ON s.id = v.schueler_id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		  AND `+schuelerAbholberechtigt+`
		ORDER BY v.erstellt_am ASC LIMIT 1`, titelID).
		Scan(&v.ID, &v.TitelID, &v.SchuelerID, &v.Notiz, &v.Status, &v.ErstelltAm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Keine Vormerkung gefunden — kein Fehler
		}
		return nil, err
	}
	return &v, nil
}

// versucheReaktivierung behandelt gesperrte/ausgesonderte Exemplare: liegt kein
// aktiver Ausleihvorgang vor und ist das Buch unreserviert bzw. wird es vom
// berechtigten Reservierer geholt, wird die Sperre automatisch aufgehoben.
// fertig=true bedeutet, dass resp bereits final gesetzt wurde (nur reaktiviert);
// fertig=false ohne Fehler heißt: weiter zur Ausleihe.
func (s *defaultOmniboxService) versucheReaktivierung(ctx context.Context, query string, copy *repository.BookCopy, activeStudentID *string, resp *OmniboxResult) (fertig bool, err error) {
	activeLoan, err := s.loanRepo.GetActiveLoanByCopyID(ctx, copy.ID)
	if err != nil {
		return false, err
	}

	isReserved := strings.HasPrefix(copy.ZustandNotiz, "Reserviert für:")

	// Falls das Exemplar reserviert ist, prüfen wir, ob der aktive Schüler der berechtigte Reservierer ist.
	reservedForThisStudent := false
	if isReserved {
		reservedForThisStudent = s.istBerechtigterReservierer(ctx, copy.TitelID, activeStudentID)
	}

	// Automatisches Reaktivieren, wenn keine aktive Ausleihe vorliegt und das Buch
	// unreserviert ist oder der berechtigte Schüler es ausleiht.
	if activeLoan == nil && (!isReserved || reservedForThisStudent) {
		// Wieder aufgetaucht: zurück in den Umlauf — der Aussonderungs-Grund muss
		// mit zurückgesetzt werden (CHECK: im Umlauf = kein Grund).
		if _, err := s.pool.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausleihbar = true, ist_ausgesondert = false, aussonderung_grund = NULL, zustand_notiz = '' WHERE id = $1", copy.ID); err != nil {
			return false, err
		}
		copy.IstAusleihbar = true
		copy.ZustandNotiz = ""

		if !reservedForThisStudent {
			resp.Type = "info"
			resp.Message = "Buch reaktiviert"
			return true, nil
		}
		// Reaktiviert für den berechtigten Schüler -> Ausleihe folgt im Aufrufer.
		return false, nil
	}

	if isReserved && !reservedForThisStudent {
		return false, fmt.Errorf("%w: Dieses Buchexemplar ist %s", ErrBlocked, copy.ZustandNotiz)
	}
	if copy.IstAusgesondert {
		return false, fmt.Errorf("%w: Buchexemplar %s ist ausgesondert und kann nicht ausgeliehen werden", ErrInvalidState, query)
	}
	return false, fmt.Errorf("%w: Buchexemplar ist nicht ausleihbar", ErrInvalidState)
}

// istBerechtigterReservierer prüft, ob der aktive Schüler der berechtigte Reservierer
// des (reservierten) Buchtitels ist. Ohne aktiven Schüler ist das Ergebnis false.
func (s *defaultOmniboxService) istBerechtigterReservierer(ctx context.Context, titelID string, activeStudentID *string) bool {
	if activeStudentID == nil || *activeStudentID == "" {
		return false
	}
	v, checkErr := s.checkVormerkung(ctx, titelID)
	return checkErr == nil && v != nil && v.SchuelerID == *activeStudentID
}

// handleBookAction verarbeitet das Scannen eines Buch-Barcodes.
// Wenn kein aktiver Ausleiher vorhanden ist, wird das Buch zurückgegeben.
// Ist ein Schüler oder Lehrer aktiv, wird das Buch an diesen ausgeliehen.
func (s *defaultOmniboxService) handleBookAction(ctx context.Context, q OmniboxQuery, resp *OmniboxResult) error {
	copy, err := s.bookRepo.GetCopyByBarcode(ctx, q.Query)
	if err != nil {
		return err
	}
	if copy == nil {
		return fmt.Errorf("%w: Buchexemplar-Barcode %s wurde nicht gefunden", ErrNotFound, q.Query)
	}

	// Gesperrte/ausgesonderte Exemplare ggf. automatisch reaktivieren.
	if !copy.IstAusleihbar || copy.IstAusgesondert {
		fertig, err := s.versucheReaktivierung(ctx, q.Query, copy, q.ActiveStudentID, resp)
		if err != nil {
			return err
		}
		if fertig {
			return nil
		}
	}

	// Ausleihe durchführen, falls ein aktiver Ausleiher vorhanden ist
	if (q.ActiveTeacherID != nil && *q.ActiveTeacherID != "") || (q.ActiveStudentID != nil && *q.ActiveStudentID != "") {
		lr, err := s.loanSvc.HandleUnifiedCheckout(ctx, copy, q.ActiveStudentID, q.ActiveTeacherID, q.StaffID, q.OverrideBlock)
		if err != nil {
			return err
		}
		s.mapLoanResult(lr, resp)
		return nil
	}

	// Rückgabe durchführen, wenn kein aktiver Ausleiher vorhanden ist
	lr, err := s.loanSvc.HandleSimpleReturn(ctx, copy, q.StaffID, q.StaffRole)
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
	resp.RegalfreigabeBarcode = lr.RegalfreigabeBarcode
}
