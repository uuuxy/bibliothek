package api

import (
	"context"
	"testing"

	"bibliothek/db"
	"github.com/pashagolub/pgxmock/v4"
)

func TestQueryReorders(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mock.Close()

	// Setup server with mock DB
	server := &Server{
		DB: &db.Database{Pool: mock},
	}

	// We expect the query from queryReorders to be executed.
	// Since the query string contains newlines and tabs, we can match it using a substring/regex.
	mock.ExpectQuery("SELECT t.id, t.titel, coalesce").
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "verlag", "cover_url", "meldebestand", "verfuegbar"}).
			AddRow("1", "Testbuch", "Testautor", "12345", "Testverlag", "", 5, 2))

	results, err := server.queryReorders(context.Background())
	if err != nil {
		t.Errorf("error was not expected while queryReorders: %s", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].VerfuegbarBestand != 2 {
		t.Errorf("expected VerfuegbarBestand 2, got %d", results[0].VerfuegbarBestand)
	}

	if results[0].Titel != "Testbuch" {
		t.Errorf("expected Titel 'Testbuch', got '%s'", results[0].Titel)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
