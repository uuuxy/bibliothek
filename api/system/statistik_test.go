package system

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"

	"github.com/google/go-cmp/cmp"
	"github.com/pashagolub/pgxmock/v5"
)

// Der LMF-Filter muss in allen vier Abfragen der Statistik ankommen (Renner, Ladenhüter,
// Kennzahlen, Monatstrend): pgxmock vergleicht per Muster, ein fehlendes Stück des Filters
// ließe die Erwartung fehlschlagen.
func TestStatistik_BestandsFilterErreichtJedeAbfrage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

	mock.ExpectQuery(`COUNT\(a\.id\) AS count[\s\S]*AND t\.ist_lernmittel`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "cover_url", "subject", "signatur", "erscheinungsjahr", "count"}).
			AddRow("t1", "LMF-Mathe 7", "Verlag", "978-3", "", "Mathematik", "Ma 7", 2023, 42))
	mock.ExpectQuery(`MAX\(a\.ausgeliehen_am\) AS last_loan[\s\S]*AND t\.ist_lernmittel`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "subject", "signatur", "erscheinungsjahr", "last_loan"}).
			AddRow("sw1", "LMF-Physik 9", "", "978-1", "Physik", "Ph 9", 2019, nil))
	mock.ExpectQuery(`aktive_ausleihen AS \([\s\S]*wiederbeschaffung[\s\S]*AND t\.ist_lernmittel`).
		WillReturnRows(pgxmock.NewRows([]string{"gesamt", "aktiv", "verliehen", "verlorene", "wiederbeschaffung", "verlust_quote", "zirkulationsquote"}).
			AddRow(200, 190, 57, 4, 129.90, 2.0, 30.0))
	mock.ExpectQuery(`WITH monate AS \([\s\S]*AND t\.ist_lernmittel`).
		WillReturnRows(pgxmock.NewRows([]string{"monat", "ausleihen", "rueckgaben"}).AddRow("2026-10", 3, 1))

	req := httptest.NewRequest(http.MethodGet, "/api/statistiken?type=lmf", nil)
	rec := httptest.NewRecorder()
	server.Statistik()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`"filter_type":"lmf"`,
		`"wiederbeschaffungswert_defekt":129.9`,
		`"zirkulationsquote":30`,
		`"fachbereich":"Mathematik"`,
		`"systematik":"Ph 9"`,
		`"erscheinungsjahr":2023`,
		`"letzte_aus":"Nie ausgeliehen"`,
		`"aktuell_verliehen":57`,
		`"nicht_gelesen":[]`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Antwort enthält %s nicht: %s", want, body)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Filter kam nicht in allen Queries an: %v", err)
	}
}

// Ein Abschnitt, der nicht gelesen wurde, steht in nicht_gelesen. Sonst sähe die Seite einen
// leeren Abschnitt und zeigte den Leerzustand, bei den Ladenhütern „Keine Ladenhüter" in Grün.
// Die Abfragen danach werden beantwortet: Die Tür liest weiter, statt abzubrechen.
func TestStatistik_LesefehlerNenntDenAbschnitt(t *testing.T) {
	for _, scheitert := range []string{"renner", "ladenhueter", "kennzahlen", "monats_trend"} {
		t.Run(scheitert, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()
			server := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}

			abfragen := []struct {
				abschnitt string
				muster    string
				zeilen    *pgxmock.Rows
			}{
				{"renner", `COUNT\(a\.id\) AS count`, pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "cover_url", "subject", "signatur", "erscheinungsjahr", "count"}).
					AddRow("t1", "Momo", "Ende", "978-3", "", "Deutsch", "De 1", 2001, 7)},
				{"ladenhueter", `MAX\(a\.ausgeliehen_am\) AS last_loan`, pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "subject", "signatur", "erscheinungsjahr", "last_loan"}).
					AddRow("t2", "Krabat", "Preußler", "", "Deutsch", "De 2", 1971, nil)},
				{"kennzahlen", `aktive_ausleihen AS \(`, pgxmock.NewRows([]string{"gesamt", "aktiv", "verliehen", "verlorene", "wiederbeschaffung", "verlust_quote", "zirkulationsquote"}).
					AddRow(10, 9, 2, 0, 0.0, 0.0, 22.2)},
				{"monats_trend", `WITH monate AS \(`, pgxmock.NewRows([]string{"monat", "ausleihen", "rueckgaben"}).AddRow("2026-10", 3, 1)},
			}
			for _, a := range abfragen {
				erwartung := mock.ExpectQuery(a.muster)
				if a.abschnitt == scheitert {
					erwartung.WillReturnError(errors.New("verbindung abgebrochen"))
				} else {
					erwartung.WillReturnRows(a.zeilen)
				}
			}

			rec := httptest.NewRecorder()
			server.Statistik()(rec, httptest.NewRequest(http.MethodGet, "/api/statistiken", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
			}
			var antwort struct {
				NichtGelesen []string `json:"nicht_gelesen"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v", err)
			}
			if diff := cmp.Diff([]string{scheitert}, antwort.NichtGelesen); diff != "" {
				t.Errorf("nicht_gelesen (-erwartet +erhalten):\n%s", diff)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Die Tür hat nach dem Fehler nicht weitergelesen: %v", err)
			}
		})
	}
}
