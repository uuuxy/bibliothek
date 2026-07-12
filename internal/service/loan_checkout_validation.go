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

// logOverride schreibt einen Audit-Eintrag, wenn eine Ausleihsperre manuell
// ignoriert wurde.
func (s *defaultLoanService) logOverride(ctx context.Context, staffID, borrowerID, reason string) {
	logAuditErr(overrideBlockAction, s.auditRepo.LogAdminAktion(ctx, staffID, "OVERRIDE_BLOCK", "", map[string]any{
		"schueler_id": borrowerID,
		"reason":      reason,
	}))
}

// pruefeSchuelerAusleihbar prüft die drei Sperrgründe (manuelle Sperre-Flags und
// die Überfällig-Automatik). Ist overrideBlock gesetzt, wird die Sperre statt
// eines Fehlers nur revisionssicher protokolliert.
func (s *defaultLoanService) pruefeSchuelerAusleihbar(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock bool) error {
	if sObj.IstGesperrt {
		if !overrideBlock {
			return fmt.Errorf("%w: Die Ausleihe für diese/n Schüler/in ist gesperrt", ErrBlocked)
		}
		s.logOverride(ctx, staffID, borrowerID, "Ausleihsperre manuell ignoriert (IstGesperrt)")
	}

	if sObj.IsManuallyBlocked {
		reason := "ohne Grund"
		if sObj.BlockReason != nil && *sObj.BlockReason != "" {
			reason = *sObj.BlockReason
		}
		if !overrideBlock {
			return fmt.Errorf("%w: Manuelle Sperre: %s", ErrBlocked, reason)
		}
		s.logOverride(ctx, staffID, borrowerID, "Ausleihsperre manuell ignoriert (Manuelle Sperre: "+reason+")")
	}

	settings, err := s.querySettings(ctx)
	if err != nil {
		return err
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
	`, borrowerID, settings.MaxOverdueDays).Scan(&overdueCount)
	if errOverdue != nil {
		return errOverdue
	}
	if overdueCount >= settings.MaxOverdueItems {
		if !overrideBlock {
			return fmt.Errorf("%w: %d überfällige Medien vorhanden (Sperr-Automatik)", ErrBlocked, overdueCount)
		}
		s.logOverride(ctx, staffID, borrowerID, fmt.Sprintf("Ausleihsperre manuell ignoriert (überfällig: %d Medien)", overdueCount))
	}
	return nil
}

// resolveStudentBorrower lädt und validiert den aktiven Schüler und bestimmt die Leihfrist.
func (s *defaultLoanService) resolveStudentBorrower(ctx context.Context, copy *repository.BookCopy, studentID, staffID string, overrideBlock bool) (*checkoutContext, error) {
	result := &checkoutContext{borrowerType: "student", borrowerID: studentID}

	sObj, err := s.studentRepo.GetByID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if sObj == nil {
		return nil, fmt.Errorf("%w: Aktives Schülerprofil nicht gefunden", ErrNotFound)
	}

	if err := s.pruefeSchuelerAusleihbar(ctx, sObj, studentID, staffID, overrideBlock); err != nil {
		return nil, err
	}

	result.student = sObj
	dt, err := s.resolveCheckoutDueDate(ctx, copy, result.student.Klasse)
	if err != nil {
		return nil, err
	}
	result.dueTime = dt
	return result, nil
}

// resolveTeacherBorrower lädt den aktiven Lehrer; für Lehrkräfte gilt eine
// Leihfrist von 1 Jahr (Dauerleihgabe / Handapparat).
func (s *defaultLoanService) resolveTeacherBorrower(ctx context.Context, teacherID string) (*checkoutContext, error) {
	result := &checkoutContext{borrowerType: "teacher", borrowerID: teacherID, teacher: &repository.User{}}

	err := s.pool.QueryRow(ctx, `
		SELECT b.id, b.barcode_id, b.vorname, b.nachname, br.rolle
		FROM benutzer b JOIN benutzer_rollen br ON b.id = br.benutzer_id
		WHERE b.id = $1 AND br.rolle = 'LEHRER' AND b.aktiv = true LIMIT 1
	`, teacherID).Scan(&result.teacher.ID, &result.teacher.BarcodeID, &result.teacher.Vorname, &result.teacher.Nachname, &result.teacher.Rolle)
	if err != nil {
		return nil, fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", ErrNotFound)
	}
	result.dueTime = time.Now().AddDate(1, 0, 0)
	return result, nil
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
	if activeStudentID != nil && *activeStudentID != "" {
		return s.resolveStudentBorrower(ctx, copy, *activeStudentID, staffID, overrideBlock)
	}
	if activeTeacherID != nil && *activeTeacherID != "" {
		return s.resolveTeacherBorrower(ctx, *activeTeacherID)
	}
	return nil, fmt.Errorf("%w: Weder Schüler noch Lehrer aktiv", ErrInvalidState)
}
