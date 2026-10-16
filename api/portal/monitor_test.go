package portal

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"

	"github.com/pashagolub/pgxmock/v5"
)

// monitorAttrappe spielt die drei Abfragen der Folien in der Reihenfolge von
// repository.LadeFolien nach: Buch des Monats, Neu eingetroffen, Beliebt.
func monitorAttrappe(t *testing.T, meist, neu, beliebt []string) *Tueren {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(mock.Close)
	zeilen := func(titel []string) *pgxmock.Rows {
		z := pgxmock.NewRows([]string{"id", "titel", "autor", "cover_url", "isbn"})
		for _, ti := range titel {
			z.AddRow("id-"+ti, ti, "", "/uploads/"+ti+".jpg", "")
		}
		return z
	}
	// Die Meistgelesenen fragen nach Tagen und Anzahl, die Neuzugänge nach der Anzahl.
	mock.ExpectQuery(`FROM ausleihen a`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(zeilen(meist))
	mock.ExpectQuery(`FROM buecher_titel bt`).WithArgs(pgxmock.AnyArg()).WillReturnRows(zeilen(neu))
	mock.ExpectQuery(`FROM ausleihen a`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(zeilen(beliebt))
	return &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}
}

// Das Buch des Monats ist das meistgelesene der letzten Wochen; hat niemand gelesen, steht dort
// der jüngste Neuzugang (repository.LadeFolien). Die PG-Tests daneben prüfen, welche Titel auf
// den Monitor dürfen und wie gezählt wird, diese Auswahl nicht.
func TestMonitorFolien_BuchDesMonats(t *testing.T) {
	faelle := []struct {
		name                string
		meist, neu, beliebt []string
		erwartet            string
	}{
		{name: "das meistgelesene vor dem Neuzugang", meist: []string{"Gelesen"},
			neu: []string{"Neu"}, beliebt: []string{"Beliebt"}, erwartet: "Gelesen"},
		{name: "ohne Leser der jüngste Neuzugang", neu: []string{"Neu", "Älter"},
			erwartet: "Neu"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			monitorAttrappe(t, f.meist, f.neu, f.beliebt).MonitorFolien()(
				rec, httptest.NewRequest(http.MethodGet, "/api/monitor/slides", nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET /api/monitor/slides = %d: %s", rec.Code, rec.Body.String())
			}
			var antwort monitorAntwort
			if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
				t.Fatalf("Antwort unlesbar: %v", err)
			}
			if antwort.BuchDesMonats == nil || antwort.BuchDesMonats.Titel != f.erwartet {
				t.Errorf("Buch des Monats = %+v, erwartet %q", antwort.BuchDesMonats, f.erwartet)
			}
		})
	}
}

// Scheitert eine Abfrage, antwortet die Tür mit 500 und nicht mit leeren Folien, die nach
// „nichts gelesen, nichts neu" aussähen. Dass der Grund dabei nicht hinausgeht, hält
// apierrors für jede 500 (TestAntworteFehler_500IstNeutral).
func TestMonitorFolien_FehlerIst500(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(mock.Close)
	mock.ExpectQuery(`FROM ausleihen a`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errors.New("relation ausleihen gesperrt"))

	rec := httptest.NewRecorder()
	(&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}}).MonitorFolien()(
		rec, httptest.NewRequest(http.MethodGet, "/api/monitor/slides", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/monitor/slides = %d, erwartet 500", rec.Code)
	}
	// Sonst käme die 500 aus einer Abfrage, die die Attrappe gar nicht erwartet.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("nicht jede Abfrage gestellt: %v", err)
	}
}
