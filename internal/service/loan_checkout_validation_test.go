package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
)

// --- Mocks: nur die im Checkout-Pfad genutzten Methoden tragen Logik,
// der Rest erfüllt das Interface als No-op. ---

type mockStudentRepo struct {
	student *repository.Student
	err     error
}

func (m *mockStudentRepo) GetByID(ctx context.Context, id string) (*repository.Student, error) {
	return m.student, m.err
}
func (m *mockStudentRepo) GetByBarcode(ctx context.Context, barcode string) (*repository.Student, error) {
	return nil, nil
}
func (m *mockStudentRepo) SearchStudentsFuzzy(ctx context.Context, q string, limit int) ([]repository.Student, error) {
	return nil, nil
}
func (m *mockStudentRepo) GetNextSequence(ctx context.Context) (int, error) { return 0, nil }
func (m *mockStudentRepo) GetAllLUSDStudents(ctx context.Context) ([]repository.Student, error) {
	return nil, nil
}
func (m *mockStudentRepo) BulkSyncLUSD(ctx context.Context, u []repository.StudentUpdate, i []repository.StudentInsert, ids []string) (int, error) {
	return 0, nil
}
func (m *mockStudentRepo) HasPhoto(ctx context.Context, id string) (bool, error) { return false, nil }
func (m *mockStudentRepo) HasOpenDamages(ctx context.Context, id string) (bool, error) {
	return false, nil
}
func (m *mockStudentRepo) GetActiveBorrowedBooks(ctx context.Context, id string) ([]repository.BorrowedBook, error) {
	return nil, nil
}
func (m *mockStudentRepo) GetDistinctClasses(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockStudentRepo) ListStudentsWithStats(ctx context.Context, klasse string) ([]repository.StudentListStat, error) {
	return nil, nil
}

type mockAuditRepo struct {
	adminAktionCalls int
}

func (m *mockAuditRepo) LogAdminAktion(ctx context.Context, adminID, aktion, ip string, details map[string]any) error {
	m.adminAktionCalls++
	return nil
}
func (m *mockAuditRepo) DeleteTitle(ctx context.Context, t, b string) error      { return nil }
func (m *mockAuditRepo) DeleteCopy(ctx context.Context, c, b string) error       { return nil }
func (m *mockAuditRepo) DeleteUser(ctx context.Context, u, b string) error       { return nil }
func (m *mockAuditRepo) DeleteStudent(ctx context.Context, s, b, g string) error { return nil }
func (m *mockAuditRepo) StornierungGebuehr(ctx context.Context, s, b string, betrag float64, g string) error {
	return nil
}
func (m *mockAuditRepo) LogAusleihe(ctx context.Context, e, s, bu, b string) error  { return nil }
func (m *mockAuditRepo) LogRueckgabe(ctx context.Context, e, s, bu, b string) error { return nil }
func (m *mockAuditRepo) LogSystemAktion(ctx context.Context, tabelle, aktion, kontext string, details map[string]any) error {
	return nil
}

func strPtr(s string) *string { return &s }

// expectSettingsAndOverdue richtet die Mock-Erwartungen für den erfolgreichen
// Validierungspfad ein: querySettings → Overdue-Zählung → querySettings (in resolveCheckoutDueDate).
func expectSettingsAndOverdue(mock pgxmock.PgxPoolIface, overdueCount int) {
	settingsRows := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", "1").
			AddRow("max_overdue_days", "14")
	}
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").WillReturnRows(settingsRows())
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(overdueCount))
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").WillReturnRows(settingsRows())
}

func newValidationService(t *testing.T, student *repository.Student) (*defaultLoanService, *mockAuditRepo, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	audit := &mockAuditRepo{}
	svc := &defaultLoanService{
		pool:        mock,
		studentRepo: &mockStudentRepo{student: student},
		auditRepo:   audit,
	}
	return svc, audit, mock
}

func activeStudent(id string) *string { return &id }

// --- Tests ---

func TestResolveBorrower_BlockedStudentRejected(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", IstGesperrt: true,
	})
	defer mock.Close()

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil, "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("gesperrter Schüler soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestResolveBorrower_ManualBlockRejected(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", IsManuallyBlocked: true, BlockReason: strPtr("Buch verloren"),
	})
	defer mock.Close()

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil, "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("manuell gesperrter Schüler soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestResolveBorrower_OverrideAllowsBlockedAndAudits(t *testing.T) {
	svc, audit, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", IstGesperrt: true,
	})
	defer mock.Close()

	expectSettingsAndOverdue(mock, 0)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil, "staff1", true)

	if err != nil {
		t.Fatalf("override soll Sperre umgehen, bekam Fehler: %v", err)
	}
	if ctx == nil || ctx.student == nil || ctx.student.ID != "s1" {
		t.Fatal("erwartete aufgelösten Schüler im checkoutContext")
	}
	if audit.adminAktionCalls < 1 {
		t.Errorf("override muss als Admin-Aktion auditiert werden, calls=%d", audit.adminAktionCalls)
	}
}

func TestResolveBorrower_OverdueAutomaticBlock(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a",
	})
	defer mock.Close()

	// Nicht manuell gesperrt, aber 2 überfällige Medien bei MaxOverdueItems=1.
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("max_overdue_items", "1"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil, "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("überfällige Medien über Limit sollen automatisch sperren, bekam: %v", err)
	}
}

func TestResolveBorrower_HappyPath(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mock.Close()

	expectSettingsAndOverdue(mock, 0)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil, "staff1", false)

	if err != nil {
		t.Fatalf("regulärer Schüler ohne Sperre soll durchgehen, bekam: %v", err)
	}
	if ctx.borrowerType != "student" || ctx.borrowerID != "s1" {
		t.Errorf("erwartete borrowerType=student/s1, bekam %q/%q", ctx.borrowerType, ctx.borrowerID)
	}
	if ctx.dueTime.IsZero() {
		t.Error("erwartete gesetztes Fälligkeitsdatum")
	}
}

func TestResolveBorrower_NoActiveBorrower(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, nil, nil, "staff1", false)

	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("weder Schüler noch Lehrer aktiv soll ErrInvalidState liefern, bekam: %v", err)
	}
}
