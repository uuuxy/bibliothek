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

// ladeGeraet lädt das Gerät anhand der Barcode-ID und prüft die Sicherheitsschranken
// (nicht gefunden, ausgesondert, gesperrt).
func (s *defaultDeviceService) ladeGeraet(ctx context.Context, query string) (repository.Geraet, error) {
	var g repository.Geraet
	err := s.pool.QueryRow(ctx, `
		SELECT id, modellname, seriennummer, barcode_id, zubehoer, ist_ausleihbar, ist_ausgesondert, zustand_notiz
		FROM geraete
		WHERE barcode_id = $1
	`, query).Scan(&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer, &g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return g, fmt.Errorf("%w: Gerät mit Barcode %s nicht gefunden", ErrNotFound, query)
		}
		return g, err
	}

	// Ausgesonderte Geräte dürfen unter keinen Umständen ausgeliehen werden.
	if g.IstAusgesondert {
		return g, fmt.Errorf("%w: Gerät ist ausgesondert", ErrInvalidState)
	}
	// Gesperrte Geräte dürfen nicht ausgeliehen werden (z. B. bei Defekt).
	if !g.IstAusleihbar {
		return g, fmt.Errorf("%w: Gerät ist aktuell gesperrt", ErrBlocked)
	}
	return g, nil
}

// ladeAkteur ermittelt den aktiven Schüler bzw. Lehrer aus den gescannten Ausweisen.
// Gesperrte Schüler blockieren die Ausleihe.
func (s *defaultDeviceService) ladeAkteur(ctx context.Context, activeStudentID, activeTeacherID *string) (*repository.Student, *repository.User, error) {
	if activeStudentID != nil && *activeStudentID != "" {
		student, err := s.studentRepo.GetByID(ctx, *activeStudentID)
		if err != nil {
			return nil, nil, err
		}
		if student != nil && student.IstGesperrt {
			return nil, nil, fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
		}
		return student, nil, nil
	}
	if activeTeacherID != nil && *activeTeacherID != "" {
		return nil, &repository.User{ID: *activeTeacherID}, nil
	}
	return nil, nil, nil
}

// ladeAktiveAusleihe sperrt und lädt die offene Ausleihe des Geräts (Row-Level-Lock via
// FOR UPDATE). hasActiveLoan ist false, wenn keine offene Ausleihe existiert.
func ladeAktiveAusleihe(ctx context.Context, tx pgx.Tx, geraetID string) (repository.Loan, bool, error) {
	var activeLoan repository.Loan
	err := tx.QueryRow(ctx, `
		SELECT id, geraet_id, schueler_id, ausleiher_benutzer_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am, bearbeiter_id, ist_fremdrueckgabe, ist_handapparat
		FROM ausleihen
		WHERE geraet_id = $1 AND rueckgabe_am IS NULL
		FOR UPDATE
	`, geraetID).Scan(
		&activeLoan.ID, &activeLoan.GeraetID, &activeLoan.SchuelerID, &activeLoan.AusleiherBenutzerID,
		&activeLoan.AusgeliehenAm, &activeLoan.RueckgabeFrist, &activeLoan.RueckgabeAm,
		&activeLoan.BearbeiterID, &activeLoan.IstFremdrueckgabe, &activeLoan.IstHandapparat,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return activeLoan, false, nil
		}
		return activeLoan, false, err
	}
	return activeLoan, true, nil
}

// leiheGeraetAus behandelt Fall A: das freie Gerät wird an Schüler oder Lehrer ausgeliehen.
// geraeteLeihfristTage ist die Standard-Leihfrist für Hardware (2 Wochen).
const geraeteLeihfristTage = 14

// geraeteRueckgabeFrist normalisiert die Geräte-Leihfrist auf das Tagesende (23:59:59) in der
// Schul-Zeitzone (Europe/Berlin) — exakt wie die Buch-Fristen (loan_rules.go). Ohne diese
// Normalisierung fiel die Frist auf die Sekunde genau N Tage später in der Server-Zeitzone
// (im Docker-Container UTC); ein um 10:00 MESZ geliehenes Gerät wäre 08:00 UTC fällig, was
// Mahnläufe und die "heute/morgen fällig"-Anzeige verschob.
func geraeteRueckgabeFrist(now time.Time) time.Time {
	d := now.In(schoolLocation()).AddDate(0, 0, geraeteLeihfristTage)
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, schoolLocation())
}

