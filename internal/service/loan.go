package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/plugins"
	"bibliothek/repository"
	"github.com/jackc/pgx/v5"
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

type SystemEinstellungen struct {
	FristBuchTage           int
	FristMedienTage         int
	MaxAusleihenSchueler    int
	LmfStichtag             string
	FerienLeseclubAktiv     bool
	FerienLeseclubZieldatum *string
}

func (s *defaultLoanService) querySettings(ctx context.Context) (*SystemEinstellungen, error) {
	rows, err := s.pool.Query(ctx, "SELECT schluessel, wert FROM system_einstellungen")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := &SystemEinstellungen{
		FristBuchTage:        21,
		FristMedienTage:      7,
		MaxAusleihenSchueler: 5,
		LmfStichtag:          "07-31",
		FerienLeseclubAktiv:  false,
	}

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case "frist_buch_tage":
			if v, err := strconv.Atoi(value); err == nil {
				settings.FristBuchTage = v
			}
		case "frist_medien_tage":
			if v, err := strconv.Atoi(value); err == nil {
				settings.FristMedienTage = v
			}
		case "max_ausleihen_schueler":
			if v, err := strconv.Atoi(value); err == nil {
				settings.MaxAusleihenSchueler = v
			}
		case "lmf_stichtag":
			settings.LmfStichtag = value
		case "ferien_leseclub_aktiv":
			settings.FerienLeseclubAktiv = (value == "true")
		case "ferien_leseclub_zieldatum":
			if value != "" {
				val := value
				settings.FerienLeseclubZieldatum = &val
			}
		}
	}
	return settings, nil
}

func calculateDueDate(titel, medientyp, lmfStichtag string, fristBuchTage, fristMedienTage int) time.Time {
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
		return now.AddDate(0, 0, fristMedienTage)
	}
	return now.AddDate(0, 0, fristBuchTage)
}

func (s *defaultLoanService) resolveCheckoutDueDate(ctx context.Context, copy *repository.BookCopy) (time.Time, error) {
	settings, err := s.querySettings(ctx)
	if err != nil {
		return calculateDueDate(copy.Titel, copy.Medientyp, "07-31", 21, 7), nil
	}
	isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")
	if !isLMF && settings.FerienLeseclubAktiv && settings.FerienLeseclubZieldatum != nil {
		t, parseErr := time.Parse("2006-01-02", *settings.FerienLeseclubZieldatum)
		if parseErr == nil {
			end := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
			return end, nil
		}
	}
	return calculateDueDate(copy.Titel, copy.Medientyp, settings.LmfStichtag, settings.FristBuchTage, settings.FristMedienTage), nil
}

func (s *defaultLoanService) processReturnVormerkungTx(ctx context.Context, tx pgx.Tx, copy *repository.BookCopy, resp *LoanResult) {
	var vID, sVorname, sNachname, sKlasse string
	err := tx.QueryRow(ctx, `
		SELECT v.id, s.vorname, s.nachname, COALESCE(s.klasse, '')
		FROM vormerkungen v
		JOIN schueler s ON v.schueler_id = s.id
		WHERE v.titel_id = $1 AND v.status = 'wartend'
		ORDER BY v.erstellt_am ASC LIMIT 1
		FOR UPDATE
	`, copy.TitelID).Scan(&vID, &sVorname, &sNachname, &sKlasse)

	if err == nil {
		schuelerName := sVorname + " " + sNachname
		if sKlasse != "" {
			schuelerName += ", " + sKlasse
		}

		_, _ = tx.Exec(ctx, "UPDATE vormerkungen SET status = 'abholbereit', bereitgestellt_exemplar_id = $1, bereitgestellt_bis = CURRENT_TIMESTAMP + INTERVAL '3 days' WHERE id = $2", copy.ID, vID)

		resp.HasVormerkung = true
		resp.VormerkungTitel = copy.Titel
		resp.VormerkungUser = schuelerName
	}
}

