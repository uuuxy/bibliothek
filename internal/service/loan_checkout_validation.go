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

// pruefeSchuelerAusleihbar prüft die Sperrgründe (Sperr-Flags, offene Schadensrechnungen
// und die Überfällig-Automatik). Ist overrideBlock gesetzt, wird die Sperre statt
// eines Fehlers nur revisionssicher protokolliert.
// pruefeSchuelerAusleihbar führt die vier Sperr-Checks der Reihe nach aus. Jeder Check ist
// in einen eigenen Helfer ausgelagert (reine Extract-Method, keine Logikänderung) — die
// Reihenfolge und das overrideBlock-/Audit-Verhalten sind identisch zum vorherigen Monolithen.
func (s *defaultLoanService) pruefeSchuelerAusleihbar(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock bool) error {
	if err := s.pruefeGesperrt(ctx, sObj, borrowerID, staffID, overrideBlock); err != nil {
		return err
	}
	if err := s.pruefeManuellGesperrt(ctx, sObj, borrowerID, staffID, overrideBlock); err != nil {
		return err
	}
	if err := s.pruefeOffeneSchaeden(ctx, borrowerID, staffID, overrideBlock); err != nil {
		return err
	}
	return s.pruefeUeberfaellig(ctx, borrowerID, staffID, overrideBlock)
}

// pruefeGesperrt blockt einen gesperrten Schüler; overrideBlock hebt die Sperre
// revisionssicher protokolliert auf.
func (s *defaultLoanService) pruefeGesperrt(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock bool) error {
	if !sObj.IstGesperrt {
		return nil
	}
	// Grund mit ausgeben — ein gesperrter Schüler ohne sichtbaren Grund zwingt das
	// Personal sonst, in der Historie zu wühlen. block_reason ist bei Sperre garantiert
	// gefüllt (chk_schueler_block_reason); der Fallback greift nur bei Altbeständen.
	reason := "Grund nicht erfasst"
	if sObj.BlockReason != nil && *sObj.BlockReason != "" {
		reason = *sObj.BlockReason
	}
	if !overrideBlock {
		return fmt.Errorf("%w: Ausleihe gesperrt: %s", ErrBlocked, reason)
	}
	s.logOverride(ctx, staffID, borrowerID, "Ausleihsperre manuell ignoriert (gesperrt: "+reason+")")
	return nil
}

// pruefeManuellGesperrt behandelt die separate manuelle Sperre (is_manually_blocked).
func (s *defaultLoanService) pruefeManuellGesperrt(ctx context.Context, sObj *repository.Student, borrowerID, staffID string, overrideBlock bool) error {
	if !sObj.IsManuallyBlocked {
		return nil
	}
	reason := "ohne Grund"
	if sObj.BlockReason != nil && *sObj.BlockReason != "" {
		reason = *sObj.BlockReason
	}
	if !overrideBlock {
		return fmt.Errorf("%w: Manuelle Sperre: %s", ErrBlocked, reason)
	}
	s.logOverride(ctx, staffID, borrowerID, "Ausleihsperre manuell ignoriert (Manuelle Sperre: "+reason+")")
	return nil
}

// pruefeOffeneSchaeden blockt bei unbezahlten, nicht stornierten Schadensrechnungen: Wer
// einen unbezahlten Schadensfall hat, darf nichts Neues ausleihen, bis die Schule entschädigt
// ist — sonst könnte man Bücher zerstören, die Rechnung ignorieren und sich am nächsten Tag
// neu eindecken. storniert_am setzt ist_bezahlt = true (repository/audit_system.go), daher
// genügt ist_bezahlt = false. overrideBlock möglich, wird dann revisionssicher protokolliert.
func (s *defaultLoanService) pruefeOffeneSchaeden(ctx context.Context, borrowerID, staffID string, overrideBlock bool) error {
	var offeneSchaeden int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`,
		borrowerID,
	).Scan(&offeneSchaeden); err != nil {
		return err
	}
	if offeneSchaeden > 0 {
		if !overrideBlock {
			return fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle offen", ErrBlocked, offeneSchaeden)
		}
		s.logOverride(ctx, staffID, borrowerID, fmt.Sprintf("Ausleihsperre manuell ignoriert (unbezahlte Schäden: %d)", offeneSchaeden))
	}
	return nil
}

// pruefeUeberfaellig setzt die Overdue-Sperr-Automatik um: ab MaxOverdueItems überfälligen
// Medien (älter als MaxOverdueDays) ist die Ausleihe gesperrt.
func (s *defaultLoanService) pruefeUeberfaellig(ctx context.Context, borrowerID, staffID string, overrideBlock bool) error {
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
//
// Die Rolle kommt aus benutzer.rolle — derselben Quelle, die Login/JWT nutzen und
// die das Admin-UI beim Anlegen schreibt. Früher wurde hier gegen benutzer_rollen
// gejoint; diese Tabelle wird aber nur einmalig beim Bootstrap befüllt, sodass
// NEU angelegte Lehrkräfte dort fehlten und keinen Handapparat ausleihen konnten.
func (s *defaultLoanService) resolveTeacherBorrower(ctx context.Context, teacherID string) (*checkoutContext, error) {
	result := &checkoutContext{borrowerType: "teacher", borrowerID: teacherID, teacher: &repository.User{}}

	err := s.pool.QueryRow(ctx, `
		SELECT b.id, b.barcode_id, b.vorname, b.nachname, b.rolle::text
		FROM benutzer b
		WHERE b.id = $1 AND LOWER(b.rolle::text) = 'lehrer' AND b.aktiv = true LIMIT 1
	`, teacherID).Scan(&result.teacher.ID, &result.teacher.BarcodeID, &result.teacher.Vorname, &result.teacher.Nachname, &result.teacher.Rolle)
	if err != nil {
		return nil, fmt.Errorf("%w: Aktives Lehrerprofil nicht gefunden", ErrNotFound)
	}
	// Lehrer-Ausleihe = Handapparat/Dauerleihe (1 Jahr), Tagesende Schul-Zeitzone —
	// dieselbe Normalisierung wie alle anderen Fristen (siehe tagesEndeInSchulzeitzone).
	result.dueTime = tagesEndeInSchulzeitzone(time.Now().In(schoolLocation()).AddDate(1, 0, 0))
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
