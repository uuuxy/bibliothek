package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
func lernmittelAttrappe(t *testing.T) (pgxmock.PgxPoolIface, *BookRepository) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		mock.Close()
	})
	return mock, NewBookRepository(mock)
}

func TestGetLernmittelFaecher_ReichtFehlerWeiter(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	const abfrage = `SELECT subject, COUNT\(\*\) AS titel`

	t.Run("die Abfrage scheitert", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "").WillReturnError(dbFehler)

		faecher, err := repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})
		assert.ErrorIs(t, err, dbFehler)
		assert.Nil(t, faecher)
	})

	t.Run("eine Zeile lässt sich nicht lesen", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "").
			WillReturnRows(pgxmock.NewRows(lernmittelFachSpalten).AddRow("Mathematik", "zwei", 3, 1, 2))

		faecher, err := repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})
		assert.Error(t, err)
		assert.Nil(t, faecher)
	})

	// Der Fehler steht an der zweiten Zeile: Die erste wird gelesen, erst rows.Err() nennt ihn.
	t.Run("die Datenbank bricht nach einer Zeile ab", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "").
			WillReturnRows(pgxmock.NewRows(lernmittelFachSpalten).
				AddRow("Mathematik", 2, 3, 1, 2).RowError(1, dbFehler))

		_, err := repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})
		assert.ErrorIs(t, err, dbFehler)
	})
}

func TestGetLernmittelTitel_ReichtFehlerWeiter(t *testing.T) {
	dbFehler := errors.New("verbindung weg")
	const abfrage = `WITH titel AS`
	// zeile ist ein Physikbuch; jahrgangVon und auflagen stellt der Fall.
	zeile := func(jahrgangVon any, auflagen []byte) *pgxmock.Rows {
		return pgxmock.NewRows(lernmittelTitelSpalten).AddRow(
			"titel-1", "Physik 1", "Autorin", "Physik", "", "9783000000001", jahrgangVon, 6,
			"Gymnasium", "01.01.2026", 10, 2, 8, auflagen)
	}

	t.Run("die Abfrage scheitert", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").WillReturnError(dbFehler)

		titel, err := repo.GetLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		assert.ErrorIs(t, err, dbFehler)
		assert.Nil(t, titel)
	})

	t.Run("eine Zeile lässt sich nicht lesen", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").WillReturnRows(zeile("fünf", nil))

		titel, err := repo.GetLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		assert.Error(t, err)
		assert.Nil(t, titel)
	})

	t.Run("die Auflagen sind kein JSON", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").
			WillReturnRows(zeile(5, []byte(`kein json`)))

		titel, err := repo.GetLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		assert.ErrorContains(t, err, "auflagen von titel-1")
		assert.Nil(t, titel)
	})

	t.Run("die Datenbank bricht nach einer Zeile ab", func(t *testing.T) {
		mock, repo := lernmittelAttrappe(t)
		mock.ExpectQuery(abfrage).WithArgs(0, "", "", false, "Physik").
			WillReturnRows(zeile(5, nil).RowError(1, dbFehler))

		_, err := repo.GetLernmittelTitel(context.Background(), "Physik", false, LernmittelFilter{})
		assert.ErrorIs(t, err, dbFehler)
	})
}