func (s *defaultDeviceService) leiheGeraetAus(ctx context.Context, tx pgx.Tx, g *repository.Geraet, student *repository.Student, teacher *repository.User, staffID string) (*DeviceResult, error) {
	if student == nil && teacher == nil {
		return nil, fmt.Errorf("%w: Bitte scannen Sie zuerst einen Schüler- oder Lehrerausweis", ErrInvalidState)
	}

	// Standard-Hardware-Leihfrist beträgt 14 Tage (2 Wochen), auf das Tagesende in der
	// Schul-Zeitzone normalisiert (analog zu den Buch-Fristen).
	dueTime := geraeteRueckgabeFrist(time.Now())
	resp := &DeviceResult{}
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
		// Lehrer leihen Geräte standardmäßig als Handapparat (Dauerleihe) aus.
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

	// Revisionssicheres Audit-Log schreiben.
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

// ermittleVorbesitzer stellt fest, ob eine Fremdrückgabe vorliegt (Rückgeber ≠ Ausleiher)
// und lädt in diesem Fall die Stammdaten des ursprünglichen Ausleihers in resp.
func (s *defaultDeviceService) ermittleVorbesitzer(ctx context.Context, activeLoan *repository.Loan, student *repository.Student, teacher *repository.User, resp *DeviceResult) (bool, error) {
	if activeLoan.SchuelerID != nil {
		if student != nil && *activeLoan.SchuelerID == student.ID {
			return false, nil
		}
		var vorbesitzer repository.Student
		err := s.pool.QueryRow(ctx, "SELECT vorname, nachname, klasse FROM schueler WHERE id = $1 AND deleted_at IS NULL", *activeLoan.SchuelerID).Scan(&vorbesitzer.Vorname, &vorbesitzer.Nachname, &vorbesitzer.Klasse)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
		resp.Vorbesitzer = &vorbesitzer
		return true, nil
	}
	if activeLoan.AusleiherBenutzerID != nil {
		if teacher != nil && *activeLoan.AusleiherBenutzerID == teacher.ID {
			return false, nil
		}
		var vorbesitzerUser repository.User
		err := s.pool.QueryRow(ctx, "SELECT vorname, nachname FROM benutzer WHERE id = $1", *activeLoan.AusleiherBenutzerID).Scan(&vorbesitzerUser.Vorname, &vorbesitzerUser.Nachname)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
		resp.VorbesitzerUser = &vorbesitzerUser
		return true, nil
	}
	return false, nil
}

// gibGeraetZurueck behandelt Fall B: das ausgeliehene Gerät wird zurückgegeben.
func (s *defaultDeviceService) gibGeraetZurueck(ctx context.Context, tx pgx.Tx, g *repository.Geraet, activeLoan *repository.Loan, student *repository.Student, teacher *repository.User, staffID string) (*DeviceResult, error) {
	resp := &DeviceResult{}

	isFremd, err := s.ermittleVorbesitzer(ctx, activeLoan, student, teacher, resp)
	if err != nil {
		return nil, err
	}

	// Rückgabe in der Datenbank eintragen (rueckgabe_am und bearbeiter_id setzen).
	if _, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3
	`, staffID, isFremd, activeLoan.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Revisionssicheres Audit-Log für die Rückgabe schreiben.
	if activeLoan.SchuelerID != nil {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, g.ID, *activeLoan.SchuelerID, "", staffID))
	} else if activeLoan.AusleiherBenutzerID != nil {
		logAuditErr("rückgabe", s.auditRepo.LogRueckgabe(ctx, g.ID, "", *activeLoan.AusleiherBenutzerID, staffID))
	}

	resp.Type = "rueckgabe"
	resp.Geraet = g
	resp.LoanID = &activeLoan.ID
	resp.Fremdrueckgabe = isFremd

	// Event für Plugins und andere Subsysteme auslösen.
	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       g.ID,
		BarcodeID:    g.BarcodeID,
		Titel:        g.Modellname,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: staffID,
	})

	return resp, nil
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
	g, err := s.ladeGeraet(ctx, query)
	if err != nil {
		return nil, err
	}

	student, teacher, err := s.ladeAkteur(ctx, activeStudentID, activeTeacherID)
	if err != nil {
		return nil, err
	}

	// Transaktion starten, um den Ausleihprozess atomar und thread-sicher zu gestalten.
	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	activeLoan, hasActiveLoan, err := ladeAktiveAusleihe(ctx, tx, g.ID)
	if err != nil {
		return nil, err
	}

	// Checklisten-Regel: Hat das Gerät Zubehör und der Benutzer hat es noch nicht
	// bestätigt, unterbrechen wir und fordern die Bestätigung an.
	if g.Zubehoer != "" && !confirmedChecklist {
		return &DeviceResult{Type: "geraet_check", Geraet: &g}, nil
	}

	if !hasActiveLoan {
		return s.leiheGeraetAus(ctx, tx, &g, student, teacher, staffID)
	}
	return s.gibGeraetZurueck(ctx, tx, &g, &activeLoan, student, teacher, staffID)
}
