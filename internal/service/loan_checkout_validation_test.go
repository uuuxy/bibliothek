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
	student *repository.Leser
	err     error
}

// Finde ist die Abfrage des Ausleihpfads seit Migration 125: Sie liest die TABELLE
// und findet damit auch einen Kollegen.
func (m *mockStudentRepo) Finde(ctx context.Context, id string) (*repository.Leser, error) {
	return m.student, m.err
}
func (m *mockStudentRepo) FindeZuBarcode(ctx context.Context, barcode string) (*repository.Leser, error) {
	return nil, nil
}
func (m *mockStudentRepo) FindeUnscharf(ctx context.Context, q string, limit int) ([]repository.Leser, int, error) {
	return nil, 0, nil
}
func (m *mockStudentRepo) HatFoto(ctx context.Context, id string) (bool, error) { return false, nil }
func (m *mockStudentRepo) ZaehleOffeneSchaeden(ctx context.Context, id string) (int, float64, error) {
	return 0, 0, nil
}
func (m *mockStudentRepo) LadeLaufendeAusleihen(ctx context.Context, id string) ([]repository.LaufendeAusleihe, error) {
	return nil, nil
}
func (m *mockStudentRepo) LadeKlassen(ctx context.Context) ([]string, error) { return nil, nil }
func (m *mockStudentRepo) LadeEtikettZeilen(ctx context.Context, ids []string) ([]repository.LeserEtikettZeile, error) {
	return nil, nil
}
func (m *mockStudentRepo) LadeSchuelerListe(ctx context.Context, klassen []string, suche string, sortierung repository.LeserSortierung) ([]repository.LeserListenZeile, error) {
	return nil, nil
}
func (m *mockStudentRepo) LadeLeserListe(ctx context.Context, klassen []string, suche string, sortierung repository.LeserSortierung) ([]repository.LeserListenZeile, error) {
	return nil, nil
}
func (m *mockStudentRepo) LadeEhemaligenListe(ctx context.Context, suche string, sortierung repository.LeserSortierung) ([]repository.LeserListenZeile, error) {
	return nil, nil
}

type mockAuditRepo struct {
	adminAktionCalls int
}

func (m *mockAuditRepo) ProtokolliereAdminAktion(ctx context.Context, adminID, aktion, ip string, details map[string]any) error {
	m.adminAktionCalls++
	return nil
}
func (m *mockAuditRepo) LoescheTitel(ctx context.Context, t, b string) error             { return nil }
func (m *mockAuditRepo) SondereExemplarAus(ctx context.Context, c, b string) error       { return nil }
func (m *mockAuditRepo) LoescheKonto(ctx context.Context, u, b string) error             { return nil }
func (m *mockAuditRepo) LegeLeserInPapierkorb(ctx context.Context, s, b, g string) error { return nil }
func (m *mockAuditRepo) LoescheLeserEndgueltig(ctx context.Context, s, b string) error   { return nil }
func (m *mockAuditRepo) LoescheAbgaengerEndgueltig(ctx context.Context, s, b string) error {
	return nil
}
func (m *mockAuditRepo) StorniereForderung(ctx context.Context, s, b, g string) error { return nil }
func (m *mockAuditRepo) VerbucheZahlung(ctx context.Context, s, b string) error       { return nil }
func (m *mockAuditRepo) ProtokolliereAusleihe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *mockAuditRepo) ProtokolliereRueckgabe(ctx context.Context, tx pgx.Tx, e, s, bu, b string) error {
	return nil
}
func (m *mockAuditRepo) ProtokolliereSystemAktion(ctx context.Context, tabelle, aktion, kontext string, details map[string]any) error {
	return nil
}

func strPtr(s string) *string { return &s }

// expectSettingsAndOverdue richtet die Mock-Erwartungen für einen Durchlauf der
// Sperrprüfung ohne Sperre ein: offene Schäden → querySettings → Overdue-Zählung.
func expectSettingsAndOverdue(mock pgxmock.PgxPoolIface, overdueCount int) {
	mock.ExpectQuery("SELECT COUNT\\(\\*\\), COALESCE\\(SUM\\(betrag\\), 0\\)::float8 FROM schadensfaelle").
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"count", "summe"}).AddRow(0, 0.0))
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_overdue_items", strPtr("1")).
			AddRow("max_overdue_days", strPtr("14")))
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("s1", 14).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(overdueCount))
}

// expectFristSettings: die eine Einstellungsabfrage, mit der resolveCheckoutDueDate die
// Leihfrist bestimmt.
func expectFristSettings(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("standard_ausleihfrist_tage", strPtr("14")))
}

func newValidationService(t *testing.T, student *repository.Leser) (*defaultLoanService, *mockAuditRepo, pgxmock.PgxPoolIface) {
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
	svc, _, mock := newValidationService(t, &repository.Leser{
		ID: "s1", Klasse: "5a", Vorname: "Max", Nachname: "Mustermann", Art: "schueler",
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
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
	svc, audit, mock := newValidationService(t, &repository.Leser{
		ID: "s1", Klasse: "5a", Art: "schueler", IstGesperrt: true, Sperrgrund: strPtr("überfällig"),
	})
	defer mock.Close()

	expectFristSettings(mock)

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
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

	copy := &repository.Exemplar{Titel: "Der Hobbit", Medientyp: "Buch", IstAusleihbar: true}
	_, err := svc.resolveBorrowerAndDueTime(context.Background(), copy, nil)

	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("ohne aktiven Leser soll ErrInvalidState kommen, bekam: %v", err)
	}
}
