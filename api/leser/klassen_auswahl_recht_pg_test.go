package leser_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"
)

// Die Klasse wählt man in der Leserdatei (view_students), in den Klassensätzen und der
// LMF-Verlängerung (edit_books) und im Mahnwesen-Routing (manage_settings). Die Liste der
// Klassen hängt deshalb an der Sitzung und nicht an einem dieser Rechte. Am echten Router: Mit
// jedem der drei Rechte allein kommt die Liste, ohne Sitzung kommt sie nicht.
func TestKlassen_JedeRolleMitKlassenauswahlBekommtDieListe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	t.Setenv("RATE_LIMIT", "100000")
	ctx := context.Background()

	if err := (&db.Verbindung{Pool: pool}).SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
	}
	authenticator, err := auth.NewAnmeldung("klassen-auswahl-testgeheimnis-mind-32-bytes!", pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	srv := api.NewServer(&db.Verbindung{Pool: pool}, authenticator, sse.NewBroker(), api.ServerKonfig{})
	router := srv.Routes()

	kontoID := testdaten.LegeKontoAn(t, pool, "Kim", "Klassenwahl", "kim@klassen-auswahl.invalid", "mitarbeiter", true)
	token, err := authenticator.ErzeugeToken(kontoID, "KLASSENWAHL-1", auth.RolleMitarbeiter, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	t.Cleanup(func() {
		// Die Rechte der Rolle stehen danach wieder auf der Vorgabe.
		pgtest.RaeumeAuf(t, pool, `DELETE FROM role_permissions WHERE role = 'MITARBEITER'`)
		for _, e := range db.RechteVorgabe {
			if e.Rolle == "MITARBEITER" {
				pgtest.RaeumeAuf(t, pool, `INSERT INTO role_permissions (role, permission, allowed)
					VALUES ($1, $2, $3) ON CONFLICT (role, permission) DO NOTHING`, e.Rolle, e.Recht, e.Erlaubt)
			}
		}
		pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE id = $1`, kontoID)
	})

	leser := testdaten.LegeSchuelerAn(t, pool, "KW-1", "Kai", "7c")
	var klasse string
	if err := pool.QueryRow(ctx, `SELECT klasse FROM leser WHERE id = $1`, leser).Scan(&klasse); err != nil {
		t.Fatal(err)
	}

	hole := func(mitSitzung bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/klassen", nil)
		if mitSitzung {
			req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	for _, recht := range []string{"view_students", "edit_books", "manage_settings"} {
		tuertest.SetzeGenauEinRecht(t, pool, &srv.Rechte, recht)
		rec := hole(true)
		if rec.Code != http.StatusOK {
			t.Errorf("GET /api/klassen mit %s allein: Status %d, erwartet 200: %s", recht, rec.Code, rec.Body.String())
			continue
		}
		if !strings.Contains(rec.Body.String(), `"`+klasse+`"`) {
			t.Errorf("GET /api/klassen mit %s allein: %s, erwartet die Klasse %q", recht, rec.Body.String(), klasse)
		}
	}

	if rec := hole(false); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/klassen ohne Sitzung: Status %d, erwartet 401", rec.Code)
	}
}
