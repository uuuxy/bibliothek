package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

type rueckgabeAttrappe struct {
	rueckgabeFehler error
	beginnFehler    error
	offeneFehler    error
	tx              pgx.Tx
}

func (m *rueckgabeAttrappe) SperreOffene(ctx context.Context, tx pgx.Tx, copyID string) (*repository.Ausleihe, error) {
	return nil, m.offeneFehler
}
func (m *rueckgabeAttrappe) BeginneTx(ctx context.Context) (pgx.Tx, error) {
	return m.tx, m.beginnFehler
}
func (m *rueckgabeAttrappe) LegeAn(ctx context.Context, tx pgx.Tx, exemplarID, leserID, bearbeiterID string, rueckgabeFrist time.Time, istDauerleihe bool) (*repository.Ausleihe, error) {
	return nil, nil
}
func (m *rueckgabeAttrappe) BucheRueckgabe(ctx context.Context, tx pgx.Tx, loanID, bearbeiterID string, istVerlust bool) error {
	return m.rueckgabeFehler
}

func TestVerarbeiteRueckgabe_RueckgabeFehler(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	expectedErr := errors.New("db error")
	svc := &AusleihDienst{
		pool:            mockPool,
		ausleihAbfragen: &rueckgabeAttrappe{rueckgabeFehler: expectedErr},
		leserAbfragen:   &leserAttrappe{leser: &repository.Leser{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	exemplar := &repository.Exemplar{ID: "copy1"}
	staffID := "staff1"
	// Der Ausleiher steht in einer Spalte — auch, wenn es der Mitarbeiter selbst ist, der das
	// Buch zurückbringt.
	leserID := "leser1"
	activeLoan := &repository.Ausleihe{ID: "loan1", SchuelerID: &leserID}
	resp := &AusleihErgebnis{}

	result, err := svc.verarbeiteRueckgabe(context.Background(), tx, exemplar, activeLoan, staffID, resp)

	if !errors.Is(err, expectedErr) {
		t.Errorf("svc.verarbeiteRueckgabe(…) = %v, erwartet %v", err, expectedErr)
	}
	if result != nil {
		t.Errorf("verarbeiteRueckgabe(…) = %v, erwartet nil", result)
	}
}

func TestVerarbeiteRueckgabe_CommitFehler(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	audit := &protokollAttrappe{}
	svc := &AusleihDienst{
		pool:              mockPool,
		ausleihAbfragen:   &rueckgabeAttrappe{},
		protokollAbfragen: audit,
		leserAbfragen:     &leserAttrappe{leser: &repository.Leser{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leserID := "leser1"
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE\\(s.klasse, ''\\)").
		WithArgs("t1", &leserID).
		WillReturnError(pgx.ErrNoRows)

	expectedErr := errors.New("commit error")
	mockPool.ExpectCommit().WillReturnError(expectedErr)

	exemplar := &repository.Exemplar{ID: "copy1", TitelID: "t1"}
	staffID := "staff1"
	// Der Ausleiher steht in einer Spalte — auch, wenn es der Mitarbeiter selbst ist, der das
	// Buch zurückbringt.
	activeLoan := &repository.Ausleihe{ID: "loan1", SchuelerID: &leserID}
	resp := &AusleihErgebnis{}

	result, err := svc.verarbeiteRueckgabe(context.Background(), tx, exemplar, activeLoan, staffID, resp)

	if !errors.Is(err, expectedErr) {
		t.Errorf("svc.verarbeiteRueckgabe(…) = %v, erwartet %v", err, expectedErr)
	}
	if result != nil {
		t.Errorf("verarbeiteRueckgabe(…) = %v, erwartet nil", result)
	}
}

func TestVerarbeiteRueckgabe_Gelingt(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	audit := &protokollAttrappe{}
	svc := &AusleihDienst{
		pool:              mockPool,
		ausleihAbfragen:   &rueckgabeAttrappe{},
		protokollAbfragen: audit,
		leserAbfragen:     &leserAttrappe{leser: &repository.Leser{ID: "leser1", Art: "lehrkraft"}},
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leserID := "leser1"
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE\\(s.klasse, ''\\)").
		WithArgs("t1", &leserID).
		WillReturnError(pgx.ErrNoRows)

	mockPool.ExpectCommit()

	exemplar := &repository.Exemplar{ID: "c1", TitelID: "t1", BarcodeID: "b1", Titel: "Test Book"}
	staffID := "staff1"
	// Der Ausleiher steht in einer Spalte — auch, wenn es der Mitarbeiter selbst ist, der das
	// Buch zurückbringt.
	activeLoan := &repository.Ausleihe{ID: "loan1", SchuelerID: &leserID}
	resp := &AusleihErgebnis{}

	result, err := svc.verarbeiteRueckgabe(context.Background(), tx, exemplar, activeLoan, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Art != "rueckgabe" {
		t.Errorf("result.Art = %v, erwartet type rueckgabe", result.Art)
	}
	if result.AusleiheID == nil || *result.AusleiheID != "loan1" {
		t.Errorf("result.AusleiheID = %v, erwartet loan1", result.AusleiheID)
	}
}

func TestVerarbeiteRueckgabe_VormerkungAktiviert(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to init pgxmock: %v", err)
	}
	defer mockPool.Close()

	audit := &protokollAttrappe{}
	svc := &AusleihDienst{
		pool:              mockPool,
		ausleihAbfragen:   &rueckgabeAttrappe{},
		protokollAbfragen: audit,
		leserAbfragen:     &leserAttrappe{leser: &repository.Leser{ID: "leser1", Art: "lehrkraft"}},
		// Donnerstag vor den Herbstferien (05.10.–17.10.2026): plus drei Tage ist Sonntag, der
		// nächste Schultag der Montag nach den Ferien.
		jetzt: func() time.Time { return time.Date(2026, time.October, 1, 10, 0, 0, 0, schulzeit.Zone()) },
	}

	mockPool.ExpectBegin()
	tx, err := mockPool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	leserID := "leser1"
	mockPool.ExpectQuery("SELECT v.id, s.vorname, s.nachname, COALESCE\\(s.klasse, ''\\)").
		WithArgs("t1", &leserID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "vorname", "nachname", "klasse"}).
			AddRow("v1", "Max", "Mustermann", "10A"))

	// Die Abholfrist liest die Einstellungen (repository.BestimmeAbholfrist); keine Sommerferien der
	// Schule, es gilt die Programmtabelle.
	mockPool.ExpectQuery("SELECT schluessel, wert FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}))

	// Status der Vormerkung auf 'abholbereit' setzen, bis Montag 19.10.2026 abends.
	mockPool.ExpectExec("UPDATE vormerkungen SET status = 'abholbereit'").
		WithArgs("c1", "v1", time.Date(2026, time.October, 19, 23, 59, 59, 0, schulzeit.Zone())).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mockPool.ExpectCommit()

	exemplar := &repository.Exemplar{ID: "c1", TitelID: "t1", BarcodeID: "b1", Titel: "Test Book"}
	staffID := "staff1"
	// Der Ausleiher steht in einer Spalte — auch, wenn es der Mitarbeiter selbst ist, der das
	// Buch zurückbringt.
	activeLoan := &repository.Ausleihe{ID: "loan1", SchuelerID: &leserID}
	resp := &AusleihErgebnis{}

	result, err := svc.verarbeiteRueckgabe(context.Background(), tx, exemplar, activeLoan, staffID, resp)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Art != "rueckgabe" {
		t.Errorf("result.Art = %v, erwartet type rueckgabe", result.Art)
	}
	if !result.HatVormerkung {
		t.Errorf("verarbeiteRueckgabe(…).HatVormerkung = false, erwartet true")
	}
	if result.VorgemerktFuer != "Max Mustermann, 10A" {
		t.Errorf("verarbeiteRueckgabe(…).VorgemerktFuer = %v, erwartet Max Mustermann, 10A", result.VorgemerktFuer)
	}
	if result.VormerkungTitel != "Test Book" {
		t.Errorf("verarbeiteRueckgabe(…).VormerkungTitel = %v, erwartet Test Book", result.VormerkungTitel)
	}
}

// TestNimmZurueck_ReichtFehlerWeiter: Beginnt die Transaktion nicht oder lässt sich die
// offene Ausleihe nicht lesen, kommt der Fehler an. Verschluckt gälte das Buch als „nicht
// ausgeliehen", und die Theke wiese eine Rückgabe ab, die es gibt.
func TestNimmZurueck_ReichtFehlerWeiter(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	exemplar := &repository.Exemplar{ID: "ex-1"}

	t.Run("die Transaktion beginnt nicht", func(t *testing.T) {
		svc := &AusleihDienst{ausleihAbfragen: &rueckgabeAttrappe{beginnFehler: dbFehler}}

		ergebnis, err := svc.NimmZurueck(context.Background(), exemplar, "theke")
		if !errors.Is(err, dbFehler) {
			t.Errorf("Fehler %v, erwartet %v", err, dbFehler)
		}
		if ergebnis != nil {
			t.Errorf("Ergebnis %+v, erwartet keines", ergebnis)
		}
	})

	t.Run("die offene Ausleihe lässt sich nicht lesen", func(t *testing.T) {
		pool, err := pgxmock.NewPool()
		if err != nil {
			t.Fatalf("pgxmock: %v", err)
		}
		defer pool.Close()
		pool.ExpectBegin()
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("Transaktion der Attrappe: %v", err)
		}
		pool.ExpectRollback()
		svc := &AusleihDienst{ausleihAbfragen: &rueckgabeAttrappe{tx: tx, offeneFehler: dbFehler}}

		ergebnis, err := svc.NimmZurueck(context.Background(), exemplar, "theke")
		if !errors.Is(err, dbFehler) {
			t.Errorf("Fehler %v, erwartet %v", err, dbFehler)
		}
		if ergebnis != nil {
			t.Errorf("Ergebnis %+v, erwartet keines", ergebnis)
		}
		if err := pool.ExpectationsWereMet(); err != nil {
			t.Errorf("die Transaktion wurde nicht zurückgerollt: %v", err)
		}
	})
}
