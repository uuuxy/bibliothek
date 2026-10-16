package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// Hat der Schüler ein Exemplar des Titels selbst ausgeliehen, merkt er ihn nicht zusätzlich
// vor; sonst bekäme er das Buch bei der Rückgabe sofort wieder. LegeAn liefert
// ErrTitelBereitsAusgeliehen und unterlässt das INSERT.
func TestVormerkungLegeAn_LehntAbWennTitelSchonAusgeliehen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungsAbfragen(mock)

	// Zuerst die Tür-Frage: Ist die Id ein Schüler? (ErrVormerkungNurFuerSchueler)
	mock.ExpectQuery("FROM schueler WHERE id").
		WithArgs("schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs("titel-1", "schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	// Kein ExpectQuery("INSERT …"): Bei bestehender Ausleihe entsteht keine Vormerkung.

	_, err = repo.LegeAn(context.Background(), "titel-1", "Notiz", "schueler-1")
	if !errors.Is(err, ErrTitelBereitsAusgeliehen) {
		t.Fatalf("repo.LegeAn(…) = %v, erwartet ErrTitelBereitsAusgeliehen", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// Der Normalfall: Der Schüler hat den Titel nicht ausgeliehen, die Vormerkung wird angelegt.
func TestVormerkungLegeAn_ErlaubtWennNichtAusgeliehen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungsAbfragen(mock)

	// Zuerst die Tür-Frage: Ist die Id ein Schüler? (ErrVormerkungNurFuerSchueler)
	mock.ExpectQuery("FROM schueler WHERE id").
		WithArgs("schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("FROM ausleihen a").
		WithArgs("titel-1", "schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("INSERT INTO vormerkungen").
		WithArgs("titel-1", "Notiz", "schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("vorm-1"))

	id, err := repo.LegeAn(context.Background(), "titel-1", "Notiz", "schueler-1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if id != "vorm-1" {
		t.Errorf("erwartete ID 'vorm-1', bekam %q", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// Ohne Schüler gibt es keine Prüfung auf eine eigene Ausleihe; es wird direkt eingefügt.
func TestVormerkungLegeAn_OhneSchuelerKeinePruefung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungsAbfragen(mock)

	// Keine Prüfung mit EXISTS erwartet, direkt das INSERT.
	mock.ExpectQuery("INSERT INTO vormerkungen").
		WithArgs("titel-1", "", "").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("vorm-2"))

	id, err := repo.LegeAn(context.Background(), "titel-1", "", "")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if id != "vorm-2" {
		t.Errorf("erwartete ID 'vorm-2', bekam %q", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// LadeListe ohne Titel und ohne Schüler lieferte alle Vormerkungen mit den Namen der Schüler:
// einen Abzug, den die Oberfläche nie anfordert und der Personendaten preisgäbe, sobald das
// Recht manage_vormerkungen an eine Helfer-Rolle vergeben ist. Ohne Filter liefert LadeListe
// ErrVormerkungFilterFehlt und fragt die Datenbank nicht.
func TestVormerkungLadeListe_VerlangtFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	repo := NewVormerkungsAbfragen(mock)

	// Kein ExpectQuery: Es findet keine Abfrage statt.
	_, err = repo.LadeListe(context.Background(), "", "")
	if !errors.Is(err, ErrVormerkungFilterFehlt) {
		t.Fatalf("repo.LadeListe(…) = %v, erwartet ErrVormerkungFilterFehlt ohne Filter", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("es darf keine Query abgesetzt werden: %v", err)
	}
}
