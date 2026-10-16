package portal

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

func TestMonitorFolien_Erfolg(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// ladeMeistgelesene (Buch des Monats)
	mock.ExpectQuery(`SELECT.*FROM ausleihen a.*`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()). // args for tage and limit
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "cover_url", "isbn"}).
			AddRow("id-1", "Buch 1", "Autor 1", "http://cover.com/1", "123"))

	// ladeNeuEingetroffene
	mock.ExpectQuery(`SELECT.*FROM buecher_titel bt.*`).
		WithArgs(pgxmock.AnyArg()). // arg for limit
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "cover_url", "isbn"}).
			AddRow("id-2", "Buch 2", "Autor 2", "http://cover.com/2", "456"))

	// ladeMeistgelesene (Beliebt)
	mock.ExpectQuery(`SELECT.*FROM ausleihen a.*`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()). // args for tage and limit
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "cover_url", "isbn"}).
			AddRow("id-3", "Buch 3", "Autor 3", "http://cover.com/3", "789"))

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/monitor/slides", nil)

	s.MonitorFolien()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var res repository.MonitorFolien
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.BuchDesMonats == nil || res.BuchDesMonats.Titel != "Buch 1" {
		t.Errorf("expected BuchDesMonats to be 'Buch 1', got %v", res.BuchDesMonats)
	}
	if len(res.NeuEingetroffen) != 1 || res.NeuEingetroffen[0].Titel != "Buch 2" {
		t.Errorf("expected NeuEingetroffen to have 1 item 'Buch 2', got %v", res.NeuEingetroffen)
	}
	if len(res.Beliebt) != 1 || res.Beliebt[0].Titel != "Buch 3" {
		t.Errorf("expected Beliebt to have 1 item 'Buch 3', got %v", res.Beliebt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestMonitorFolien_Fehler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Simulate DB error on the first query
	mock.ExpectQuery(`SELECT.*FROM ausleihen a.*`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()). // args for tage and limit
		WillReturnError(errors.New("db random error"))

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/monitor/slides", nil)

	s.MonitorFolien()(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
