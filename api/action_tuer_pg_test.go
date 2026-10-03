package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/google/uuid"
)

// Die Theken-Tür weist ab, bevor gebucht wird: ohne Sitzung, ohne Suchtext, und solange eine
// Anfrage mit demselben Schlüssel noch arbeitet. In keinem der Fälle wird der Dienst gerufen.
func TestActionHandler_AblehnungenVorDerBuchung(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}, IdempotenzWartezeit: 150 * time.Millisecond}
	// Ein Aufruf des Dienstes endete als 500 — keiner der erwarteten Status käme zustande.
	dienst := &sperrOmnibox{fehler: errors.New("darf nicht gerufen werden")}
	rufe := func(rumpf string, angemeldet bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/action", strings.NewReader(rumpf))
		if angemeldet {
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
				&auth.Claims{Rolle: auth.RoleAdmin, UserID: "u1"}))
		}
		w := httptest.NewRecorder()
		srv.ActionHandler(dienst)(w, req)
		return w
	}

	if w := rufe(`{"query":"B-123"}`, false); w.Code != http.StatusUnauthorized {
		t.Errorf("ohne Sitzung: Status %d, erwartet 401: %s", w.Code, w.Body.String())
	}
	if w := rufe(`{"query":"   "}`, true); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "leer") {
		t.Errorf("Suchtext aus Leerzeichen: Status %d, erwartet 400 mit dem Hinweis „leer“: %s", w.Code, w.Body.String())
	}

	// Eine andere Anfrage hält den Schlüssel und hat noch nicht geantwortet.
	schluessel := uuid.NewString()
	reserviert, _, err := repository.ReserviereIdempotenzSchluessel(ctx, pool, schluessel)
	if err != nil || !reserviert {
		t.Fatalf("Schlüssel reservieren: reserviert %v, %v", reserviert, err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE idempotency_key = $1`, schluessel); err != nil {
			t.Errorf("Schlüssel aufräumen: %v", err)
		}
	})
	w := rufe(`{"query":"B-123","idempotency_key":"`+schluessel+`"}`, true)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "in_arbeit") {
		t.Errorf("Schlüssel in Arbeit: Status %d, erwartet 409 mit „in_arbeit“: %s", w.Code, w.Body.String())
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM idempotency_keys WHERE idempotency_key = $1 AND status_code = 0`, schluessel); n != 1 {
		t.Errorf("%d Reservierungen unter dem Schlüssel, erwartet 1 — die abgewiesene Anfrage darf sie weder übernehmen noch freigeben", n)
	}
}

// Die Nachbuch-Tür nimmt ohne Sitzung nichts an; der Dienst wird nicht gerufen.
func TestNachbuchenHandler_OhneSitzung(t *testing.T) {
	rumpf := `{"gesendet_am":"2026-09-01T10:00:00Z","eintraege":[{"schluessel":"` + uuid.NewString() +
		`","absicht":"ausleihe","barcode":"B-1","gescannt_am":"2026-09-01T09:59:00Z"}]}`
	rec := httptest.NewRecorder()
	(&Server{}).NachbuchenHandler(nachbuchFehler{})(rec, httptest.NewRequest(http.MethodPost, "/api/action/nachbuchen", strings.NewReader(rumpf)))
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "nicht angemeldet") {
		t.Errorf("ohne Sitzung: Status %d, erwartet 401 mit „nicht angemeldet“: %s", rec.Code, rec.Body.String())
	}
}
