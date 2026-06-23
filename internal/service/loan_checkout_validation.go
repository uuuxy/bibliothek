package service

import (
	"context"
	"fmt"
	"time"

	"bibliothek/repository"
)

// checkoutContext holds the resolved borrower information and the due time.
type checkoutContext struct {
	borrowerID   string
	borrowerType string
	student      *repository.Student
	teacher      *repository.User
	dueTime      time.Time
}

// resolveBorrowerAndDueTime validates the borrower (student or teacher) and determines the due date.
func (s *defaultLoanService) resolveBorrowerAndDueTime(
	ctx context.Context,
	copy *repository.BookCopy,
	activeStudentID *string,
	activeTeacherID *string,
	staffID string,
	overrideBlock bool,
) (*checkoutContext, error) {
	result := &checkoutContext{}

	if activeStudentID != nil && *activeStudentID != "" {
		result.borrowerType = "student"
		result.borrowerID = *activeStudentID
		sObj, err := s.studentRepo.GetByID(ctx, result.borrowerID)
		if err != nil {
			return nil, err
		}
		if sObj == nil {
			return nil, fmt.Errorf("%w: Aktives Schülerprofil nicht gefunden", ErrNotFound)
		}

		if sObj.IstGesperrt {
			if !overrideBlock {
				return nil, fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
			}
			logAuditErr("override-block", s.auditRepo.LogAdminAktion(ctx, staffID, "OVERRIDE_BLOCK", "", map[string]any{
				"schueler_id": result.borrowerID,
				"reason":      "Ausleihsperre manuell ignoriert (IstGesperrt)",
			}))
		}

		if sObj.IsManuallyBlocked {
			reason := "ohne Grund"
			if sObj.BlockReason != nil && *sObj.BlockReason != "" {
				reason = *sObj.BlockReason
			}
			if !overrideBlock {
				return nil, fmt.Errorf("%w: Manuelle Sperre: %s", ErrBlocked, reason)
			}
			logAuditErr("override-block", s.auditRepo.LogAdminAktion(ctx, staffID, "OVERRIDE_BLOCK", "", map[string]any{
				"schueler_id": result.borrowerID,
				"reason":      "Ausleihsperre manuell ignoriert (Manuelle Sperre: " + reason + ")",
			}))
		}

		settings, err := s.querySettings(ctx)
		if err != nil {
			return nil, err
		}

		var overdueCount int
		errOverdue := s.pool.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM ausleihen 
			WHERE schueler_id = $1 
			  AND rueckgabe_am IS NULL 
			  AND rueckgabe_frist < CURRENT_TIMESTAMP - (INTERVAL '1 day' * $2)
			  AND ist_handapparat = false
			  AND geraet_id IS NULL
		`, result.borrowerID, settings.MaxOverdueDays).Scan(&overdueCount)

		if errOverdue != nil {
			return nil, errOverdue
		}
		if overdueCount >= settings.MaxOverdueItems {
			if !overrideBlock {
				return nil, fmt.Errorf("%w: %d überfällige Medien vorhanden (Sperr-Automatik)", ErrBlocked, overdueCount)
			}
			logAuditErr("override-block", s.auditRepo.LogAdminAktion(ctx, staffID, "OVERRIDE_BLOCK", "", map[string]any{
				"schueler_id": result.borrowerID,
				"reason":      fmt.Sprintf("Ausleihsperre manuell ignoriert (überfällig: %d Medien)", overdueCount),
			}))
		}

		result.student = sObj
		dt, err := s.resolveCheckoutDueDate(ctx, copy, result.student.Klasse)
		if err != nil {
			return nil, err
		}
		result.dueTime = dt

	} else if activeTeacherID != nil && *activeTeacherID != "" {
		result.borrowerType = "teacher"
		result.borrowerID = *activeTeacherID
		result.teacher = &repository.User{}

		err := s.pool.QueryRow(ctx, `
			SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle 
			FROM benutzer b JOIN benutzer_rollen br ON b.id = br.benutzer_id
			WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true LIMIT 1
		`, result.borrowerID).Scan(&result.teacher.ID, &result.teacher.BarcodeID, &result.teacher.Vorname, &result.teacher.Nachname, &result.teacher.Rolle)
		if err != nil {
			return nil, fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", ErrNotFound)
		}
		// Für Lehrkräfte gilt standardmäßig eine Leihfrist von 1 Jahr (Dauerleihgabe / Handapparat)
		result.dueTime = time.Now().AddDate(1, 0, 0)
	} else {
		return nil, fmt.Errorf("%w: Weder Schüler noch Lehrer aktiv", ErrInvalidState)
	}

	return result, nil
}
