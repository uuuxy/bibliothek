package jobs

import (
	"context"
	"testing"
	"time"

	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

// protokollAttrappe steht für das Protokoll der Läufe (systemProtokoll) und für die Löschung der
// Leser (leserLoeschung) und zählt die Aufrufe, nach denen die Tests fragen.
type protokollAttrappe struct {
	abgaengerLoeschungen int
	systemEintraege      int
}

func (p *protokollAttrappe) LoescheKollegenAusDemPapierkorb(ctx context.Context, leserID string) (bool, error) {
	return true, nil
}

func (p *protokollAttrappe) LoescheAbgaengerEndgueltig(ctx context.Context, leserID string, bearbeiterID string) error {
	p.abgaengerLoeschungen++
	return nil
}

func (p *protokollAttrappe) ProtokolliereSystemAktion(ctx context.Context, tabelle string, aktion string, kontext string, details map[string]any) error {
	p.systemEintraege++
	return nil
}

func TestBearbeiterAnonymisierung_Anonymisiert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %s", err)
	}
	defer mock.Close()

	protokoll := &protokollAttrappe{}
	z := NewZeitplan(mock, protokoll)
	z.leserLoeschung = protokoll

	// Eine Ausleihe wird anonymisiert.
	mock.ExpectExec("UPDATE ausleihen SET bearbeiter_id = NULL").
		WithArgs(repository.BearbeiterKennungTage, repository.KulanzJob).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	z.FuehreBearbeiterAnonymisierungAus(t.Context())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen an die Attrappe: %s", err)
	}

	if protokoll.systemEintraege != 1 {
		t.Errorf("erwartete 1 Audit-Log Aufruf, bekam %d", protokoll.systemEintraege)
	}
}

func TestBearbeiterAnonymisierung_OhneTreffer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %s", err)
	}
	defer mock.Close()

	protokoll := &protokollAttrappe{}
	z := NewZeitplan(mock, protokoll)
	z.leserLoeschung = protokoll

	// Keine Ausleihe ist alt genug.
	mock.ExpectExec("UPDATE ausleihen SET bearbeiter_id = NULL").
		WithArgs(repository.BearbeiterKennungTage, repository.KulanzJob).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	z.FuehreBearbeiterAnonymisierungAus(t.Context())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen an die Attrappe: %s", err)
	}

	if protokoll.systemEintraege != 0 {
		t.Errorf("erwartete 0 Audit-Log Aufrufe, bekam %d", protokoll.systemEintraege)
	}
}

func TestAbgaengerLoeschung_Loescht(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %s", err)
	}
	defer mock.Close()

	protokoll := &protokollAttrappe{}
	z := NewZeitplan(mock, protokoll)
	z.leserLoeschung = protokoll

	// Ein Abgänger ist reif für die Löschung.
	jetzt := time.Now()
	stichjahr := jetzt.Year()
	stichtag := time.Date(stichjahr, time.January, 30, 0, 0, 0, 0, time.UTC)
	if jetzt.Before(stichtag) {
		stichjahr--
	}

	zeilen := pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse", "barcode_id", "abgaenger_jahr"}).
		AddRow("uuid-1234", "Max", "Mustermann", "10A", "12345", 2023)

	mock.ExpectQuery("SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr FROM schueler").
		WithArgs(stichjahr).
		WillReturnRows(zeilen)

	z.FuehreAbgaengerLoeschungAus(t.Context())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen an die Attrappe: %s", err)
	}

	// Der Lauf muss endgültig löschen (LoescheAbgaengerEndgueltig); in den Papierkorb legen kann
	// er nicht, leserLoeschung bietet nur das endgültige Löschen. Im Papierkorb blieben die
	// Personendaten.
	if protokoll.abgaengerLoeschungen != 1 {
		t.Errorf("erwartete 1 LoescheAbgaengerEndgueltig Aufruf, bekam %d", protokoll.abgaengerLoeschungen)
	}
	if protokoll.systemEintraege != 1 {
		t.Errorf("erwartete 1 ProtokolliereSystemAktion Aufruf, bekam %d", protokoll.systemEintraege)
	}
}

func TestAbgaengerLoeschung_OhneLoeschreife(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %s", err)
	}
	defer mock.Close()

	protokoll := &protokollAttrappe{}
	z := NewZeitplan(mock, protokoll)
	z.leserLoeschung = protokoll

	// Kein Abgänger ist reif, etwa weil noch Forderungen offen sind.
	jetzt := time.Now()
	stichjahr := jetzt.Year()
	stichtag := time.Date(stichjahr, time.January, 30, 0, 0, 0, 0, time.UTC)
	if jetzt.Before(stichtag) {
		stichjahr--
	}

	zeilen := pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse", "barcode_id", "abgaenger_jahr"})

	mock.ExpectQuery("SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr FROM schueler").
		WithArgs(stichjahr).
		WillReturnRows(zeilen)

	z.FuehreAbgaengerLoeschungAus(t.Context())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("offene Erwartungen an die Attrappe: %s", err)
	}

	// Keine Aufrufe, denn niemand war reif.
	if protokoll.abgaengerLoeschungen != 0 {
		t.Errorf("erwartete 0 LoescheAbgaengerEndgueltig Aufrufe, bekam %d", protokoll.abgaengerLoeschungen)
	}
	if protokoll.systemEintraege != 0 {
		t.Errorf("erwartete 0 ProtokolliereSystemAktion Aufrufe, bekam %d", protokoll.systemEintraege)
	}
}
