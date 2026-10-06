package inventur

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetLernmittelFaecher_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	expectedErr := errors.New("db query error")
	mock.ExpectQuery(`SELECT subject, COUNT\(\*\) AS titel`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(expectedErr)

	_, err = repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})

	assert.ErrorIs(t, err, expectedErr)
	assert.ErrorContains(t, err, "lernmittel je fach")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetLernmittelFaecher_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	// Missing one column in the mock response to trigger a scan error
	rows := pgxmock.NewRows([]string{"subject", "titel", "gesamt", "verliehen"}).
		AddRow("Mathematik", 2, 3, 1)

	mock.ExpectQuery(`SELECT subject, COUNT\(\*\) AS titel`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	_, err = repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})

	assert.Error(t, err)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetLernmittelFaecher_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	rows := pgxmock.NewRows([]string{"subject", "titel", "gesamt", "verliehen", "verfuegbar"}).
		AddRow("Mathematik", 2, 3, 1, 2).
		AddRow("", 1, 1, 0, 1)

	mock.ExpectQuery(`SELECT subject, COUNT\(\*\) AS titel`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	result, err := repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	assert.Equal(t, "Mathematik", result[0].Fach)
	assert.Equal(t, 2, result[0].Titel)
	assert.Equal(t, 3, result[0].Gesamt)
	assert.Equal(t, 1, result[0].Verliehen)
	assert.Equal(t, 2, result[0].Verfuegbar)

	assert.Equal(t, "", result[1].Fach)
	assert.Equal(t, 1, result[1].Titel)
	assert.Equal(t, 1, result[1].Gesamt)
	assert.Equal(t, 0, result[1].Verliehen)
	assert.Equal(t, 1, result[1].Verfuegbar)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetLernmittelFaecher_RowsErr(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	repo := NewBookRepository(mock)

	expectedErr := errors.New("rows error")
	rows := pgxmock.NewRows([]string{"subject", "titel", "gesamt", "verliehen", "verfuegbar"}).
		AddRow("Mathematik", 2, 3, 1, 2).
		RowError(0, expectedErr)

	mock.ExpectQuery(`SELECT subject, COUNT\(\*\) AS titel`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	_, err = repo.GetLernmittelFaecher(context.Background(), LernmittelFilter{})

	assert.ErrorIs(t, err, expectedErr)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
