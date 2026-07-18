package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestVormerkungCreate_RejectsWhenTitleAlreadyBorrowed sichert Bug 4 (Vormerkungs-
// Monopolisierung) ab: Hat der Schüler ein Exemplar dieses Titels bereits selbst
// ausgeliehen, darf er ihn nicht zusätzlich vormerken — sonst könnte er das Buch bei der
// Rückgabe sofort wieder für sich abgreifen. Create muss ErrTitelBereitsAusgeliehen liefern
// und das INSERT unterlassen.
func TestVormerkungCreate_RejectsWhenTitleAlreadyBorrowed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungRepository(mock)

	mock.ExpectQuery("FROM ausleihen a").
		WithArgs("titel-1", "schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	// KEIN ExpectQuery("INSERT …"): Bei bestehender Ausleihe darf keine Vormerkung entstehen.

	_, err = repo.Create(context.Background(), "titel-1", "Notiz", "schueler-1")
	if !errors.Is(err, ErrTitelBereitsAusgeliehen) {
		t.Fatalf("erwartet ErrTitelBereitsAusgeliehen, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// TestVormerkungCreate_AllowsWhenNotBorrowed: Der Normalfall — der Schüler hat den Titel
// nicht ausgeliehen, die Vormerkung wird angelegt.
func TestVormerkungCreate_AllowsWhenNotBorrowed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungRepository(mock)

	mock.ExpectQuery("FROM ausleihen a").
		WithArgs("titel-1", "schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("INSERT INTO vormerkungen").
		WithArgs("titel-1", "Notiz", "schueler-1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("vorm-1"))

	id, err := repo.Create(context.Background(), "titel-1", "Notiz", "schueler-1")
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

// TestVormerkungCreate_AnonymousSkipsBorrowCheck: Ohne Schüler (anonyme Vormerkung) gibt es
// keinen Ausleih-Check — es wird direkt eingefügt.
func TestVormerkungCreate_AnonymousSkipsBorrowCheck(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungRepository(mock)

	// KEIN EXISTS-Check erwartet, direkt das INSERT.
	mock.ExpectQuery("INSERT INTO vormerkungen").
		WithArgs("titel-1", "", "").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("vorm-2"))

	id, err := repo.Create(context.Background(), "titel-1", "", "")
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
