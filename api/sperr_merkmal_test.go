package api

// X-Sperre — das Merkmal, an dem der Sperr-Dialog der Theke wählt, was er anbietet:
// „uebergehbar" das einmalige Übergehen (Forderung, Überfällig-Automatik), „leser" das
// Aufheben der Sperre am Leser (seit dem 24.09.2026, service/sperr_merkmal.go; rot gesehen
// am Rückbau: sperrMerkmal ohne den Fall „leser" — der Header bleibt leer).
//
// Bis zum 13.09.2026 entschied das Frontend am Wortlaut der 403-Meldung. Die Schadens-
// Sperre („1 unbezahlte(r) Schadensfall/-fälle offen") traf keins der Stichwörter: kein
// Dialog, kein Override, obwohl der Server es erlaubt (am Stack nachgestellt). Der Body
// bleibt die eine kanonische Fehlerform {"error": …}; das Merkmal steht im Header.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

type sperrOmnibox struct{ fehler error }

func (o *sperrOmnibox) ProcessQuery(context.Context, service.OmniboxQuery) (*service.OmniboxResult, error) {
	return nil, o.fehler
}

func helferOhneRechte(t *testing.T) {
	t.Helper()
	InvalidatePermissionCache()
	t.Cleanup(InvalidatePermissionCache)
	permCacheMu.Lock()
	permCache["helfer:view_students"] = cacheEntry{Allowed: false, ExpiresAt: time.Now().Add(time.Minute)}
	permCache["helfer:edit_students"] = cacheEntry{Allowed: false, ExpiresAt: time.Now().Add(time.Minute)}
	permCacheMu.Unlock()
}

func aktionAnfrage(schluessel string) *http.Request {
	body := `{"query":"B-123","active_leser_id":"22222222-2222-2222-2222-222222222222","idempotency_key":"` + schluessel + `"}`
	req := httptest.NewRequest("POST", "/api/action", strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{Rolle: auth.Role("helfer"), UserID: "u1"}))
}

func TestSperrMerkmalImHeader(t *testing.T) {
	helferOhneRechte(t)
	grund := "Eltern zahlen nicht"

	faelle := []struct {
		name    string
		fehler  error
		merkmal string
	}{
		{"manuelle Sperre, Grund für Helfer gekürzt", &service.SperrGrundFehler{
			Kern:  service.SperreAmLeser(fmt.Errorf("%w: Manuelle Sperre", service.ErrBlocked)),
			Grund: grund,
		}, "leser"},
		{"Sperre der Ehemaligen, Grund für Helfer gekürzt", &service.SperrGrundFehler{
			Kern:  service.SperreAmLeser(service.ErrBlocked),
			Grund: grund,
		}, "leser"},
		{"offener Schaden", service.UebergehbareSperre(
			fmt.Errorf("%w: 1 unbezahlte(r) Schadensfall/-fälle offen", service.ErrBlocked)), "uebergehbar"},
		{"Gerät gesperrt (kein Override)", fmt.Errorf("%w: Gerät ist aktuell gesperrt", service.ErrBlocked), ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			(&Server{}).ActionHandler(&sperrOmnibox{fehler: f.fehler})(w, aktionAnfrage(""))
			if w.Code != http.StatusForbidden {
				t.Fatalf("Status %d, erwartet 403: %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("X-Sperre"); got != f.merkmal {
				t.Errorf("X-Sperre = %q, erwartet %q", got, f.merkmal)
			}
			if strings.Contains(w.Body.String(), grund) {
				t.Errorf("Sperrgrund erreicht die Helferin: %s", w.Body.String())
			}
		})
	}
}

// merkmalImCache prüft, dass das zwischengespeicherte Fehler-JSON das Merkmal trägt.
type merkmalImCache struct{ wert string }

func (m merkmalImCache) Match(v any) bool {
	roh, ok := v.([]byte)
	if !ok {
		return false
	}
	var daten map[string]string
	return json.Unmarshal(roh, &daten) == nil && daten["sperre"] == m.wert
}

