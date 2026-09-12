package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// Kann die Datenbank bei der Token-Prüfung nicht antworten, ist das keine abgelaufene
// Sitzung. Bis zum 11.09.2026 antworteten Me und Refresh dann mit 401; der Client meldete
// daraufhin ab (authStore: Refresh-401 → handleLogout), und ein kurzer Datenbank-Aussetzer
// warf alle Arbeitsplätze hinaus. Die Anfrage bleibt verweigert, aber mit 503.
func TestTokenPruefungGestoert_Ist503(t *testing.T) {
	faelle := []struct {
		name    string
		ursache string
		mocks   func(pgxmock.PgxPoolIface)
	}{
		{"Sperrliste nicht erreichbar", "connection reset by peer", func(m pgxmock.PgxPoolIface) {
			m.ExpectQuery(`SELECT EXISTS`).WithArgs(pgxmock.AnyArg()).
				WillReturnError(errors.New("connection reset by peer"))
		}},
		{"Kontostatus nicht erreichbar", "context deadline exceeded", func(m pgxmock.PgxPoolIface) {
			expectNotBlacklisted(m)
			m.ExpectQuery(`SELECT aktiv, rolle FROM benutzer`).WithArgs(pgxmock.AnyArg()).
				WillReturnError(errors.New("context deadline exceeded"))
		}},
	}
	for _, f := range faelle {
		t.Run("Me/"+f.name, func(t *testing.T) {
			a, mock := newTestAuthenticator(t, 12*time.Hour)
			token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
			if err != nil {
				t.Fatal(err)
			}
			f.mocks(mock)
			rec := doMe(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("Status %d, want 503 — eine 401 meldet den Arbeitsplatz ab", rec.Code)
			}
			pruefeSentinelOhneUrsache(t, rec, f.ursache)
		})
		t.Run("Refresh/"+f.name, func(t *testing.T) {
			a, mock := newTestAuthenticator(t, 12*time.Hour)
			token, err := a.GenerateToken("user-1", "B-1", RoleAdmin)
			if err != nil {
				t.Fatal(err)
			}
			f.mocks(mock)
			rec := doRefresh(t, a, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("Status %d, want 503 — eine 401 meldet den Arbeitsplatz ab", rec.Code)
			}
			pruefeSentinelOhneUrsache(t, rec, f.ursache)
		})
	}
}

// pruefeSentinelOhneUrsache: Im 503-Body steht GENAU der Sentinel-Satz — nicht die
// gewrappte Ursache.
//
// Anlass (Rasterdurchgang 12.09.2026, Register „Der 503 trägt den rohen Datenbank-Text
// nach draußen"): SendHTTPError schickt bei allem außer 500 den Fehlertext an den
// Client und filtert nur, was istDatenbankFehler erkennt — SQL-Wortlaute und
// Constraint-Namen. „connection reset by peer" und „context deadline exceeded" sind
// keins davon, also las das Personal an der Theke „sitzung konnte nicht geprüft werden,
// bitte erneut versuchen: sperrliste: connection reset by peer". Kein Schema und keine
// Daten, aber Betriebsinnenleben in einer Meldung, die für die Bibliothek gedacht ist.
//
// Die Ursache bleibt im Server-Log; der Client bekommt den Satz, der ihm gilt.
func pruefeSentinelOhneUrsache(t *testing.T, rec *httptest.ResponseRecorder, ursache string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("503-Body ist kein JSON: %q (%v)", rec.Body.String(), err)
	}
	msg := body["error"]
	if msg != ErrPruefungGestoert.Error() {
		t.Errorf("503-Meldung %q, erwartet genau %q", msg, ErrPruefungGestoert.Error())
	}
	// Die Einzelteile der Wrapping-Kette je einzeln — ein gekürzter Text würde die
	// Gleichheitsprüfung oben zwar auch rot machen, aber nicht sagen, WAS durchkam.
	for _, leck := range []string{ursache, "sperrliste", "kontostatus"} {
		if strings.Contains(strings.ToLower(msg), leck) {
			t.Errorf("Betriebsinnenleben im 503-Body: %q steht in %q", leck, msg)
		}
	}
}
