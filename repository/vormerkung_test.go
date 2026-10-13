package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// TestVormerkungLegeAn_LehntAbWennTitelSchonAusgeliehen sichert Bug 4 (Vormerkungs-
// Monopolisierung) ab: Hat der Schüler ein Exemplar dieses Titels bereits selbst
// ausgeliehen, darf er ihn nicht zusätzlich vormerken — sonst könnte er das Buch bei der
// Rückgabe sofort wieder für sich abgreifen. Create muss ErrTitelBereitsAusgeliehen liefern
// und das INSERT unterlassen.
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
	// KEIN ExpectQuery("INSERT …"): Bei bestehender Ausleihe darf keine Vormerkung entstehen.

	_, err = repo.LegeAn(context.Background(), "titel-1", "Notiz", "schueler-1")
	if !errors.Is(err, ErrTitelBereitsAusgeliehen) {
		t.Fatalf("erwartet ErrTitelBereitsAusgeliehen, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene/unerwartete Mock-Erwartungen: %v", err)
	}
}

// TestVormerkungLegeAn_ErlaubtWennNichtAusgeliehen: Der Normalfall — der Schüler hat den Titel
// nicht ausgeliehen, die Vormerkung wird angelegt.
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

// TestVormerkungLegeAn_OhneSchuelerKeinePruefung: Ohne Schüler (anonyme Vormerkung) gibt es
// keinen Ausleih-Check — es wird direkt eingefügt.
func TestVormerkungLegeAn_OhneSchuelerKeinePruefung(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewVormerkungsAbfragen(mock)

	// KEIN EXISTS-Check erwartet, direkt das INSERT.
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

// TestVormerkungLadeListe_VerlangtFilter belegt die Datenminimierung (IDOR-Sweep 19.08.2026):
// Ein List-Aufruf ohne Titel- UND ohne Schüler-Filter lieferte früher den GESAMTEN
// Vormerkungsbestand mit allen Schülernamen — ein Voll-Abzug, den die Oberfläche nie
// anfordert und der im Helfer-Opt-in-Fall (manage_vormerkungen) zum PII-Dump würde.
// Ohne Scope muss List ErrVormerkungFilterFehlt liefern und KEINE Query absetzen.
func TestVormerkungLadeListe_VerlangtFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	repo := NewVormerkungsAbfragen(mock)

	// Keine ExpectQuery: Es darf gar keine Datenbankabfrage stattfinden.
	_, err = repo.LadeListe(context.Background(), "", "")
	if !errors.Is(err, ErrVormerkungFilterFehlt) {
		t.Fatalf("erwartet ErrVormerkungFilterFehlt ohne Filter, war: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("es darf keine Query abgesetzt werden: %v", err)
	}
}
