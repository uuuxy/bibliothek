package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/db"
	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// DeviceResult beschreibt das Ergebnis einer Geräte-Aktion wie Ausleihe, Rückgabe oder Checklisten-Anforderung.
type DeviceResult struct {
	// Type definiert den Typ des Ergebnisses (z. B. "ausleihe", "rueckgabe", "geraet_check").
	Type string
	// Geraet enthält die Stammdaten des betroffenen Geräts.
	Geraet *repository.Geraet
	// Student ist der Schüler, für den die Aktion durchgeführt wurde.
	Student *repository.Student
	// Teacher ist der Lehrer bzw. Benutzer, für den die Aktion durchgeführt wurde.
	Teacher *repository.User
	// DueDate gibt die berechnete Rückgabefrist für das Gerät an (nur bei Ausleihe).
	DueDate *time.Time
	// LoanID ist die ID des verknüpften Ausleihdatensatzes.
	LoanID *string
	// Fremdrueckgabe ist wahr, wenn das Gerät von einer anderen Person als dem Ausleiher zurückgegeben wurde.
	Fremdrueckgabe bool
	// Vorbesitzer speichert den Schüler, der das Gerät zuletzt ausgeliehen hatte (bei Fremdrückgabe).
	Vorbesitzer *repository.Student
	// VorbesitzerUser speichert den Lehrer, der das Gerät zuletzt ausgeliehen hatte (bei Fremdrückgabe).
	VorbesitzerUser *repository.User
}

// DeviceService definiert die Schnittstelle für alle Aktionen rund um Hardware-Geräte (z. B. Laptops, Tablets).
type DeviceService interface {
	// HandleDeviceAction verarbeitet das Scannen eines Geräts. Je nach Zustand wird das Gerät
	// entweder ausgeliehen (falls frei) oder zurückgegeben (falls aktuell ausgeliehen).
	// Zudem wird geprüft, ob eine Zubehör-Checkliste vor der Ausleihe bestätigt werden muss.
	HandleDeviceAction(ctx context.Context, query string, activeStudentID *string, activeTeacherID *string, confirmedChecklist bool, staffID string) (*DeviceResult, error)
}

// defaultDeviceService ist die Standard-Implementierung des DeviceService.
type defaultDeviceService struct {
	pool        db.PgxPoolIface
	studentRepo repository.StudentRepository
	loanRepo    repository.LoanRepository
	auditRepo   repository.AuditRepository
}

// NewDeviceService erstellt eine neue Instanz des Standard-Geräteservice.
func NewDeviceService(pool db.PgxPoolIface, studentRepo repository.StudentRepository, loanRepo repository.LoanRepository, auditRepo repository.AuditRepository) DeviceService {
	return &defaultDeviceService{
		pool:        pool,
		studentRepo: studentRepo,
		loanRepo:    loanRepo,
		auditRepo:   auditRepo,
	}
}

// HandleDeviceAction implementiert die Geschäftslogik für die Ausleihe und Rückgabe von Geräten.
func (s *defaultDeviceService) HandleDeviceAction(
	ctx context.Context,
	query string,
	activeStudentID *string,
	activeTeacherID *string,
	confirmedChecklist bool,
	staffID string,
) (*DeviceResult, error) {
	resp := &DeviceResult{}

	// Gerät anhand der Barcode-ID aus der Datenbank laden
	var g repository.Geraet
	err := s.pool.QueryRow(ctx, `
		SELECT id, modellname, seriennummer, barcode_id, zubehoer, ist_ausleihbar, ist_ausgesondert, zustand_notiz
		FROM geraete
		WHERE barcode_id = $1
	`, query).Scan(&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer, &g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: Gerät mit Barcode %s nicht gefunden", ErrNotFound, query)
		}
		return nil, err
	}

	// Sicherheitsschranke: Ausgesonderte Geräte dürfen unter keinen Umständen ausgeliehen werden
	if g.IstAusgesondert {
		return nil, fmt.Errorf("%w: Gerät ist ausgesondert", ErrInvalidState)
	}

	// Sicherheitsschranke: Gesperrte Geräte dürfen nicht ausgeliehen werden (z. B. bei Defekt)
	if !g.IstAusleihbar {
		return nil, fmt.Errorf("%w: Gerät ist aktuell gesperrt", ErrBlocked)
	}

	// Kontext-Prüfung: Schüler- oder Lehrer-Daten laden, falls ein Ausweis als aktiv markiert ist
	var student *repository.Student
	var teacher *repository.User

	if activeStudentID != nil && *activeStudentID != "" {
		student, err = s.studentRepo.GetByID(ctx, *activeStudentID)
		if err != nil {
			return nil, err
		}
		// Wenn der Schüler gesperrt ist (z. B. wegen ausstehender Mahnungen/Gebühren), Ausleihe blockieren
		if student != nil && student.IstGesperrt {
			return nil, fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
		}
	} else if activeTeacherID != nil && *activeTeacherID != "" {
		teacher = &repository.User{ID: *activeTeacherID}
	}

	// Transaktion starten, um den Ausleihprozess atomar und thread-sicher zu gestalten
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Aktive Ausleihe für dieses Gerät abfragen und sperren (Row-Level-Lock via FOR UPDATE).
	// Dies verhindert, dass zwei parallele Requests gleichzeitig dieselbe Ausleihe manipulieren.
	var activeLoan repository.Loan
	err = tx.QueryRow(ctx, `
		SELECT id, geraet_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE geraet_id = $1 AND rueckgabe_am IS NULL
		FOR UPDATE
	`, g.ID).Scan(
		&activeLoan.ID, &activeLoan.GeraetID, &activeLoan.SchuelerID, &activeLoan.AusleiherBenutzerID,
		&activeLoan.AusgeliehenAm, &activeLoan.RueckgabeFrist, &activeLoan.RueckgabeAm,
		&activeLoan.BearbeiterID, &activeLoan.IstFremdrueckgabe, &activeLoan.IstHandapparat,
	)

	hasActiveLoan := true
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			hasActiveLoan = false
		} else {
			return nil, err
		}
	}

	// Checklisten-Regel: Wenn das Gerät Zubehör hat (z. B. Ladekabel, Tasche) und der Benutzer
	// dies im Frontend noch nicht explizit bestätigt hat, unterbrechen wir und fordern die Bestätigung an.
	if g.Zubehoer != "" && !confirmedChecklist {
		resp.Type = "geraet_check"
		resp.Geraet = &g
		return resp, nil
	}

	// Fall A: Keine aktive Ausleihe vorhanden -> Das Gerät wird ausgeliehen
	if !hasActiveLoan {
		return s.handleDeviceCheckout(ctx, tx, &g, student, teacher, staffID, resp)
	}

	// Fall B: Gerät ist bereits ausgeliehen -> Das Gerät wird zurückgegeben
	return s.handleDeviceReturn(ctx, tx, &g, &activeLoan, student, teacher, staffID, resp)
}

