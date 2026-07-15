package service

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// expectSettings bedient die system_einstellungen-Abfrage aus querySettings.
func expectSettings(mock pgxmock.PgxPoolIface, maxAusleihen string) {
	mock.ExpectQuery("SELECT schluessel, coalesce\\(wert, ''\\) FROM system_einstellungen").
		WillReturnRows(pgxmock.NewRows([]string{"schluessel", "wert"}).
			AddRow("max_ausleihen_schueler", maxAusleihen))
}

func schuelerCtx(id string) *checkoutContext {
	return &checkoutContext{borrowerType: "student", borrowerID: id}
}

func buchMitTitel(titel string) *repository.BookCopy {
	return &repository.BookCopy{Titel: titel, Medientyp: "Buch", IstAusleihbar: true}
}

// --- Ausleihlimit (max_ausleihen_schueler) ---

func TestPruefeAusleihlimit_ErreichtesLimitBlockiert(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("Der Hobbit"), 5, false)

	if !errors.Is(err, ErrBlocked) {
		t.Errorf("erreichtes Limit (5 von 5) soll ErrBlocked liefern, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_UnterLimitErlaubt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("Der Hobbit"), 4, false)

	if err != nil {
		t.Errorf("4 von 5 Ausleihen soll erlaubt sein, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_LMFBuchIstAusgenommen(t *testing.T) {
	// Lernmittelfreiheit-Bücher zählen nicht gegen das Limit — sonst könnte ein
	// Schüler mit vollem Klassensatz keine regulären Bücher mehr leihen.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("LMF-Mathematik 7"), 10, false)

	if err != nil {
		t.Errorf("LMF-Buch soll trotz überschrittenem Limit durchgehen, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_EigeneRueckgabeIstAusgenommen(t *testing.T) {
	// Beim Zurückgeben des eigenen Buchs darf das Limit nicht blockieren.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	expectSettings(mock, "5")

	err := svc.pruefeSchuelerAusleihlimit(context.Background(), schuelerCtx("s1"), buchMitTitel("Der Hobbit"), 10, true)

	if err != nil {
		t.Errorf("eigene Rückgabe soll trotz Limit durchgehen, bekam: %v", err)
	}
}

func TestPruefeAusleihlimit_LehrerOhneLimit(t *testing.T) {
	// Für Lehrkräfte (Handapparat) gilt kein Limit — die Regel greift vor jeder
	// Settings-Abfrage, daher ist hier bewusst KEINE Query zu erwarten.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()

	lehrerCtx := &checkoutContext{borrowerType: "teacher", borrowerID: "l1"}
	err := svc.pruefeSchuelerAusleihlimit(context.Background(), lehrerCtx, buchMitTitel("Der Hobbit"), 99, false)

	if err != nil {
		t.Errorf("Lehrkraft soll kein Ausleihlimit haben, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("für Lehrkräfte darf keine Settings-Abfrage laufen: %v", err)
	}
}

// --- Vormerkungs-Konflikt (abholbereit reserviertes Exemplar) ---

// beginTx startet eine gemockte Transaktion für die Konflikt-Prüfung.
func beginTx(t *testing.T, mock pgxmock.PgxPoolIface) pgx.Tx {
	t.Helper()
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	return tx
}

const vormerkungQuery = "SELECT v.schueler_id, s.vorname, s.nachname"

func TestVormerkungKonflikt_FremdeReservierungBlockiert(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	mock.ExpectQuery(vormerkungQuery).WithArgs("copy1").
		WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}).
			AddRow("s2", "Erika", "Musterfrau"))

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), false)

	if !errors.Is(err, ErrConflict) {
		t.Errorf("für s2 reserviertes Exemplar darf nicht an s1 gehen, bekam: %v", err)
	}
}

func TestVormerkungKonflikt_EigeneReservierungErlaubt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	mock.ExpectQuery(vormerkungQuery).WithArgs("copy1").
		WillReturnRows(pgxmock.NewRows([]string{"schueler_id", "vorname", "nachname"}).
			AddRow("s1", "Max", "Mustermann"))

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), false)

	if err != nil {
		t.Errorf("eigene Reservierung soll abholbar sein, bekam: %v", err)
	}
}

func TestVormerkungKonflikt_OhneReservierungErlaubt(t *testing.T) {
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	mock.ExpectQuery(vormerkungQuery).WithArgs("copy1").WillReturnError(pgx.ErrNoRows)

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), false)

	if err != nil {
		t.Errorf("ohne Reservierung soll die Ausleihe durchgehen, bekam: %v", err)
	}
}

func TestVormerkungKonflikt_BeiRueckgabeKeinePruefung(t *testing.T) {
	// Eine Rückgabe darf nie an einer Reservierung scheitern — es wird bewusst
	// gar keine Abfrage erwartet.
	svc, _, mock := newValidationService(t, nil)
	defer mock.Close()
	tx := beginTx(t, mock)

	err := svc.pruefeVormerkungKonflikt(context.Background(), tx, "copy1", schuelerCtx("s1"), true)

	if err != nil {
		t.Errorf("Rückgabe soll nicht geprüft werden, bekam: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("bei Rückgabe darf keine Vormerkungs-Abfrage laufen: %v", err)
	}
}

// --- Eigene Rückgabe vs. Fremdrückgabe ---

func TestIstEigeneRueckgabe(t *testing.T) {
	tests := []struct {
		name     string
		chkCtx   *checkoutContext
		loan     *repository.Loan
		expected bool
	}{
		{"keine aktive Ausleihe", schuelerCtx("s1"), nil, false},
		{
			"Schüler scannt eigenes Buch",
			schuelerCtx("s1"),
			&repository.Loan{SchuelerID: strPtr("s1")},
			true,
		},
		{
			"Schüler scannt fremdes Buch",
			schuelerCtx("s1"),
			&repository.Loan{SchuelerID: strPtr("s2")},
			false,
		},
		{
			"Lehrkraft scannt eigenes Buch",
			&checkoutContext{borrowerType: "teacher", borrowerID: "l1"},
			&repository.Loan{AusleiherBenutzerID: strPtr("l1")},
			true,
		},
		{
			"Lehrkraft scannt fremdes Buch",
			&checkoutContext{borrowerType: "teacher", borrowerID: "l1"},
			&repository.Loan{AusleiherBenutzerID: strPtr("l2")},
			false,
		},
		{
			"Schüler scannt Buch einer Lehrkraft",
			schuelerCtx("s1"),
			&repository.Loan{AusleiherBenutzerID: strPtr("l1")},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := istEigeneRueckgabe(tt.chkCtx, tt.loan); got != tt.expected {
				t.Errorf("istEigeneRueckgabe() = %v; want %v", got, tt.expected)
			}
		})
	}
}
