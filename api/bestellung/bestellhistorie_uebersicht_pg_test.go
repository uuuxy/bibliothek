package bestellung

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"github.com/pashagolub/pgxmock/v5"
)

func TestBestellhistorieUebersicht_FehlerLadeBestellKennzahlen(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock erstellen: %v", err)
	}
	defer mock.Close()

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	mock.ExpectQuery(`SELECT count\(\*\), coalesce\(sum\(b.gesamtbetrag\), 0\), coalesce\(sum\(b.anzahl_exemplare\), 0\).*`).
		WillReturnError(errors.New("datenbankfehler bei kennzahlen"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/uebersicht", nil)

	srv.BestellhistorieUebersicht().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestBestellhistorieUebersicht_FehlerKennzahlenJeTopf(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock erstellen: %v", err)
	}
	defer mock.Close()

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	// LadeBestellKennzahlen succeed
	mock.ExpectQuery(`SELECT count\(\*\), coalesce\(sum\(b.gesamtbetrag\), 0\), coalesce\(sum\(b.anzahl_exemplare\), 0\).*`).
		WillReturnRows(pgxmock.NewRows([]string{"count", "sum_gesamtbetrag", "sum_anzahl_exemplare", "offene_bestaetigungen"}).
			AddRow(10, 150.00, 50, 2))

	// LadeBestellKennzahlenJeTopf fail
	mock.ExpectQuery(`SELECT coalesce\(mittel, ''\), count\(\*\), coalesce\(sum\(gesamtbetrag\), 0\), coalesce\(sum\(anzahl_exemplare\), 0\).*`).
		WillReturnError(errors.New("datenbankfehler bei topfkennzahlen"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/uebersicht", nil)

	srv.BestellhistorieUebersicht().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestBestellhistorieUebersicht_Erfolgreich(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("mock erstellen: %v", err)
	}
	defer mock.Close()

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	// LadeBestellKennzahlen succeed
	mock.ExpectQuery(`SELECT count\(\*\), coalesce\(sum\(b.gesamtbetrag\), 0\), coalesce\(sum\(b.anzahl_exemplare\), 0\).*`).
		WillReturnRows(pgxmock.NewRows([]string{"count", "sum_gesamtbetrag", "sum_anzahl_exemplare", "offene_bestaetigungen"}).
			AddRow(10, 150.00, 50, 2))

	// LadeBestellKennzahlenJeTopf succeed
	mock.ExpectQuery(`SELECT coalesce\(mittel, ''\), count\(\*\), coalesce\(sum\(gesamtbetrag\), 0\), coalesce\(sum\(anzahl_exemplare\), 0\).*`).
		WillReturnRows(pgxmock.NewRows([]string{"mittel", "count", "sum_gesamtbetrag", "sum_anzahl_exemplare"}).
			AddRow("land", 4, 60.00, 20).
			AddRow("schultraeger", 6, 90.00, 30))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie/uebersicht", nil)

	srv.BestellhistorieUebersicht().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Code = %d, want %d", rec.Code, http.StatusOK)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
