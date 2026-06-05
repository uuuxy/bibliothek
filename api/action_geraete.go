package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/auth"
	"bibliothek/plugins"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

func (s *Server) handleGeraetAction(
	ctx context.Context,
	query string,
	claims *auth.Claims,
	activeStudentID *string,
	activeTeacherID *string,
	confirmedChecklist bool,
	studentRepo repository.StudentRepository,
	loanRepo repository.LoanRepository, // Used only to begin TX
	resp *ActionResponse,
) error {
	// Query geraet by barcode (query = G-1234)
	var g repository.Geraet
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT id, modellname, seriennummer, barcode_id, zubehoer, ist_ausleihbar, ist_ausgesondert, zustand_notiz
		FROM geraete
		WHERE barcode_id = $1
	`, query).Scan(&g.ID, &g.Modellname, &g.Seriennummer, &g.BarcodeID, &g.Zubehoer, &g.IstAusleihbar, &g.IstAusgesondert, &g.ZustandNotiz)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: Gerät mit Barcode %s nicht gefunden", errNotFound, query)
		}
		return err
	}

	if g.IstAusgesondert {
		return fmt.Errorf("%w: Gerät ist ausgesondert", errInvalidState)
	}

	if !g.IstAusleihbar {
		return fmt.Errorf("%w: Gerät ist aktuell gesperrt", errBlocked)
	}

	// Active context handling
	var student *repository.Student
	var teacher *repository.User

	if activeStudentID != nil && *activeStudentID != "" {
		student, err = studentRepo.GetByID(ctx, *activeStudentID)
		if err != nil {
			return err
		}
		if student != nil && student.IstGesperrt {
			return fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", errBlocked)
		}
	} else if activeTeacherID != nil && *activeTeacherID != "" {
		teacher = &repository.User{ID: *activeTeacherID}
	}

	// Begin TX to check active loan and perform action securely
	tx, err := loanRepo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Fetch active loan for this Geraet with row-level lock
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
			return err
		}
	}

	// Checklist Requirement Rule
	if g.Zubehoer != "" && !confirmedChecklist {
		resp.Type = "geraet_check"
		resp.Geraet = &g
		return nil
	}

	// Subcase A: Geraet is not borrowed -> Checkout
	if !hasActiveLoan {
		if student == nil && teacher == nil {
			return fmt.Errorf("%w: Bitte scannen Sie zuerst einen Schüler- oder Lehrerausweis", errInvalidState)
		}

		dueTime := time.Now().AddDate(0, 0, 14) // Standard hardware loan is 14 days
		var newLoanID string
		if student != nil {
			err = tx.QueryRow(ctx, `
				INSERT INTO ausleihen (geraet_id, schueler_id, rueckgabe_frist, bearbeiter_id)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, g.ID, student.ID, dueTime, claims.UserID).Scan(&newLoanID)
			resp.Student = student
		} else {
			err = tx.QueryRow(ctx, `
				INSERT INTO ausleihen (geraet_id, ausleiher_benutzer_id, rueckgabe_frist, bearbeiter_id, ist_handapparat)
				VALUES ($1, $2, $3, $4, true)
				RETURNING id
			`, g.ID, teacher.ID, dueTime, claims.UserID).Scan(&newLoanID)
			resp.Teacher = teacher
		}

		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		auditRepo := repository.NewAuditRepository(s.DB.Pool)
		if student != nil {
			_ = auditRepo.LogAusleihe(ctx, g.ID, student.ID, "", claims.UserID)
		} else {
			_ = auditRepo.LogAusleihe(ctx, g.ID, "", teacher.ID, claims.UserID)
		}

		resp.Type = "ausleihe"
		resp.Geraet = &g
		resp.DueDate = &dueTime
		resp.LoanID = &newLoanID

		return nil
	}

	// Subcase B: Geraet is currently borrowed -> Return

	// Determine if Fremdrueckgabe
	isFremd := false
	if activeLoan.SchuelerID != nil {
		if student == nil || *activeLoan.SchuelerID != student.ID {
			isFremd = true
			var vorbesitzer repository.Student
			err = s.DB.Pool.QueryRow(ctx, "SELECT vorname, nachname, klasse FROM schueler WHERE id = $1", *activeLoan.SchuelerID).Scan(&vorbesitzer.Vorname, &vorbesitzer.Nachname, &vorbesitzer.Klasse)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			} else if errors.Is(err, pgx.ErrNoRows) {
				// Continue with empty struct
			}
			resp.Vorbesitzer = &vorbesitzer
		}
	} else if activeLoan.AusleiherBenutzerID != nil {
		if teacher == nil || *activeLoan.AusleiherBenutzerID != teacher.ID {
			isFremd = true
			var vorbesitzerUser repository.User
			err = s.DB.Pool.QueryRow(ctx, "SELECT vorname, nachname FROM benutzer WHERE id = $1", *activeLoan.AusleiherBenutzerID).Scan(&vorbesitzerUser.Vorname, &vorbesitzerUser.Nachname)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			} else if errors.Is(err, pgx.ErrNoRows) {
				// Continue with empty struct
			}
			resp.VorbesitzerUser = &vorbesitzerUser
		}
	}

	// Process Return
	_, err = tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = $1, ist_fremdrueckgabe = $2
		WHERE id = $3
	`, claims.UserID, isFremd, activeLoan.ID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	auditRepo := repository.NewAuditRepository(s.DB.Pool)
	if activeLoan.SchuelerID != nil {
		_ = auditRepo.LogRueckgabe(ctx, g.ID, *activeLoan.SchuelerID, "", claims.UserID)
	} else if activeLoan.AusleiherBenutzerID != nil {
		_ = auditRepo.LogRueckgabe(ctx, g.ID, "", *activeLoan.AusleiherBenutzerID, claims.UserID)
	}

	resp.Type = "rueckgabe"
	resp.Geraet = &g
	resp.LoanID = &activeLoan.ID
	resp.Fremdrueckgabe = isFremd

	plugins.DispatchEvent(ctx, plugins.EventBookReturned, plugins.BookReturnedPayload{
		CopyID:       g.ID,
		BarcodeID:    g.BarcodeID,
		Titel:        g.Modellname,
		SchuelerID:   activeLoan.SchuelerID,
		BearbeiterID: claims.UserID,
	})

	return nil
}
