package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// snapshotBenutzerRows liefert die Zeile, die LoescheKonto vor dem Löschen für das Protokoll
// liest.
func snapshotBenutzerRows() *pgxmock.Rows {
	// leser_id ist NULL: Diese Fälle prüfen die Prüfung auf offene Ausleihen. Was mit einer
	// Leserzeile geschieht, prüft der Test am Postgres (konto_loeschen_leserzeile_pg_test.go),
	// weil die Prüfung den Katalog der Fremdschlüssel liest.
	return pgxmock.NewRows([]string{"vorname", "nachname", "email", "rolle", "leser_id"}).
		AddRow("Erika", "Muster", "erika@schule.de", "KOLLEGIUM", nil)
}

// Hat das Konto noch nicht zurückgegebene Ausleihen, bricht LoescheKonto mit
// KontoHatAusleihenFehler ab und setzt das DELETE nicht ab: Sonst verlöre die Leserzeile ihr Konto
// (benutzer.leser_id, ON DELETE SET NULL), und die Bücher blieben ohne zuordenbaren Entleiher
// stehen.
func TestLoescheKonto_LehntAbBeiOffenenAusleihen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewProtokollAbfragen(mock)
	const userID = "user-123"

	mock.ExpectBegin()
	mock.ExpectQuery("FROM benutzer WHERE id").
		WithArgs(userID).
		WillReturnRows(snapshotBenutzerRows())
	mock.ExpectQuery("JOIN benutzer b ON b.leser_id = a.schueler_id").
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
	// Kein ExpectExec("DELETE …"): Das Löschen scheitert vor der Anweisung, die löscht.
	mock.ExpectRollback()

	err = repo.LoescheKonto(context.Background(), userID, "admin-1")
	var mitAusleihen *KontoHatAusleihenFehler
	if !errors.As(err, &mitAusleihen) {
		t.Fatalf("LoescheKonto = %v, erwartet KontoHatAusleihenFehler", err)
	}
	if mitAusleihen.Offen != 2 {
		t.Errorf("LoescheKonto: Offen = %d, erwartet 2", mitAusleihen.Offen)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// Die Prüfung hält den Normalfall nicht auf: Ohne offene Ausleihen wird das Konto gelöscht, das
// Löschen protokolliert und die Transaktion abgeschlossen.
func TestLoescheKonto_LoeschtOhneOffeneAusleihen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewProtokollAbfragen(mock)
	const userID = "user-456"

	mock.ExpectBegin()
	mock.ExpectQuery("FROM benutzer WHERE id").
		WithArgs(userID).
		WillReturnRows(snapshotBenutzerRows())
	mock.ExpectQuery("JOIN benutzer b ON b.leser_id = a.schueler_id").
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

	if err := repo.LoescheKonto(context.Background(), userID, "admin-1"); err != nil {
		t.Fatalf("unerwarteter Fehler bei Löschung ohne offene Ausleihen: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}
