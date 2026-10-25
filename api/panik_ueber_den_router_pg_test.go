package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Gerät die Bearbeitung einer Anfrage in Panik, antwortet der Server mit einem neutralen 500
// und läuft weiter. Das hängt an panikMiddleware, einer der äußeren Schichten des Routers.
//
// Über den ganzen Router mit Sitzung: Die Schicht hängt am Router, nicht an der Tür.

const panikText = "Probe-Panik aus dem Test"

// poolMitPanik gerät in Panik, sobald die Tür das Design der Ausweise liest und scharf gesetzt
// ist. Sitzung und Rechte laufen am echten Pool.
type poolMitPanik struct {
	db.Pool
	scharf *bool
}

func (p poolMitPanik) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if *p.scharf && strings.Contains(sql, "FROM system_einstellungen") && len(args) > 0 && args[0] == repository.AusweisLayoutSchluessel {
		panic(panikText)
	}
	return p.Pool.QueryRow(ctx, sql, args...)
}

// anfrageInPanik schickt dieselbe Anfrage zweimal über den ganzen Router: erst ungestört,
// dann mit einem Pool, der beim Lesen der Tür in Panik gerät. Zurück kommt die Antwort auf
// die zweite.
func anfrageInPanik(t *testing.T, konto string) *httptest.ResponseRecorder {
	t.Helper()
	pool := pgtest.Pool(t)
	scharf := false
	_, sitzung, router := routertest.MitSitzungUeber(t, pool, poolMitPanik{Pool: pool, scharf: &scharf},
		konto, "Paula", "Panik")
	t.Cleanup(func() { pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE email = $1`, konto) })
	rufe := func() (rec *httptest.ResponseRecorder) {
		t.Helper()
		// Ohne panikMiddleware erreichte die Panik den Aufrufer; der Test soll das
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

func TestPanik_DerServerAntwortetMit500(t *testing.T) {
	var protokoll bytes.Buffer
	vorher := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&protokoll, nil)))
	t.Cleanup(func() { slog.SetDefault(vorher) })

	pruefeNeutrales500(t, anfrageInPanik(t, "panik@example.org"))

	// Die Panik steht einmal im Log, mit ihrem Stack; eine zweite Zeile zählte sie auf der Seite
	// „Fehler im Log“ doppelt.
	if n := strings.Count(protokoll.String(), panikText); n != 1 {
		t.Errorf("%d Zeilen mit dem Text der Panik, erwartet eine:\n%s", n, protokoll.String())
	}
}
