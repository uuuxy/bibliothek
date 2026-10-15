package inventur

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
)

// Die beiden Lernmittel-Abfragen reichen jeden Fehler weiter: den der Abfrage, den beim Lesen
// einer Zeile und den nach der letzten Zeile. An einer echten Datenbank lässt sich keiner davon
// auslösen (lernmittel_pg_test.go prüft die Zahlen). Verschluckt ergäbe er eine leere oder
// gekürzte Liste, die aussieht wie ein Bestand ohne Schulbücher.

var (
	lernmittelFachSpalten  = []string{"subject", "titel", "gesamt", "verliehen", "verfuegbar"}
	lernmittelTitelSpalten = []string{
		"id", "titel", "autor", "subject", "cover_url", "isbn", "jahrgang_von", "jahrgang_bis",
		"track", "gezaehlt", "gesamt", "verliehen", "verfuegbar", "auflagen",
	}
)

// lernmittelAttrappe liefert eine Datenbank-Attrappe und prüft am Ende, dass jede erwartete
// Abfrage kam.
func lernmittelAttrappe(t *testing.T) (pgxmock.PgxPoolIface, *TitelAbfragen) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("offene Erwartungen: %v", err)
		}
		mock.Close()
	})
	return mock, NewTitelAbfragen(mock)
}

func TestLadeLernmittelFaecher_ReichtFehlerWeiter(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	const abfrage = `SELECT subject, COUNT\(\*\) AS titel`

	t.Run("die Abfrage scheitert", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "").WillReturnError(dbFehler)

		faecher, err := abfragen.LadeLernmittelFaecher(context.Background(), LernmittelFilter{})
		if !errors.Is(err, dbFehler) {
			t.Errorf("abfragen.LadeLernmittelFaecher: Fehler %v, erwartet %v", err, dbFehler)
		}
		if faecher != nil {
			t.Errorf("faecher = %v, erwartet nil", faecher)
		}
	})

	t.Run("eine Zeile lässt sich nicht lesen", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "").
			WillReturnRows(pgxmock.NewRows(lernmittelFachSpalten).AddRow("Mathematik", "zwei", 3, 1, 2))

		faecher, err := abfragen.LadeLernmittelFaecher(context.Background(), LernmittelFilter{})
		if err == nil {
			t.Error("abfragen.LadeLernmittelFaecher: kein Fehler, erwartet einen")
		}
		if faecher != nil {
			t.Errorf("faecher = %v, erwartet nil", faecher)
		}
	})

	// Der Fehler steht an der zweiten Zeile: Die erste wird gelesen, erst rows.Err() nennt ihn.
	t.Run("die Datenbank bricht nach einer Zeile ab", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "").
			WillReturnRows(pgxmock.NewRows(lernmittelFachSpalten).
				AddRow("Mathematik", 2, 3, 1, 2).RowError(1, dbFehler))

		_, err := abfragen.LadeLernmittelFaecher(context.Background(), LernmittelFilter{})
		if !errors.Is(err, dbFehler) {
			t.Errorf("abfragen.LadeLernmittelFaecher: Fehler %v, erwartet %v", err, dbFehler)
		}
	})
}

func TestLadeLernmittelTitel_ReichtFehlerWeiter(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	const abfrage = `WITH titel AS`
	// zeile ist ein Physikbuch; jahrgangVon und auflagen stellt der Fall.
	zeile := func(jahrgangVon any, auflagen []byte) *pgxmock.Rows {
		return pgxmock.NewRows(lernmittelTitelSpalten).AddRow(
			"titel-1", "Physik 1", "Autorin", "Physik", "", "9783000000001", jahrgangVon, 6,
			"Gymnasium", "01.01.2026", 10, 2, 8, auflagen)
	}

	t.Run("die Abfrage scheitert", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").WillReturnError(dbFehler)

		titel, err := abfragen.LadeLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		if !errors.Is(err, dbFehler) {
			t.Errorf("abfragen.LadeLernmittelTitel: Fehler %v, erwartet %v", err, dbFehler)
		}
		if titel != nil {
			t.Errorf("titel = %v, erwartet nil", titel)
		}
	})

	t.Run("eine Zeile lässt sich nicht lesen", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").WillReturnRows(zeile("fünf", nil))

		titel, err := abfragen.LadeLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		if err == nil {
			t.Error("abfragen.LadeLernmittelTitel: kein Fehler, erwartet einen")
		}
		if titel != nil {
			t.Errorf("titel = %v, erwartet nil", titel)
		}
	})

	t.Run("die Auflagen sind kein JSON", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").
			WillReturnRows(zeile(5, []byte(`kein json`)))

		titel, err := abfragen.LadeLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		if err == nil || !strings.Contains(err.Error(), "auflagen von titel-1") {
			t.Errorf("abfragen.LadeLernmittelTitel: Fehler %v, erwartet einen mit %q", err, "auflagen von titel-1")
		}
		if titel != nil {
			t.Errorf("titel = %v, erwartet nil", titel)
		}
	})

	t.Run("die Datenbank bricht nach einer Zeile ab", func(t *testing.T) {
		mock, abfragen := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").
			WillReturnRows(zeile(5, nil).RowError(1, dbFehler))

		_, err := abfragen.LadeLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		if !errors.Is(err, dbFehler) {
			t.Errorf("abfragen.LadeLernmittelTitel: Fehler %v, erwartet %v", err, dbFehler)
		}
	})
}
