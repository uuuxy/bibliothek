package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/sse"
)

// Abmelden beendet die ganze Anmeldung, nicht nur das Token, mit dem abgemeldet wird. Die
// Erneuerung stellt ein neues Token aus und lässt das alte gelten, weil gleichzeitige Anfragen es
// noch tragen; beide tragen dieselbe Kennung der Anmeldung. Widerriefe das Abmelden nur das
// letzte Token, bliebe das vorige bis zu seinem Ablauf ein voller Zugang.
func TestAbmeldung_BeendetAuchDasTokenVorDerErneuerung(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const geheimnis = "pg-test-abmeldung-ganze-sitzung-32-zeichen"

	kontoID := testdaten.LegeKontoAn(t, pool, "Abmelde", "Probe", "abmeldung-ganze-sitzung@example.org", "mitarbeiter", true)
	t.Cleanup(func() { pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE id = $1`, kontoID) })

	var sitzungID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sitzungen (benutzer_id, passwort_pruefwert, laeuft_ab)
		VALUES ($1, 'nur-fuer-den-test', NOW() + INTERVAL '1 hour')
		RETURNING id::text`, kontoID).Scan(&sitzungID); err != nil {
		t.Fatalf("Zeile der Anmeldung anlegen: %v", err)
	}

	// Das Token vor der Erneuerung läuft in einer Stunde ab, weniger als die halbe Laufzeit des
	// Servers: Die Erneuerung stellt ein neues aus.
	kurz, err := auth.NewAnmeldung(geheimnis, pool, time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	alt, err := kurz.ErzeugeToken(kontoID, auth.RolleMitarbeiter, sitzungID)
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}
	anmeldung, err := auth.NewAnmeldung(geheimnis, pool, 12*time.Hour)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	router := NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), ServerKonfig{}).Routes()

	rufe := func(methode, pfad, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(methode, pfad, nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec := rufe(http.MethodPost, "/api/auth/refresh", alt)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"refresh":"renewed"`) {
		t.Fatalf("Erneuerung: Status %d: %s", rec.Code, rec.Body.String())
	}
	neu := ""
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_token" {
			neu = c.Value
		}
	}
	if neu == "" || neu == alt {
		t.Fatal("die Erneuerung hat kein neues Token ausgestellt")
	}

	// Vor dem Abmelden gelten beide; sonst prüfte der Rest nichts.
	for name, token := range map[string]string{"vor der Erneuerung": alt, "erneuert": neu} {
		if rec := rufe(http.MethodGet, "/api/einstellungen/sitzung", token); rec.Code != http.StatusOK {
			t.Fatalf("vor dem Abmelden, Token %s: Status %d: %s", name, rec.Code, rec.Body.String())
		}
	}

	if rec := rufe(http.MethodPost, "/api/auth/logout", neu); rec.Code != http.StatusOK {
		t.Fatalf("Abmelden: Status %d: %s", rec.Code, rec.Body.String())
	}

	for _, pfad := range []string{"/api/einstellungen/sitzung", "/api/auth/me"} {
		if rec := rufe(http.MethodGet, pfad, alt); rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s mit dem Token vor der Erneuerung, nach dem Abmelden: Status %d, erwartet 401: %s",
				pfad, rec.Code, rec.Body.String())
		}
	}
}
