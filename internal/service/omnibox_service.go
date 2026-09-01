package service

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	// Abholbereit: Vormerkungen des GESCANNTEN Schülers, deren Buch im Abholfach
	// liegt (Betreiber-Entscheidung 01.09.2026). Schüler scannen nicht selbst —
	// der Hinweis sagt der Mitarbeiterin am Terminal, dass sie ins Abholfach
	// greifen soll, solange der Schüler vor ihr steht. Ohne ihn läge das Buch im
	// Fach, bis die 3-Tage-Frist es still an den Nächsten weiterreicht.
	Abholbereit []AbholbereiteVormerkung
}

// AbholbereiteVormerkung ist ein Eintrag des Abholfach-Hinweises: Titel und
// Abholfrist — bewusst ohne IDs, die Theke braucht nur den Griff ins Fach.
type AbholbereiteVormerkung struct {
	Titel             string
	BereitgestelltBis *time.Time
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
	userRepo    repository.UserRepository
	loanRepo    repository.LoanRepository
	loanSvc     LoanService
	deviceSvc   DeviceService
}

// NewOmniboxService erzeugt eine neue Instanz des standardmäßigen OmniboxService.
func NewOmniboxService(
	pool db.PgxPoolIface,
	studentRepo repository.StudentRepository,
	bookRepo repository.BookRepository,
	userRepo repository.UserRepository,
	loanRepo repository.LoanRepository,
	loanSvc LoanService,
	deviceSvc DeviceService,
) OmniboxService {
	return &defaultOmniboxService{
		pool:        pool,
		studentRepo: studentRepo,
		bookRepo:    bookRepo,
		userRepo:    userRepo,
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
// Auflösungsreihenfolge: Buch → Schülerausweis → Lehrerausweis → Volltextsuche.
//
// Die Präfixe S-/L-/B-/G- sind eine Abkürzung, keine Voraussetzung: Littera kennt sie
// nicht, und die Ausweise aus dem Altbestand tragen nackte Nummern. Ein Schülerausweis
// liefert gemessen `B97601826457` (Nummer des Kartenherstellers), ein Buchetikett eine
// 13-stellige EAN-13 — die Formen sind verschieden genug, dass die Reihenfolge hier
// eindeutig entscheidet.
//
// Die Lehrer-Stufe fehlte lange, und das war kein bewusster Ausschluss: Lehrkräfte
// stehen bei uns in `benutzer`, Schüler in `schueler`. Ein gescannter Lehrerausweis lief
// deshalb bis in die Volltextsuche und meldete „keine Treffer" — obwohl handleTeacherAction
// die passende Abfrage längst hatte, nur eben allein hinter dem L--Präfix. In Littera
// gibt es diesen Unterschied nicht; die Karte ist dieselbe, nur der Aufdruck lautet
// „Lehrerausweis".
//
// Die Lookups liefern bei Nichttreffer (nil, nil); ein non-nil Fehler ist daher ein
// echter DB-Fehler und wird propagiert (→ HTTP 500), statt ihn als "nicht gefunden" zu
// verschlucken.
func (s *defaultOmniboxService) resolveOhnePraefix(ctx context.Context, q OmniboxQuery, resp *OmniboxResult) error {
	copy, lookupErr := s.bookRepo.GetCopyByBarcode(ctx, q.Query)
	if lookupErr != nil {
		return fmt.Errorf("datenbankfehler bei Barcode-Auflösung: %w", lookupErr)
	}
	if copy != nil {
		return s.handleBookAction(ctx, q, resp)
	}

	// Littera-Buchetikett: Der Strichcode liefert eine EAN-13, im System steht die
	// kurze Mediennummer (siehe dekodiereLitteraEtikett). Erst rückrechnen, dann
	// erneut nachschlagen — nur ein existierendes Exemplar löst eine Aktion aus,
	// alles andere fällt unverändert zur Ausweis-/Volltextstufe durch.
	if nummer, istEtikett := dekodiereLitteraEtikett(q.Query); istEtikett {
		dekodiert, dekodiertErr := s.bookRepo.GetCopyByBarcode(ctx, nummer)
		if dekodiertErr != nil {
			return fmt.Errorf("datenbankfehler bei Etikett-Auflösung: %w", dekodiertErr)
		}
		if dekodiert != nil {
			q.Query = nummer
			return s.handleBookAction(ctx, q, resp)
		}
	}

	student, studentErr := s.studentRepo.GetByBarcode(ctx, q.Query)
	if studentErr != nil {
		return fmt.Errorf("datenbankfehler bei Ausweis-Auflösung: %w", studentErr)
	}
	if student != nil {
		return s.handleStudentAction(ctx, q.Query, resp)
	}

	// handleTeacherAction meldet ErrNotFound, wenn kein Lehrerausweis passt — das ist
	// hier kein Fehler, sondern der Übergang zur Volltextsuche.
	teacherErr := s.handleTeacherAction(ctx, q.Query, resp)
	if teacherErr == nil {
		return nil
	}
	if !errors.Is(teacherErr, ErrNotFound) {
		return teacherErr
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
	resp.Abholbereit = s.ladeAbholbereiteVormerkungen(ctx, student.ID)
	return nil
}

// ladeAbholbereiteVormerkungen holt die abholbereiten Vormerkungen des Schülers
// für den Abholfach-Hinweis. Ein Fehler hier bricht den Scan NICHT ab — die
// Theke wäre sonst wegen eines Hinweises arbeitsunfähig — sondern wird geloggt
// (dieselbe Abwägung wie resolveFotoURL im Profil).
func (s *defaultOmniboxService) ladeAbholbereiteVormerkungen(ctx context.Context, schuelerID string) []AbholbereiteVormerkung {
	// Die Unit-Tests der Scan-Weiche bauen den Service ohne Pool (Stub-Repos);
	// dort gibt es keine Vormerkungen und nichts zu laden.
	if s.pool == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT t.titel, v.bereitgestellt_bis
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		WHERE v.schueler_id = $1 AND v.status = 'abholbereit'
		ORDER BY v.bereitgestellt_bis ASC NULLS LAST
		LIMIT 5`, schuelerID)
	if err != nil {
		log.Printf("omnibox: Abholfach-Hinweis nicht ladbar für Schüler %s: %v", schuelerID, err)
		return nil
	}
	defer rows.Close()

	var out []AbholbereiteVormerkung
	for rows.Next() {
		var v AbholbereiteVormerkung
		if err := rows.Scan(&v.Titel, &v.BereitgestelltBis); err != nil {
			log.Printf("omnibox: Abholfach-Hinweis unlesbar: %v", err)
			return nil
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		log.Printf("omnibox: Abholfach-Hinweis unvollständig: %v", err)
		return nil
	}
	return out
}

// handleTeacherAction lädt die Lehrerdaten bei Scan eines Lehrer-Barcodes.
// handleTeacherAction lädt eine Lehrkraft über ihren Ausweis.
//
// Die Abfrage lag früher als rohes SQL direkt an diesem Service. Sie ist ins
// UserRepository gewandert, als die Lehrer-Stufe in die präfixlose Auflösung kam: Solange
// sie nur hinter dem L--Präfix hing, fiel nicht auf, dass sie sich — anders als Buch und
// Schüler — nicht durch ein Stub-Repository ersetzen ließ und damit ungetestet blieb.
func (s *defaultOmniboxService) handleTeacherAction(ctx context.Context, query string, resp *OmniboxResult) error {
	teacher, err := s.userRepo.GetLehrerByBarcode(ctx, query)
	if err != nil {
		return fmt.Errorf("datenbankfehler beim Laden des Lehrers: %w", err)
	}
	if teacher == nil {
		return fmt.Errorf("%w: Lehrer-Barcode %s nicht gefunden", ErrNotFound, query)
	}
	resp.Type = "teacher"
	resp.Teacher = teacher
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
		tag, err := s.pool.Exec(ctx, "UPDATE buecher_exemplare SET ist_ausleihbar = true, ist_ausgesondert = false, aussonderung_grund = NULL, zustand_notiz = '', bestellstatus = NULL WHERE id = $1", copy.ID)
		if err != nil {
			return false, err
		}
		// 0 Zeilen = Exemplar zwischen Lookup und Update entfernt (Race): Ohne diese
		// Prüfung liefe das In-Memory-Objekt („reaktiviert") der DB davon und die
		// Meldung „Buch reaktiviert" wäre gelogen (Phantom-Erfolg-Sweep 31.08.2026).
		if tag.RowsAffected() == 0 {
			return false, repository.ErrExemplarNichtGefunden
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
