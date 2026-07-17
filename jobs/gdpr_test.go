package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// -- Mocking AuditRepository --

type MockAuditRepo struct {
	DeleteStudentCalls   int
	PurgeAbgaengerCalls  int
	LogSystemAktionCalls int
}

func (m *MockAuditRepo) DeleteTitle(ctx context.Context, titleID string, bearbeiterID string) error {
	return nil
}

func (m *MockAuditRepo) DeleteCopy(ctx context.Context, copyID string, bearbeiterID string) error {
	return nil
}

func (m *MockAuditRepo) DeleteUser(ctx context.Context, userID string, bearbeiterID string) error {
	return nil
}

func (m *MockAuditRepo) DeleteStudent(ctx context.Context, studentID string, bearbeiterID string, grund string) error {
	m.DeleteStudentCalls++
	return nil
}

func (m *MockAuditRepo) PurgeStudent(ctx context.Context, studentID string, bearbeiterID string) error {
	return nil
}

func (m *MockAuditRepo) PurgeAbgaenger(ctx context.Context, studentID string, bearbeiterID string) error {
	m.PurgeAbgaengerCalls++
	return nil
}

func (m *MockAuditRepo) LogAdminAktion(ctx context.Context, adminID string, aktion string, ip string, details map[string]any) error {
	return nil
}

func (m *MockAuditRepo) StornierungGebuehr(ctx context.Context, schadensfallID string, bearbeiterID string, betrag float64, grund string) error {
	return nil
}

func (m *MockAuditRepo) LogAusleihe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return nil
}

func (m *MockAuditRepo) LogRueckgabe(ctx context.Context, exemplarID string, schuelerID string, benutzerID string, bearbeiterID string) error {
	return nil
}

func (m *MockAuditRepo) LogSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	m.LogSystemAktionCalls++
	return nil
}

// -- Tests --

func TestRunGDPRAnonymizeLoans_Anonymized(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	auditMock := &MockAuditRepo{}
	scheduler := NewScheduler(mock, auditMock)

	// Simuliere: 1 Ausleihe wird anonymisiert
	mock.ExpectExec("UPDATE ausleihen SET bearbeiter_id = NULL").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	scheduler.RunGDPRAnonymizeLoans()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if auditMock.LogSystemAktionCalls != 1 {
		t.Errorf("erwartete 1 Audit-Log Aufruf, bekam %d", auditMock.LogSystemAktionCalls)
	}
}

func TestRunGDPRAnonymizeLoans_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	auditMock := &MockAuditRepo{}
	scheduler := NewScheduler(mock, auditMock)

	// Simuliere: 0 Ausleihen (keine abgelaufenen)
	mock.ExpectExec("UPDATE ausleihen SET bearbeiter_id = NULL").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	scheduler.RunGDPRAnonymizeLoans()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if auditMock.LogSystemAktionCalls != 0 {
		t.Errorf("erwartete 0 Audit-Log Aufrufe, bekam %d", auditMock.LogSystemAktionCalls)
	}
}

func TestRunGDPRDeleteAbgaenger_Deleted(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	auditMock := &MockAuditRepo{}
	scheduler := NewScheduler(mock, auditMock)

	// Simuliere: 1 Abgänger wird gefunden
	now := time.Now()
	cutoffYear := now.Year()
	cutoffDate := time.Date(cutoffYear, time.January, 30, 0, 0, 0, 0, time.UTC)
	if now.Before(cutoffDate) {
		cutoffYear--
	}

	rows := pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse", "barcode_id", "abgaenger_jahr"}).
		AddRow("uuid-1234", "Max", "Mustermann", "10A", "12345", 2023)

	mock.ExpectQuery("SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr FROM schueler").
		WithArgs(cutoffYear).
		WillReturnRows(rows)

	scheduler.RunGDPRDeleteAbgaenger()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Der Job muss die ECHTE Löschung (PurgeAbgaenger) rufen, nicht den Soft-Delete
	// (DeleteStudent) — sonst bliebe die PII im Papierkorb.
	if auditMock.PurgeAbgaengerCalls != 1 {
		t.Errorf("erwartete 1 PurgeAbgaenger Aufruf, bekam %d", auditMock.PurgeAbgaengerCalls)
	}
	if auditMock.DeleteStudentCalls != 0 {
		t.Errorf("DeleteStudent (Soft-Delete) darf NICHT gerufen werden, war %d", auditMock.DeleteStudentCalls)
	}
	if auditMock.LogSystemAktionCalls != 1 {
		t.Errorf("erwartete 1 LogSystemAktion Aufruf, bekam %d", auditMock.LogSystemAktionCalls)
	}
}

func TestRunGDPRDeleteAbgaenger_Blocked(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	auditMock := &MockAuditRepo{}
	scheduler := NewScheduler(mock, auditMock)

	// Simuliere: 0 Abgänger werden gefunden (z.B. weil sie noch offene Schulden haben)
	now := time.Now()
	cutoffYear := now.Year()
	cutoffDate := time.Date(cutoffYear, time.January, 30, 0, 0, 0, 0, time.UTC)
	if now.Before(cutoffDate) {
		cutoffYear--
	}

	rows := pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse", "barcode_id", "abgaenger_jahr"})

	mock.ExpectQuery("SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr FROM schueler").
		WithArgs(cutoffYear).
		WillReturnRows(rows)

	scheduler.RunGDPRDeleteAbgaenger()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// 0 Aufrufe, da niemand berechtigt war
	if auditMock.PurgeAbgaengerCalls != 0 {
		t.Errorf("erwartete 0 PurgeAbgaenger Aufrufe, bekam %d", auditMock.PurgeAbgaengerCalls)
	}
	if auditMock.LogSystemAktionCalls != 0 {
		t.Errorf("erwartete 0 LogSystemAktion Aufrufe, bekam %d", auditMock.LogSystemAktionCalls)
	}
}
