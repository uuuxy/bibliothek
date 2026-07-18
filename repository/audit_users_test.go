package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// snapshotBenutzerRows liefert die Stammdaten-Zeile, die DeleteUser vor dem Löschen
// für das Audit-Log sichert.
func snapshotBenutzerRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"vorname", "nachname", "email", "rolle"}).
		AddRow("Erika", "Muster", "erika@schule.de", "LEHRER")
}

// TestDeleteUser_RejectsWhenActiveLoans sichert Bug 2 (Stranded Handapparat) ab: Hat ein
// Mitarbeiter/Lehrer noch nicht zurückgegebene Handapparat-Ausleihen, muss DeleteUser mit
// ErrUserHasActiveLoans abbrechen — und darf das DELETE gar nicht erst absetzen, sonst
// blieben die Bücher an ausleiher_benutzer_id = NULL (ON DELETE SET NULL) verwaist.
func TestDeleteUser_RejectsWhenActiveLoans(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewAuditRepository(mock)
	const userID = "user-123"

	mock.ExpectBegin()
	mock.ExpectQuery("FROM benutzer WHERE id").
		WithArgs(userID).
		WillReturnRows(snapshotBenutzerRows())
	mock.ExpectQuery("FROM ausleihen WHERE ausleiher_benutzer_id").
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
	// KEIN ExpectExec("DELETE …"): Die Löschung muss vor dem destruktiven Statement scheitern.
	mock.ExpectRollback()

	err = repo.DeleteUser(context.Background(), userID, "admin-1")
	if !errors.Is(err, ErrUserHasActiveLoans) {
		t.Fatalf("erwartet ErrUserHasActiveLoans, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// TestDeleteUser_SucceedsWhenNoActiveLoans stellt sicher, dass die Pre-Flight-Prüfung den
// Normalfall (keine offenen Ausleihen) nicht blockiert: Der Benutzer wird gelöscht, die
// Löschung ins Audit-Log geschrieben und committet.
func TestDeleteUser_SucceedsWhenNoActiveLoans(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewAuditRepository(mock)
	const userID = "user-456"

	mock.ExpectBegin()
	mock.ExpectQuery("FROM benutzer WHERE id").
		WithArgs(userID).
		WillReturnRows(snapshotBenutzerRows())
	mock.ExpectQuery("FROM ausleihen WHERE ausleiher_benutzer_id").
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("DELETE FROM benutzer WHERE id").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectExec("INSERT INTO audit_log").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if err := repo.DeleteUser(context.Background(), userID, "admin-1"); err != nil {
		t.Fatalf("unerwarteter Fehler bei Löschung ohne offene Ausleihen: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}