// Eine wiederholte Anfrage mit demselben Idempotenz-Schlüssel kommt aus dem Cache. Ohne
// das Merkmal im Cache verlöre sie den Dialog.
func TestSperrMerkmalUeberlebtIdempotenzCache(t *testing.T) {
	helferOhneRechte(t)
	const schluessel = "sperr-merkmal-1"

	t.Run("erste Antwort legt das Merkmal ab", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		// Seit dem 15.09.2026 (Commit 7): Reservierung VOR der Arbeit, Antwort danach in die
		// Reservierung (repository/idempotenz.go).
		mock.ExpectQuery("INSERT INTO idempotency_keys").
			WithArgs(schluessel, repository.IdempotenzReservierungsfrist.Seconds()).
			WillReturnRows(pgxmock.NewRows([]string{"idempotency_key"}).AddRow(schluessel))
		mock.ExpectExec("UPDATE idempotency_keys SET response_data").
			WithArgs(schluessel, merkmalImCache{"uebergehbar"}, http.StatusForbidden).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		w := httptest.NewRecorder()
		s := &Server{DB: &db.Database{Pool: mock}}
		fehler := service.UebergehbareSperre(fmt.Errorf("%w: 1 unbezahlte(r) Schadensfall/-fälle offen", service.ErrBlocked))
		s.ActionHandler(&sperrOmnibox{fehler: fehler})(w, aktionAnfrage(schluessel))

		if got := w.Header().Get("X-Sperre"); got != "uebergehbar" {
			t.Errorf("X-Sperre = %q", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Cache-Eintrag ohne Merkmal: %v", err)
		}
	})

	t.Run("Wiederholung aus dem Cache setzt den Header", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery("INSERT INTO idempotency_keys").
			WithArgs(schluessel, repository.IdempotenzReservierungsfrist.Seconds()).WillReturnError(pgx.ErrNoRows) // der Schlüssel ist vergeben
		mock.ExpectQuery("SELECT response_data, status_code FROM idempotency_keys").
			WithArgs(schluessel).
			WillReturnRows(pgxmock.NewRows([]string{"response_data", "status_code"}).
				AddRow([]byte(`{"error":"ausleihe gesperrt: 1 unbezahlte(r) Schadensfall/-fälle offen","sperre":"uebergehbar"}`), http.StatusForbidden))

		w := httptest.NewRecorder()
		s := &Server{DB: &db.Database{Pool: mock}}
		s.ActionHandler(&sperrOmnibox{fehler: errors.New("darf nicht gerufen werden")})(w, aktionAnfrage(schluessel))

		if w.Code != http.StatusForbidden || w.Header().Get("X-Sperre") != "uebergehbar" {
			t.Errorf("Wiederholung: Status %d, X-Sperre %q", w.Code, w.Header().Get("X-Sperre"))
		}
	})

	t.Run("Sperre am Leser: Merkmal in den Cache und wieder heraus", func(t *testing.T) {
		const schluesselLeser = "sperr-merkmal-leser"
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery("INSERT INTO idempotency_keys").
			WithArgs(schluesselLeser, repository.IdempotenzReservierungsfrist.Seconds()).
			WillReturnRows(pgxmock.NewRows([]string{"idempotency_key"}).AddRow(schluesselLeser))
		mock.ExpectExec("UPDATE idempotency_keys SET response_data").
			WithArgs(schluesselLeser, merkmalImCache{"leser"}, http.StatusForbidden).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		w := httptest.NewRecorder()
		s := &Server{DB: &db.Database{Pool: mock}}
		fehler := &service.SperrGrundFehler{Kern: service.SperreAmLeser(fmt.Errorf("%w: Manuelle Sperre", service.ErrBlocked)), Grund: "x"}
		s.ActionHandler(&sperrOmnibox{fehler: fehler})(w, aktionAnfrage(schluesselLeser))
		if got := w.Header().Get("X-Sperre"); got != "leser" {
			t.Errorf("X-Sperre = %q, erwartet leser", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Cache-Eintrag ohne Merkmal: %v", err)
		}

		wieder, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer wieder.Close()
		wieder.ExpectQuery("INSERT INTO idempotency_keys").
			WithArgs(schluesselLeser, repository.IdempotenzReservierungsfrist.Seconds()).WillReturnError(pgx.ErrNoRows)
		wieder.ExpectQuery("SELECT response_data, status_code FROM idempotency_keys").
			WithArgs(schluesselLeser).
			WillReturnRows(pgxmock.NewRows([]string{"response_data", "status_code"}).
				AddRow([]byte(`{"error":"die ausleihe ist gesperrt: Manuelle Sperre — bitte an die Bibliotheksleitung wenden","sperre":"leser"}`), http.StatusForbidden))
		w = httptest.NewRecorder()
		(&Server{DB: &db.Database{Pool: wieder}}).ActionHandler(&sperrOmnibox{fehler: errors.New("darf nicht gerufen werden")})(w, aktionAnfrage(schluesselLeser))
		if w.Code != http.StatusForbidden || w.Header().Get("X-Sperre") != "leser" {
			t.Errorf("Wiederholung: Status %d, X-Sperre %q", w.Code, w.Header().Get("X-Sperre"))
		}
	})
}
