package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

func TestResolveBestandsFilter(t *testing.T) {
	cases := []struct {
		in, wantFragment, wantName string
	}{
		{"lmf", "AND LOWER(t.titel) LIKE 'lmf-%'", "lmf"},
		{"freihand", "AND LOWER(t.titel) NOT LIKE 'lmf-%'", "freihand"},
		{"", "", "alle"},
		{"kaputt", "", "alle"}, // unbekannte Werte fallen sicher auf Gesamtbestand zurück
	}
	for _, c := range cases {
		frag, name := resolveBestandsFilter(c.in)
		if frag != c.wantFragment || name != c.wantName {
			t.Errorf("resolveBestandsFilter(%q) = (%q, %q), want (%q, %q)", c.in, frag, name, c.wantFragment, c.wantName)
		}
	}
}

// Der LMF-Filter muss in ALLEN drei Statistik-Queries ankommen (Renner,
// Ladenhüter, Kennzahlen) — pgxmock matcht per Regex, ein fehlendes
// Filter-Fragment ließe die Erwartung fehlschlagen.
func TestGetStatistics_TypeFilterReachesAllQueries(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()
	server := &Server{DB: &db.Database{Pool: mock}}

	mock.ExpectQuery(`COUNT\(a\.id\) AS count[\s\S]*LIKE 'lmf-%'`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "cover_url", "subject", "signatur", "erscheinungsjahr", "count"}).
			AddRow("t1", "LMF-Mathe 7", "Verlag", "", "Mathematik", "Ma 7", 2023, 42))
	mock.ExpectQuery(`MAX\(a\.ausgeliehen_am\) AS last_loan[\s\S]*LIKE 'lmf-%'`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "subject", "signatur", "erscheinungsjahr", "last_loan"}).
			AddRow("sw1", "LMF-Physik 9", "", "978-1", "Physik", "Ph 9", 2019, nil))
	mock.ExpectQuery(`wiederbeschaffung[\s\S]*LIKE 'lmf-%'`).
		WillReturnRows(pgxmock.NewRows([]string{"gesamt", "aktiv", "verliehen", "verlorene", "wiederbeschaffung", "verlust_quote", "zirkulationsquote"}).
			AddRow(200, 190, 57, 4, 129.90, 2.0, 30.0))

	req := httptest.NewRequest(http.MethodGet, "/api/statistiken?type=lmf", nil)
	rec := httptest.NewRecorder()
	server.GetStatisticsHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d: %s", rec.Code, rec.Body.String())
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
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Antwort enthält %s nicht: %s", want, body)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Filter kam nicht in allen Queries an: %v", err)
	}
}
