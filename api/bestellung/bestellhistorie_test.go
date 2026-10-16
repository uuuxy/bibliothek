package bestellung

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v5"
)

func TestBestellhistorie(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock init: %v", err)
	}
	defer mock.Close()

	srv := &Tueren{
		Basis: &tuer.Basis{
			DB: &db.Verbindung{Pool: mock},
		},
	}

	bestelldatum := time.Now()

	tests := []struct {
		name           string
		query          string
		setupMock      func()
		expectedStatus int
		expectedLen    int
	}{
		{
			name:  "leere Historie",
			query: "",
			setupMock: func() {
				mock.ExpectQuery(`SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum`).
					WithArgs(200).
					WillReturnRows(pgxmock.NewRows([]string{"id", "lieferant_name", "lieferant_email", "kundennummer", "bestelldatum", "gesamtbetrag", "anzahl_exemplare", "mit_bestaetigung", "bestaetigt_am", "etiketten_groesse", "bestaetigt_durch", "link_aktiv", "link_gueltig_bis", "mittel", "mail_gescheitert_am"}))
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name:  "mit Limit und Mittel",
			query: "?limit=10&mittel=land",
			setupMock: func() {
				mock.ExpectQuery(`SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum`).
					WithArgs(10, "land").
					WillReturnRows(pgxmock.NewRows([]string{"id", "lieferant_name", "lieferant_email", "kundennummer", "bestelldatum", "gesamtbetrag", "anzahl_exemplare", "mit_bestaetigung", "bestaetigt_am", "etiketten_groesse", "bestaetigt_durch", "link_aktiv", "link_gueltig_bis", "mittel", "mail_gescheitert_am"}).
						AddRow("1", "Lieferant", "test@test.de", "KUNDE-1", bestelldatum, 10.0, 1, false, nil, nil, nil, false, nil, "land", nil))
				mock.ExpectQuery(`SELECT p.bestellung_id, p.titel_name, p.isbn, p.menge, p.einzelpreis`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(pgxmock.NewRows([]string{"bestellung_id", "titel_name", "isbn", "menge", "einzelpreis", "titel_id", "etiketten_offen"}).
						AddRow("1", "Buch", "123", 1, 10.0, "T1", 1))
			},
			expectedStatus: http.StatusOK,
			expectedLen:    1,
		},
		{
			name:  "Limit überschreitet Maximum",
			query: "?limit=9999",
			setupMock: func() {
				mock.ExpectQuery(`SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum`).
					WithArgs(500).
					WillReturnRows(pgxmock.NewRows([]string{"id", "lieferant_name", "lieferant_email", "kundennummer", "bestelldatum", "gesamtbetrag", "anzahl_exemplare", "mit_bestaetigung", "bestaetigt_am", "etiketten_groesse", "bestaetigt_durch", "link_aktiv", "link_gueltig_bis", "mittel", "mail_gescheitert_am"}))
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name:  "ungültiger Mittel-Filter",
			query: "?mittel=ungueltig",
			setupMock: func() {
				// Wird vor der DB abgefangen
			},
			expectedStatus: http.StatusBadRequest,
			expectedLen:    0, // Wird ignoriert
		},
		{
			name:  "Fehler bei LadeBestellVerlauf",
			query: "",
			setupMock: func() {
				mock.ExpectQuery(`SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum`).
					WithArgs(200).
					WillReturnError(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedLen:    0,
		},
		{
			name:  "Fehler bei LadeBestellVerlaufPositionen",
			query: "",
			setupMock: func() {
				mock.ExpectQuery(`SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum`).
					WithArgs(200).
					WillReturnRows(pgxmock.NewRows([]string{"id", "lieferant_name", "lieferant_email", "kundennummer", "bestelldatum", "gesamtbetrag", "anzahl_exemplare", "mit_bestaetigung", "bestaetigt_am", "etiketten_groesse", "bestaetigt_durch", "link_aktiv", "link_gueltig_bis", "mittel", "mail_gescheitert_am"}).
						AddRow("1", "Lieferant", "test@test.de", "KUNDE-1", bestelldatum, 10.0, 1, false, nil, nil, nil, false, nil, "", nil))
				mock.ExpectQuery(`SELECT p.bestellung_id, p.titel_name, p.isbn, p.menge, p.einzelpreis`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnError(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/api/bestellhistorie"+tt.query, nil)
			req = req.WithContext(context.Background())
			rec := httptest.NewRecorder()

			srv.Bestellhistorie().ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var result []repository.BestellVerlaufEintrag
				if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
					t.Fatalf("JSON decode error: %v", err)
				}
				if len(result) != tt.expectedLen {
					t.Errorf("len %d, want %d", len(result), tt.expectedLen)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}
