package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/repository"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
)

// Gerät die Bearbeitung einer Anfrage in Panik, antwortet der Server mit einem neutralen 500
// und läuft weiter; ist Sentry eingerichtet, kommt die Meldung dort an. Beides hängt an den
// zwei äußeren Schichten des Routers (PanicRecoveryMiddleware, darunter die Schicht von Sentry
// mit Repanic) und an der Sentry-Bibliothek, die bei jeder Anfrage mitläuft, auch ohne
// SENTRY_DSN.
//
// Über den ganzen Router mit Sitzung: Die Schichten hängen am Router, nicht an der Tür.

const panikText = "Probe-Panik aus dem Test"

// poolMitPanik gerät in Panik, sobald die Tür das Design der Ausweise liest und scharf gesetzt
// ist. Sitzung und Rechte laufen am echten Pool.
type poolMitPanik struct {
	db.PgxPoolIface
	scharf *bool
}

func (p poolMitPanik) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if *p.scharf && strings.Contains(sql, "FROM system_einstellungen") && len(args) > 0 && args[0] == repository.AusweisLayoutSchluessel {
		panic(panikText)
	}
	return p.PgxPoolIface.QueryRow(ctx, sql, args...)
}

// anfrageInPanik schickt dieselbe Anfrage zweimal über den ganzen Router: erst ungestört,
// dann mit einem Pool, der beim Lesen der Tür in Panik gerät. Zurück kommt die Antwort auf
// die zweite.
func anfrageInPanik(t *testing.T, konto string) *httptest.ResponseRecorder {
	t.Helper()
	pool := pgtest.Pool(t)
	scharf := false
	_, sitzung, router := routerMitSitzungUeber(t, pool, poolMitPanik{PgxPoolIface: pool, scharf: &scharf},
		konto, "Paula", "Panik")
	t.Cleanup(func() { pgtest.Aufraeumen(t, pool, `DELETE FROM benutzer WHERE email = $1`, konto) })
	rufe := func() (rec *httptest.ResponseRecorder) {
		t.Helper()
		// Ohne PanicRecoveryMiddleware erreichte die Panik den Aufrufer; der Test soll das
		// melden und nicht mit ihr enden.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("die Panik erreichte den Aufrufer des Routers: %v", r)
			}
		}()
		req := httptest.NewRequest(http.MethodGet, "/api/ausweis-layout", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	// Gegenprobe: Ungestört antwortet die Tür ohne Fehler. Ohne sie bestünde der Test auch
	// an einer Tür, die immer 500 antwortet.
	if rec := rufe(); rec.Code != http.StatusOK {
		t.Fatalf("ungestört: Status %d, erwartet 200: %.200s", rec.Code, rec.Body.String())
	}
	scharf = true
	defer func() { scharf = false }()
	return rufe()
}

// pruefeNeutrales500 verlangt Status 500 und eine Fehlerantwort, die den Text der Panik
// nicht trägt.
func pruefeNeutrales500(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status %d, erwartet 500: %.200s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || antwort.Error == "" {
		t.Errorf("Antwort ist keine Fehlermeldung der Form {\"error\": …}: %.200s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), panikText) {
		t.Errorf("die Antwort trägt den Text der Panik: %.200s", rec.Body.String())
	}
}

func TestPanik_OhneSentryAntwortetDerServerMit500(t *testing.T) {
	pruefeNeutrales500(t, anfrageInPanik(t, "panik-ohne-sentry@example.org"))
}

// Eingerichtet wird Sentry wie in main.go, allein mit der Adresse. Sie zeigt auf einen
// Server des Tests.
func TestPanik_MitSentryKommtDieMeldungAn(t *testing.T) {
	angekommen := make(chan struct{}, 1)
	sentryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rumpf, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.Contains(string(rumpf), panikText) {
			select {
			case angekommen <- struct{}{}:
			default:
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(sentryServer.Close)
	dsn := "http://schluessel@" + strings.TrimPrefix(sentryServer.URL, "http://") + "/1"
	if err := sentry.Init(sentry.ClientOptions{Dsn: dsn}); err != nil {
		t.Fatalf("sentry.Init: %v", err)
	}
	// Der Client hängt am Prozess; die Tests danach laufen wieder ohne ihn.
	t.Cleanup(func() { sentry.CurrentHub().BindClient(nil) })

	pruefeNeutrales500(t, anfrageInPanik(t, "panik-mit-sentry@example.org"))

	// Gewartet wird auf die Ankunft und nicht auf Flush: Die Bibliothek reicht eine Meldung
	// nebenläufig an den Versand weiter, und Flush kann davor zurückkehren.
	sentry.Flush(5 * time.Second)
	select {
	case <-angekommen:
	case <-time.After(5 * time.Second):
		t.Error("die Panik kam nicht bei Sentry an")
	}
}
