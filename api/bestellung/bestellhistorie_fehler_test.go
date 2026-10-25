package bestellung

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v5"
)

// Scheitert eine Abfrage der Bestellhistorie, antwortet die Tür mit 500 und nicht mit dem, was
// bis dahin gelesen ist. Die Übersicht trägt Summen über alle Bestellungen: Eine Null aus einer
// gescheiterten Abfrage sähe aus wie „nichts bestellt" (bestellhistorieUebersicht). Die Liste
// ohne ihre Positionen sähe aus wie Bestellungen ohne Titel. Die PG-Tests daneben gehen nur über
// gelungene Abfragen.
func TestBestellhistorie_GescheiterteAbfrageIst500(t *testing.T) {
	const (
		kennzahlen = `SELECT count\(\*\), coalesce\(sum\(b\.gesamtbetrag\), 0\)`
		jeTopf     = `SELECT coalesce\(mittel, ''\), count\(\*\)`
		verlauf    = `SELECT b\.id, b\.lieferant_name`
		positionen = `SELECT p\.bestellung_id, p\.titel_name`
	)
	kaputt := errors.New("verbindung abgebrochen")

	faelle := []struct {
		name    string
		tuer    func(*Tueren) http.HandlerFunc
		ziel    string
		erwarte func(pgxmock.PgxPoolIface)
		// folgt beantwortet die Abfrage, die eine Tür als nächste stellte, wenn sie den Fehler
		// verschluckte. Ohne Antwort scheiterte auch sie, und die 500 käme von dort.
		folgt func(pgxmock.PgxPoolIface)
	}{
		{
			name: "Übersicht, Summen",
			tuer: (*Tueren).BestellhistorieUebersicht,
			ziel: "/api/bestellhistorie/uebersicht",
			erwarte: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(kennzahlen).WillReturnError(kaputt)
			},
			folgt: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(jeTopf).WillReturnRows(pgxmock.NewRows(
					[]string{"mittel", "gesamt", "gesamtbetrag", "exemplare"}).AddRow("land", 1, "5.00", 1))
			},
		},
		{
			name: "Übersicht, Summen je Topf",
			tuer: (*Tueren).BestellhistorieUebersicht,
			ziel: "/api/bestellhistorie/uebersicht",
			erwarte: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(kennzahlen).WillReturnRows(pgxmock.NewRows(
					[]string{"gesamt", "gesamtbetrag", "exemplare", "offen"}).AddRow(10, "150.00", 50, 2))
				m.ExpectQuery(jeTopf).WillReturnError(kaputt)
			},
		},
		{
			name: "Liste",
			tuer: (*Tueren).Bestellhistorie,
			ziel: "/api/bestellhistorie",
			erwarte: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(verlauf).WithArgs(bestellhistorieVorgabeGrenze).WillReturnError(kaputt)
			},
		},
		{
			name: "Positionen der Liste",
			tuer: (*Tueren).Bestellhistorie,
			ziel: "/api/bestellhistorie",
			erwarte: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(verlauf).WithArgs(bestellhistorieVorgabeGrenze).WillReturnRows(pgxmock.NewRows([]string{
					"id", "lieferant_name", "lieferant_email", "kundennummer", "bestelldatum",
					"gesamtbetrag", "anzahl_exemplare", "mit_bestaetigung", "bestaetigt_am",
					"etiketten_groesse", "bestaetigt_durch", "link_aktiv", "link_gueltig_bis",
					"mittel", "mail_gescheitert_am",
				}).AddRow("b1", "Lieferant", "bestellung@example.org", "K-1", time.Now(),
					"10.00", 1, false, nil, nil, nil, false, nil, "", nil))
				m.ExpectQuery(positionen).WithArgs(pgxmock.AnyArg()).WillReturnError(kaputt)
			},
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool: %v", err)
			}
			t.Cleanup(mock.Close)
			f.erwarte(mock)
			if f.folgt != nil {
				f.folgt(mock)
			}

			srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
			rec := httptest.NewRecorder()
			f.tuer(srv)(rec, httptest.NewRequest(http.MethodGet, f.ziel, nil))

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("GET %s = %d, erwartet 500", f.ziel, rec.Code)
			}
			// Sonst wäre der Fall womöglich an einer früheren Abfrage gescheitert als an der,
			// um die es geht. Eine beantwortete Folgeabfrage bleibt dagegen offen.
			if err := mock.ExpectationsWereMet(); (err != nil) != (f.folgt != nil) {
				t.Errorf("GET %s: gestellte Abfragen passen nicht zum Fall: %v", f.ziel, err)
			}
		})
	}
}
