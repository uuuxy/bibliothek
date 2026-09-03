package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v4"
)

func TestResolveBestandsFilter(t *testing.T) {
	cases := []struct {
		in, wantFragment, wantName string
	}{
		// Seit Migration 093 ist das Prädikat die Spalte selbst — kein abgeleitetes
		// Textmuster mehr, das Go und SQL getrennt formulieren könnten.
		{"lmf", "AND t.ist_lernmittel", "lmf"},
		{"freihand", "AND NOT t.ist_lernmittel", "freihand"},
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

	mock.ExpectQuery(`COUNT\(a\.id\) AS count[\s\S]*AND t\.ist_lernmittel`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "cover_url", "subject", "signatur", "erscheinungsjahr", "count"}).
			AddRow("t1", "LMF-Mathe 7", "Verlag", "", "Mathematik", "Ma 7", 2023, 42))
	mock.ExpectQuery(`MAX\(a\.ausgeliehen_am\) AS last_loan[\s\S]*AND t\.ist_lernmittel`).
		WillReturnRows(pgxmock.NewRows([]string{"id", "titel", "autor", "isbn", "subject", "signatur", "erscheinungsjahr", "last_loan"}).
			AddRow("sw1", "LMF-Physik 9", "", "978-1", "Physik", "Ph 9", 2019, nil))
	mock.ExpectQuery(`aktive_ausleihen AS \([\s\S]*wiederbeschaffung[\s\S]*AND t\.ist_lernmittel`).
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

// Das Schuljahr beginnt am 1. August — Juli zählt zum vorigen (Rasterdurchgang 02.09.2026:
// vorher rechnete CURRENT_DATE in der DB-Sitzungszone UTC).
func TestSchuljahresBeginn(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	faelle := map[time.Time]string{
		time.Date(2026, 8, 1, 0, 30, 0, 0, berlin):  "2026-08-01",
		time.Date(2026, 7, 31, 23, 0, 0, 0, berlin): "2025-08-01",
		time.Date(2027, 1, 1, 0, 30, 0, 0, berlin):  "2026-08-01",
	}
	for zeit, soll := range faelle {
		if got := schuljahresBeginn(zeit).Format("2006-01-02"); got != soll {
			t.Errorf("%s → %s, erwartet %s", zeit, got, soll)
		}
	}
	if f := resolveZeitraumFilter("schuljahr"); !strings.Contains(f, "'::date") || strings.Contains(f, "CURRENT_DATE") {
		t.Errorf("Schuljahr-Filter muss ein festes Datum aus der Schulzeitzone tragen: %q", f)
	}
}
