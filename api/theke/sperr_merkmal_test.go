package theke

// X-Sperre ist das Merkmal, an dem der Sperr-Dialog der Theke wählt, was er anbietet:
// „uebergehbar" das einmalige Übergehen (Forderung, Überfällig-Automatik), „leser" das
// Aufheben der Sperre am Leser (internal/service/sperr_merkmal.go; rot gesehen am Rückbau:
// sperrMerkmal ohne den Fall „leser", der Kopf bleibt leer).
//
// Entschiede die Oberfläche am Wortlaut der 403-Meldung, träfe die Schadens-Sperre („1
// unbezahlte(r) Schadensfall/-fälle offen") keins der Stichwörter: kein Dialog, kein
// Übergehen, obwohl der Server es erlaubt. Der Rumpf bleibt die eine Fehlerform
// {"error": …}; das Merkmal steht im Kopf.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

type sperrOmnibox struct{ fehler error }

func (o *sperrOmnibox) VerarbeiteEingabe(context.Context, service.OmniboxEingabe) (*service.OmniboxErgebnis, error) {
	return nil, o.fehler
}

// helferOhneRechte lässt b dem Helfer Schülerdaten verweigern, als hätte die Tabelle der Rechte
// es geliefert, und liefert b.
func helferOhneRechte(b *tuer.Basis) *tuer.Basis {
	b.Rechte.MerkeFuerTest("helfer", "view_students", false)
	b.Rechte.MerkeFuerTest("helfer", "edit_students", false)
	return b
}

func scanAnfrage(schluessel string) *http.Request {
	body := `{"query":"B-123","active_leser_id":"22222222-2222-2222-2222-222222222222","idempotency_key":"` + schluessel + `"}`
	req := httptest.NewRequest("POST", "/api/action", strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel,
		&auth.Claims{Rolle: auth.Rolle("helfer"), KontoID: "u1"}))
}

func TestSperrMerkmalImHeader(t *testing.T) {
	grund := "Eltern zahlen nicht"

	// meldung: Die Theke zeigt sie als Satz, groß begonnen, mit dem allgemeinen Grund; der
	// Fehlertext bleibt klein.
	faelle := []struct {
		name    string
		fehler  error
		merkmal string
		meldung string
	}{
		{"manuelle Sperre, Grund für Helfer gekürzt", &service.SperrgrundFehler{
			Kern:  service.SperreAmLeser(service.Sperre("Manuelle Sperre")),
			Grund: grund,
		}, "leser", "Die Ausleihe ist gesperrt: Manuelle Sperre. Bitte an die Bibliotheksleitung wenden."},
		{"Sperre der Ehemaligen, Grund für Helfer gekürzt", &service.SperrgrundFehler{
			Kern:  service.SperreAmLeser(service.ErrGesperrt),
			Grund: grund,
		}, "leser", "Die Ausleihe ist gesperrt. Bitte an die Bibliotheksleitung wenden."},
		{"offener Schaden", service.UebergehbareSperre(service.Sperre("1 unbezahlte(r) Schadensfall/-fälle offen")),
			"uebergehbar", "Die Ausleihe ist gesperrt: 1 unbezahlte(r) Schadensfall/-fälle offen"},
		{"Gerät gesperrt (kein Override)", service.Sperre("Gerät ist aktuell gesperrt"), "",
			"Die Ausleihe ist gesperrt: Gerät ist aktuell gesperrt"},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			(&Tueren{Basis: helferOhneRechte(&tuer.Basis{})}).EingabeVerarbeiten(&sperrOmnibox{fehler: f.fehler})(w, scanAnfrage(""))
			if w.Code != http.StatusForbidden {
				t.Fatalf("Status %d, erwartet 403: %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("X-Sperre"); got != f.merkmal {
				t.Errorf("X-Sperre = %q, erwartet %q", got, f.merkmal)
			}
			if strings.Contains(w.Body.String(), grund) {
				t.Errorf("Sperrgrund erreicht die Helferin: %s", w.Body.String())
			}
			var rumpf struct {
				Fehler string `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &rumpf); err != nil || rumpf.Fehler != f.meldung {
				t.Errorf("Meldung %q, erwartet %q (%v)", rumpf.Fehler, f.meldung, err)
			}
		})
	}
}

// merkmalInDerAblage prüft, dass das zwischengespeicherte Fehler-JSON das Merkmal trägt.
type merkmalInDerAblage struct{ wert string }

func (m merkmalInDerAblage) Match(v any) bool {
	roh, ok := v.([]byte)
	if !ok {
		return false
	}
	var daten map[string]string
	return json.Unmarshal(roh, &daten) == nil && daten["sperre"] == m.wert
}

// Eine wiederholte Anfrage mit demselben Idempotenz-Schlüssel kommt aus dem Cache. Ohne
// das Merkmal im Cache verlöre sie den Dialog.
func TestSperrMerkmalUeberlebtIdempotenzAblage(t *testing.T) {
	const schluessel = "sperr-merkmal-1"

	t.Run("erste Antwort legt das Merkmal ab", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		// Reservierung vor der Arbeit, die Antwort danach in die Reservierung
		// (repository/idempotenz.go).
		mock.ExpectQuery("INSERT INTO idempotency_keys").
			WithArgs(schluessel, repository.IdempotenzReservierungsfrist.Seconds()).
			WillReturnRows(pgxmock.NewRows([]string{"idempotency_key"}).AddRow(schluessel))
		mock.ExpectExec("UPDATE idempotency_keys SET response_data").
			WithArgs(schluessel, merkmalInDerAblage{"uebergehbar"}, http.StatusForbidden).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		w := httptest.NewRecorder()
		s := &Tueren{Basis: helferOhneRechte(&tuer.Basis{DB: &db.Verbindung{Pool: mock}})}
		fehler := service.UebergehbareSperre(service.Sperre("1 unbezahlte(r) Schadensfall/-fälle offen"))
		s.EingabeVerarbeiten(&sperrOmnibox{fehler: fehler})(w, scanAnfrage(schluessel))

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
		s := &Tueren{Basis: helferOhneRechte(&tuer.Basis{DB: &db.Verbindung{Pool: mock}})}
		s.EingabeVerarbeiten(&sperrOmnibox{fehler: errors.New("darf nicht gerufen werden")})(w, scanAnfrage(schluessel))

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
			WithArgs(schluesselLeser, merkmalInDerAblage{"leser"}, http.StatusForbidden).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		w := httptest.NewRecorder()
		s := &Tueren{Basis: helferOhneRechte(&tuer.Basis{DB: &db.Verbindung{Pool: mock}})}
		fehler := &service.SperrgrundFehler{Kern: service.SperreAmLeser(service.Sperre("Manuelle Sperre")), Grund: "x"}
		s.EingabeVerarbeiten(&sperrOmnibox{fehler: fehler})(w, scanAnfrage(schluesselLeser))
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
		(&Tueren{Basis: helferOhneRechte(&tuer.Basis{DB: &db.Verbindung{Pool: wieder}})}).EingabeVerarbeiten(&sperrOmnibox{fehler: errors.New("darf nicht gerufen werden")})(w, scanAnfrage(schluesselLeser))
		if w.Code != http.StatusForbidden || w.Header().Get("X-Sperre") != "leser" {
			t.Errorf("Wiederholung: Status %d, X-Sperre %q", w.Code, w.Header().Get("X-Sperre"))
		}
	})
}
