package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

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
func (m *mockStudentRepo) ListStudentsWithStats(ctx context.Context, klasse, suche string) ([]repository.StudentListStat, error) {
	return nil, nil
}
func (m *mockStudentRepo) ListEhemaligeWithStats(ctx context.Context, suche string) ([]repository.StudentListStat, error) {
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

// Die Sperrprüfung hängt seit 11.09.2026 nicht mehr am Auflösen des Ausleihers:
// HandleUnifiedCheckout ruft sie erst, wenn feststeht, dass es keine eigene Rückgabe ist
// (api/theke_eigene_rueckgabe_pg_test.go). Die Fälle hier prüfen sie deshalb direkt.

func TestSperrpruefung_GesperrterSchuelerAbgewiesen(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a", IstGesperrt: true}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("gesperrter Schüler soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestSperrpruefung_ManuelleSperreAbgewiesen(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a", IsManuallyBlocked: true, BlockReason: strPtr("Buch verloren")}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("manuell gesperrter Schüler soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestSperrpruefung_UebergehenLaesstDurchUndProtokolliert(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a", IstGesperrt: true}
	svc, audit, mock := newValidationService(t, schueler)
	defer mock.Close()

	expectSettingsAndOverdue(mock, 0)

	if err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", true); err != nil {
		t.Fatalf("override soll Sperre umgehen, bekam Fehler: %v", err)
	}
	if audit.adminAktionCalls < 1 {
		t.Errorf("override muss als Admin-Aktion auditiert werden, calls=%d", audit.adminAktionCalls)
	}
}

func TestSperrpruefung_UeberfaelligSperrtAutomatisch(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	// Nicht manuell gesperrt, keine offenen Schäden, aber 2 überfällige Medien bei MaxOverdueItems=1.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).AddRow("max_overdue_items", "1"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("überfällige Medien über Limit sollen automatisch sperren, bekam: %v", err)
	}
}

func TestSperrpruefung_OffenerSchadenSperrt(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, _, mock := newValidationService(t, schueler)
	defer mock.Close()

	// Kein Sperr-Flag, aber ein offener (unbezahlter) Schadensfall -> automatische Sperre,
	// noch bevor Settings/Overdue überhaupt abgefragt werden.
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("offener Schadensfall soll automatisch sperren, bekam: %v", err)
	}
}

func TestSperrpruefung_OffenerSchadenUebergangen(t *testing.T) {
	schueler := &repository.Student{ID: "s1", Klasse: "5a"}
	svc, audit, mock := newValidationService(t, schueler)
	defer mock.Close()

	// Offener Schadensfall, aber overrideBlock=true: Ausleihe geht durch, wird auditiert.
	// Danach laufen Settings/Overdue wie im Happy-Path (0 überfällig).
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", "1").AddRow("max_overdue_days", "14"))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	if err := svc.pruefeSchuelerAusleihbar(context.Background(), schueler, "s1", "staff1", true); err != nil {
		t.Fatalf("override soll offene Schäden umgehen, bekam Fehler: %v", err)
	}
	if audit.adminAktionCalls < 1 {
		t.Errorf("override muss als Admin-Aktion auditiert werden, calls=%d", audit.adminAktionCalls)
	}
}

func TestResolveBorrower_HappyPath(t *testing.T) {
	svc, _, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", Vorname: "Max", Nachname: "Mustermann",
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil)

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

// Auflösen sperrt nicht: Ein gesperrter Schüler wird aufgelöst, die Sperre prüft der
// Checkout danach. Sperrte schon das Auflösen, wäre die eigene Rückgabe wieder gesperrt.
func TestResolveBorrower_SperrtNicht(t *testing.T) {
	svc, audit, mock := newValidationService(t, &repository.Student{
		ID: "s1", Klasse: "5a", IstGesperrt: true, BlockReason: strPtr("überfällig"),
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	ctx, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, activeStudent("s1"), nil)

	if err != nil {
		t.Fatalf("Auflösen darf nicht sperren, bekam: %v", err)
	}
	if ctx.student == nil || !ctx.student.IstGesperrt {
		t.Error("erwartete den gesperrten Schüler im checkoutContext")
	}
	if audit.adminAktionCalls != 0 {
		t.Errorf("Auflösen darf kein Übergehen protokollieren, calls=%d", audit.adminAktionCalls)
	}
}

func TestResolveBorrower_NoActiveBorrower(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	copy := &repository.BookCopy{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, nil, nil)

	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("weder Schüler noch Lehrer aktiv soll ErrInvalidState liefern, bekam: %v", err)
	}
}
