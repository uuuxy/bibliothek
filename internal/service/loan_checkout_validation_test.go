package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// --- Mocks: nur die im Checkout-Pfad genutzten Methoden tragen Logik,
// der Rest erfüllt das Interface als No-op. ---

type mockStudentRepo struct {
	student *repository.Student
	err     error
}

// GetLeserByID ist die Abfrage des Ausleihpfads seit Migration 125: Sie liest die TABELLE
// und findet damit auch einen Kollegen.
func (m *mockStudentRepo) GetLeserByID(ctx context.Context, id string) (*repository.Student, error) {
	return m.student, m.err
}
func (m *mockStudentRepo) GetLeserByBarcode(ctx context.Context, barcode string) (*repository.Student, error) {
	return nil, nil
}
func (m *mockStudentRepo) SearchStudentsFuzzy(ctx context.Context, q string, limit int) ([]repository.Student, int, error) {
	return nil, 0, nil
}
func (m *mockStudentRepo) HasPhoto(ctx context.Context, id string) (bool, error) { return false, nil }
func (m *mockStudentRepo) HasOpenDamages(ctx context.Context, id string) (bool, error) {
	return false, nil
}
func (m *mockStudentRepo) GetActiveBorrowedBooks(ctx context.Context, id string) ([]repository.BorrowedBook, error) {
	return nil, nil
}
func (m *mockStudentRepo) GetDistinctClasses(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockStudentRepo) EtikettenZeilen(ctx context.Context, ids []string) ([]repository.SchuelerEtikettZeile, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListStudentsWithStats(ctx context.Context, klassen []string, suche string, sortierung repository.SchuelerSortierung) ([]repository.StudentListStat, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListLeserMitStats(ctx context.Context, klassen []string, suche string, sortierung repository.SchuelerSortierung) ([]repository.StudentListStat, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListEhemaligeWithStats(ctx context.Context, suche string, sortierung repository.SchuelerSortierung) ([]repository.StudentListStat, error) {
	return nil, nil
}

type mockAuditRepo struct {
	adminAktionCalls int
}

func (m *mockAuditRepo) LogAdminAktion(ctx context.Context, adminID, aktion, ip string, details map[string]any) error {
	m.adminAktionCalls++
	return nil
}
func (m *mockAuditRepo) DeleteTitle(ctx context.Context, t, b string) error           { return nil }
func (m *mockAuditRepo) DeleteCopy(ctx context.Context, c, b string) error            { return nil }
func (m *mockAuditRepo) DeleteUser(ctx context.Context, u, b string) error            { return nil }
func (m *mockAuditRepo) DeleteStudent(ctx context.Context, s, b, g string) error      { return nil }
func (m *mockAuditRepo) PurgeStudent(ctx context.Context, s, b string) error          { return nil }
func (m *mockAuditRepo) PurgeAbgaenger(ctx context.Context, s, b string) error        { return nil }
func (m *mockAuditRepo) StornierungGebuehr(ctx context.Context, s, b, g string) error { return nil }
func (m *mockAuditRepo) BezahltGebuehr(ctx context.Context, s, b string) error        { return nil }
func (m *mockAuditRepo) LogAusleihe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *mockAuditRepo) LogRueckgabe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *mockAuditRepo) LogSystemAktion(ctx context.Context, tabelle, aktion, kontext string, details map[string]any) error {
	return nil
}

func strPtr(s string) *string { return &s }

// expectSettingsAndOverdue richtet die Mock-Erwartungen für einen Durchlauf der
// Sperrprüfung ohne Sperre ein: offene Schäden → querySettings → Overdue-Zählung.
func expectSettingsAndOverdue(mock pgxmock.PgxPoolIface, overdueCount int) {
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", "1").
			AddRow("max_overdue_days", "14"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(overdueCount))
}

// expectFristSettings: die eine Einstellungsabfrage, mit der resolveCheckoutDueDate die
// Leihfrist bestimmt.
func expectFristSettings(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("standard_ausleihfrist_tage", "14"))
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

func TestResolveBorrower_HappyPath(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", Vorname: "Max", Nachname: "Mustermann", Art: "schueler",
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"))

	if err != nil {
		t.Fatalf("regulärer Schüler ohne Sperre soll durchgehen, bekam: %v", err)
	}
	if !ctx.istSchueler() || ctx.borrowerID != "s1" {
		t.Errorf("erwartete den Schüler s1, bekam Art %q / %q", ctx.leser.Art, ctx.borrowerID)
	}
	if ctx.dueTime.IsZero() {
		t.Error("erwartete gesetztes Fälligkeitsdatum")
	}
}

// Auflösen sperrt nicht: Ein gesperrter Schüler wird aufgelöst, die Sperre prüft der
// Checkout danach. Sperrte schon das Auflösen, wäre die eigene Rückgabe wieder gesperrt.
func TestResolveBorrower_SperrtNicht(t *testing.T) {
	svc, audit, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", Art: "schueler", IstGesperrt: true, BlockReason: strPtr("überfällig"),
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"))

	if err != nil {
		t.Fatalf("Auflösen darf nicht sperren, bekam: %v", err)
	}
	if ctx.leser == nil || !ctx.leser.IstGesperrt {
		t.Error("erwartete den gesperrten Leser im checkoutContext")
	}
	if audit.adminAktionCalls != 0 {
		t.Errorf("Auflösen darf kein Übergehen protokollieren, calls=%d", audit.adminAktionCalls)
	}
}

func TestResolveBorrower_NoActiveBorrower(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, nil)

	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("ohne aktiven Leser soll ErrInvalidState kommen, bekam: %v", err)
	}
}