func (s *defaultLoanService) HandleUnifiedCheckout(
	ctx context.Context,
	copy *repository.BookCopy,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
) (*LoanResult, error) {
	resp := &LoanResult{}
	
	if !copy.IstAusleihbar {
		return nil, fmt.Errorf("%w: dieses Buchexemplar ist nicht ausleihbar", ErrInvalidState)
	}

	var borrowerID string
	var borrowerType string
	var student *repository.Student
	var teacher *repository.User
	var dueTime time.Time

	if activeStudentID != nil && *activeStudentID != "" {
		borrowerType = "student"
		borrowerID = *activeStudentID
		sObj, err := s.studentRepo.GetByID(ctx, borrowerID)
		if err != nil {
			return nil, err
		}
		if sObj == nil {
			return nil, fmt.Errorf("%w: Aktives Schülerprofil nicht gefunden", ErrNotFound)
		}
		if sObj.IstGesperrt {
			return nil, fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
		}
		student = sObj
		dt, err := s.resolveCheckoutDueDate(ctx, copy)
		if err != nil {
			return nil, err
		}
		dueTime = dt
	} else if activeTeacherID != nil && *activeTeacherID != "" {
		borrowerType = "teacher"
		borrowerID = *activeTeacherID
		teacher = &repository.User{}
		err := s.pool.QueryRow(ctx, `
			SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
			FROM benutzer b JOIN benutzer_rollen br ON b.id = br.benutzer_id
			WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true LIMIT 1
		`, borrowerID).Scan(&teacher.ID, &teacher.BarcodeID, &teacher.Vorname, &teacher.Nachname, &teacher.Rolle)
		if err != nil {
			return nil, fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", ErrNotFound)
		}
		dueTime = time.Now().AddDate(1, 0, 0)
	} else {
		return nil, fmt.Errorf("%w: Weder Schüler noch Lehrer aktiv", ErrInvalidState)
	}

	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var activeLoansCount int
	if borrowerType == "student" {
		if _, err = tx.Exec(ctx, "SELECT id FROM schueler WHERE id = $1 FOR UPDATE", borrowerID); err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM ausleihen a
			JOIN buecher_exemplare be ON a.exemplar_id = be.id
			JOIN buecher_titel bt ON be.titel_id = bt.id
			WHERE a.schueler_id = $1 
			  AND a.rueckgabe_am IS NULL
			  AND LOWER(bt.titel) NOT LIKE 'lmf-%'
		`, borrowerID).Scan(&activeLoansCount)
		if err != nil {
			return nil, err
		}
	}

	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	isReturningThis := false
	if activeLoan != nil {
		if borrowerType == "student" && activeLoan.SchuelerID != nil && *activeLoan.SchuelerID == borrowerID {
			isReturningThis = true
		} else if borrowerType == "teacher" && activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == borrowerID {
			isReturningThis = true
		}
	}

	if borrowerType == "student" {
		settings, err := s.querySettings(ctx)
		if err != nil {
			return nil, err
		}

		isLMF := strings.HasPrefix(strings.ToLower(copy.Titel), "lmf-")

		if !isLMF && activeLoansCount >= settings.MaxAusleihenSchueler {
			if !isReturningThis {
				return nil, fmt.Errorf("%w: Ausleihlimit von %d Bibliotheks-Büchern überschritten (aktuell: %d)", ErrBlocked, settings.MaxAusleihenSchueler, activeLoansCount)
			}
		}
	}

	if !isReturningThis {
		var reservedSchuelerID string
		var resVorname, resNachname string
		err = tx.QueryRow(ctx, `
			SELECT v.schueler_id, s.vorname, s.nachname
			FROM vormerkungen v
			JOIN schueler s ON v.schueler_id = s.id
			WHERE v.bereitgestellt_exemplar_id = $1 
			  AND v.status = 'abholbereit' 
			  AND v.bereitgestellt_bis > CURRENT_TIMESTAMP
		`, copy.ID).Scan(&reservedSchuelerID, &resVorname, &resNachname)

		if err == nil {
			if borrowerType != "student" || borrowerID != reservedSchuelerID {
				return nil, fmt.Errorf("%w: Achtung: Dieses Exemplar ist noch für %s %s reserviert!", ErrConflict, resVorname, resNachname)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	if activeLoan == nil {
		var loan *repository.Loan
		if borrowerType == "student" {
			loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
		} else {
			loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
		}
		if err != nil {
			return nil, err
		}
		if borrowerType == "student" {
			_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		if borrowerType == "student" {
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
		}

		resp.Type = "ausleihe"
		resp.Book = copy
		if loan != nil {
			resp.DueDate = &loan.RueckgabeFrist
		}
		return resp, nil
	}

	if isReturningThis {
		if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return nil, err
		}

		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		if borrowerType == "student" {
			_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, borrowerID, "", staffID)
			resp.Student = student
		} else {
			_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", borrowerID, staffID)
			resp.Teacher = teacher
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
		resp.LoanID = &activeLoan.ID
		return resp, nil
	}

	var prevStudent *repository.Student
	var prevTeacher *repository.User

	if activeLoan.SchuelerID != nil {
		prevStudent, _ = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		prevTeacher = &repository.User{}
		err = tx.QueryRow(ctx, "SELECT b.id, b.vorname, b.nachname, COALESCE(br.rolle, 'HELFER') FROM benutzer b LEFT JOIN benutzer_rollen br ON b.id = br.benutzer_id WHERE b.id = $1 LIMIT 1", *activeLoan.AusleiherBenutzerID).Scan(&prevTeacher.ID, &prevTeacher.Vorname, &prevTeacher.Nachname, &prevTeacher.Rolle)
		if errors.Is(err, pgx.ErrNoRows) {
			prevTeacher = nil
		}
	}

	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, true); err != nil {
		return nil, err
	}

	var loan *repository.Loan
	if borrowerType == "student" {
		loan, err = s.loanRepo.CreateLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime)
	} else {
		loan, err = s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, borrowerID, staffID, dueTime, true)
	}
	if err != nil {
		return nil, err
	}

	if borrowerType == "student" {
		_, _ = tx.Exec(ctx, "DELETE FROM vormerkungen WHERE titel_id = $1 AND schueler_id = $2", copy.TitelID, borrowerID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if activeLoan.SchuelerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
	}

	if borrowerType == "student" {
		_ = s.auditRepo.LogAusleihe(ctx, copy.ID, borrowerID, "", staffID)
		resp.Student = student
	} else {
		_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", borrowerID, staffID)
		resp.Teacher = teacher
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
	if loan != nil {
		resp.DueDate = &loan.RueckgabeFrist
	}
	resp.Fremdrueckgabe = true
	resp.Vorbesitzer = prevStudent
	resp.VorbesitzerUser = prevTeacher
	return resp, nil
}

func (s *defaultLoanService) HandleSimpleReturn(
	ctx context.Context,
	copy *repository.BookCopy,
	staffID string,
	staffRole string,
) (*LoanResult, error) {
	resp := &LoanResult{}

	tx, err := s.loanRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	activeLoan, err := s.loanRepo.GetActiveLoanByCopyIDTx(ctx, tx, copy.ID)
	if err != nil {
		return nil, err
	}

	if activeLoan == nil {
		if staffRole == "LEHRER" {
			dueTime := time.Now().AddDate(1, 0, 0) // 1 year
			loan, err := s.loanRepo.CreateUserLoanTx(ctx, tx, copy.ID, staffID, staffID, dueTime, true)
			if err != nil {
				return nil, err
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			_ = s.auditRepo.LogAusleihe(ctx, copy.ID, "", staffID, staffID)

			resp.Type = "ausleihe"
			resp.Book = copy
			if loan != nil {
				resp.DueDate = &loan.RueckgabeFrist
			}
			return resp, nil
		}
		return nil, fmt.Errorf("%w: Dieses Buchexemplar ist aktuell nicht ausgeliehen", ErrInvalidState)
	}

	if activeLoan.AusleiherBenutzerID != nil && *activeLoan.AusleiherBenutzerID == staffID {
		if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
			return nil, err
		}

		s.processReturnVormerkungTx(ctx, tx, copy, resp)

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)

		plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
			CopyID:       copy.ID,
			BarcodeID:    copy.BarcodeID,
			Titel:        copy.Titel,
			SchuelerID:   activeLoan.SchuelerID,
			BearbeiterID: staffID,
		})

		resp.Type = "rueckgabe"
		resp.Book = copy
		resp.LoanID = &activeLoan.ID
		return resp, nil
	}

	var borrowerStudent *repository.Student
	if activeLoan.SchuelerID != nil {
		borrowerStudent, err = s.studentRepo.GetByID(ctx, *activeLoan.SchuelerID)
		if err != nil {
			return nil, err
		}
	}

	if err = s.loanRepo.ReturnLoanTx(ctx, tx, activeLoan.ID, staffID, false); err != nil {
		return nil, err
	}

	s.processReturnVormerkungTx(ctx, tx, copy, resp)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	if activeLoan.SchuelerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, *activeLoan.SchuelerID, "", staffID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = s.auditRepo.LogRueckgabe(ctx, copy.ID, "", *activeLoan.AusleiherBenutzerID, staffID)
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
	resp.Student = borrowerStudent
	resp.LoanID = &activeLoan.ID
	return resp, nil
}