func (s *defaultDeviceService) handleDeviceCheckout(
	ctx context.Context,
	tx pgx.Tx,
	g *repository.Geraet,
	student *repository.Student,
	teacher *repository.User,
	staffID string,
	resp *DeviceResult,
) (*DeviceResult, error) {
	if student == nil && teacher == nil {
		return nil, fmt.Errorf("%w: Bitte scannen Sie zuerst einen Schüler- oder Lehrerausweis", ErrInvalidState)
	}

	// Standard-Hardware-Leihfrist beträgt 14 Tage (2 Wochen)
	dueTime := time.Now().AddDate(0, 0, 14)
	var newLoanID string
	var err error
	if student != nil {
		err = tx.QueryRow(ctx, `
			INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist, bearbeiter_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, g.ID, student.ID, dueTime, staffID).Scan(&newLoanID)
		resp.Student = student
	} else {
		// Lehrer leihen Geräte standardmäßig als Handapparat (Dauerleihe) aus
		err = tx.QueryRow(ctx, `
			INSERT INTO ausleihen (geraet_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
			VALUES ($1, $2, $3, $4, true)
			RETURNING id
		`, g.ID, teacher.ID, dueTime, staffID).Scan(&newLoanID)
		resp.Teacher = teacher
	}

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log schreiben
	if student != nil {
		logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, g.ID, student.ID, "", staffID))
	} else {
		logAuditErr("ausleihe", s.auditRepo.LogAusleihe(ctx, g.ID, "", teacher.ID, staffID))
	}

	resp.Type = "ausleihe"
	resp.Geraet = g
	resp.DueDate = &dueTime
	resp.LoanID = &newLoanID

	return resp, nil
}

func (s *defaultDeviceService) handleDeviceReturn(
	ctx context.Context,
	tx pgx.Tx,
	g *repository.Geraet,
	activeLoan *repository.Loan,
	student *repository.Student,
	teacher *repository.User,
	staffID string,
	resp *DeviceResult,
) (*DeviceResult, error) {
	// Prüfen, ob es sich um eine Fremdrückgabe handelt.
	// Das ist der Fall, wenn die Person, die das Gerät zurückgibt (aktiver Schüler/Lehrer),
	// nicht mit der Person übereinstimmt, die das Gerät ausgeliehen hat.
	isFremd := false
	if activeLoan.SchuelerID != nil {
		if student == nil || *activeLoan.SchuelerID != student.ID {
			isFremd = true
			var vorbesitzer repository.Student
			err := s.pool.QueryRow(ctx, "SELECT vorname, nachname, klasse FROM schueler WHERE id = $1 AND deleted_at IS NULL", *activeLoan.SchuelerID).Scan(&vorbesitzer.Vorname, &vorbesitzer.Nachname, &vorbesitzer.Klasse)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			resp.Vorbesitzer = &vorbesitzer
		}
	} else if activeLoan.AusleiherBenutzerID != nil {
		if teacher == nil || *activeLoan.AusleiherBenutzerID != teacher.ID {
			isFremd = true
			var vorbesitzerUser repository.User
			err := s.pool.QueryRow(ctx, "SELECT vorname, nachname FROM benutzer WHERE id = $1", *activeLoan.AusleiherBenutzerID).Scan(&vorbesitzerUser.Vorname, &vorbesitzerUser.Nachname)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			resp.VorbesitzerUser = &vorbesitzerUser
		}
	}

	// Rückgabe in der Datenbank eintragen (rueckgabe_am und bearbeiter_id setzen)
	_, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3
	`, staffID, isFremd, activeLoan.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log für die Rückgabe schreiben
	if activeLoan.SchuelerID != nil {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, g.ID, *activeLoan.SchuelerID, "", staffID))
	} else if activeLoan.AusleiherBenutzerID != nil {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, g.ID, "", *activeLoan.AusleiherBenutzerID, staffID))
	}

	resp.Type = "rueckgabe"
	resp.Geraet = g
	resp.LoanID = &activeLoan.ID
	resp.Fremdrueckgabe = isFremd

	// Event für Plugins und andere Subsysteme auslösen
	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       g.ID,
		BarcodeID:    g.BarcodeID,
		Titel:        g.Modellname,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: staffID,
	})

	return resp, nil
}
