package katalog

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v5"

	"bibliothek/api/tuer"
	"bibliothek/db"
)

// Scheitert das Lesen, nennt die Zeile zum 500 im Log die Ursache aus der Datenbank. Ein fester
// Text an ihrer Stelle („database error“) ließ den Betrieb raten, was ausgefallen war.
func TestNachschlagelisten_500NenntUrsacheImLog(t *testing.T) {
	for _, fall := range []struct {
		name string
		tuer func(*Tueren) http.HandlerFunc
	}{
		{"Systematik", (*Tueren).Systematik},
		{"Faecher", (*Tueren).Faecher},
	} {
		t.Run(fall.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()
			mock.ExpectQuery(".").WillReturnError(errors.New("verbindung weg"))
			var protokoll bytes.Buffer
			vorher := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&protokoll, nil)))
			t.Cleanup(func() { slog.SetDefault(vorher) })

			rec := httptest.NewRecorder()
			fall.tuer(&Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: mock}}})(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("Status %d, erwartet 500", rec.Code)
			}
			if !strings.Contains(protokoll.String(), "verbindung weg") {
				t.Errorf("Log ohne Ursache: %s", protokoll.String())
			}
			if strings.Contains(rec.Body.String(), "verbindung weg") {
				t.Errorf("Ursache geht hinaus: %s", rec.Body.String())
			}
		})
	}
}
