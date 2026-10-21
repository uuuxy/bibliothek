package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
)

// Kann die Datenbank bei der Prüfung des Tokens nicht antworten, ist das keine abgelaufene
// Sitzung. Auf 401 meldet die Oberfläche ab (authStore: 401 der Verlängerung führt zur
// Abmeldung), und ein kurzer Aussetzer der Datenbank würfe alle Arbeitsplätze hinaus. Die
// Anfrage bleibt verweigert, aber mit 503.
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
			erwarteNichtWiderrufen(m)
			m.ExpectQuery(`SELECT aktiv, rolle FROM benutzer`).WithArgs(pgxmock.AnyArg()).
				WillReturnError(errors.New("context deadline exceeded"))
		}},
	}
	for _, f := range faelle {
		t.Run("Me/"+f.name, func(t *testing.T) {
			a, mock := newTestAnmeldung(t, 12*time.Hour)
			token, err := a.ErzeugeToken("user-1", RolleAdmin, "")
			if err != nil {
				t.Fatal(err)
			}
			f.mocks(mock)
			rec := rufeAngemeldetesKonto(t, a, mock, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("AngemeldetesKonto: Status %d, erwartet 503 — eine 401 meldet den Arbeitsplatz ab", rec.Code)
			}
			pruefeFehlerwertOhneUrsache(t, rec, f.ursache)
		})
		t.Run("Refresh/"+f.name, func(t *testing.T) {
			a, mock := newTestAnmeldung(t, 12*time.Hour)
			token, err := a.ErzeugeToken("user-1", RolleAdmin, "")
			if err != nil {
				t.Fatal(err)
			}
			f.mocks(mock)
			rec := rufeSitzungVerlaengern(t, a, &http.Cookie{Name: "session_token", Value: token})
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("SitzungVerlaengern: Status %d, erwartet 503 — eine 401 meldet den Arbeitsplatz ab", rec.Code)
			}
			pruefeFehlerwertOhneUrsache(t, rec, f.ursache)
		})
	}
}

// pruefeFehlerwertOhneUrsache prüft, dass im Rumpf der 503 genau der Satz des Fehlerwerts
// steht, nicht die eingewickelte Ursache.
//
// AntworteFehler schickt bei allem außer 500 den Text des Fehlers an die Oberfläche und filtert
// nur, was istDatenbankFehler erkennt: Wortlaute von SQL und Namen von Constraints. „connection
// reset by peer" und „context deadline exceeded" sind keins davon; an der Theke stünde sonst
// „sitzung konnte nicht geprüft werden, bitte erneut versuchen: sperrliste: connection reset by
// peer". Das ist weder Schema noch Daten, aber Innenleben des Betriebs in einer Meldung für die
// Bibliothek.
//
// Die Ursache bleibt im Log des Servers; die Oberfläche bekommt den Satz, der ihr gilt.
func pruefeFehlerwertOhneUrsache(t *testing.T, rec *httptest.ResponseRecorder, ursache string) {
	t.Helper()
	var rumpf map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &rumpf); err != nil {
		t.Fatalf("503-Body ist kein JSON: %q (%v)", rec.Body.String(), err)
	}
	msg := rumpf["error"]
	if want := "Die Anmeldung ließ sich nicht prüfen. Bitte erneut versuchen."; msg != want {
		t.Errorf("503-Meldung %q, erwartet genau %q", msg, want)
	}
	// Die Glieder der eingewickelten Kette je einzeln: Ein gekürzter Text machte den Vergleich oben
	// zwar auch rot, sagte aber nicht, was durchkam.
	for _, leck := range []string{ursache, "sperrliste", "kontostatus"} {
		if strings.Contains(strings.ToLower(msg), leck) {
			t.Errorf("Betriebsinnenleben im 503-Body: %q steht in %q", leck, msg)
		}
	}
}
