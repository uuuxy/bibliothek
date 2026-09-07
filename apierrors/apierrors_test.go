package apierrors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decodeErrorBody parst genau das eine kanonische Fehlerschema {"error": "..."} und schlägt
// fehl, wenn der Body abweicht (fremder Key, kein JSON).
func decodeErrorBody(t *testing.T, body string) string {
	t.Helper()
	var parsed map[string]string
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("Fehler-Body ist kein JSON: %q (%v)", body, err)
	}
	if len(parsed) != 1 {
		t.Fatalf("Fehler-Body muss genau {\"error\":...} sein, war: %q", body)
	}
	msg, ok := parsed["error"]
	if !ok {
		t.Fatalf("Fehler-Body ohne \"error\"-Feld: %q", body)
	}
	return msg
}

// TestSendHTTPError_UserFacingStatusKeepsMessage: Bei nutzer-sichtbaren Status (kein 500,
// kein DB-Fehler) bleibt die fachliche Meldung erhalten.
func TestSendHTTPError_UserFacingStatusKeepsMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	SendHTTPError(rec, http.StatusConflict, errors.New("Benutzer hat noch aktive Ausleihen"))

	if rec.Code != http.StatusConflict {
		t.Errorf("Status: erwartet 409, war %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type: erwartet application/json, war %q", ct)
	}
	if msg := decodeErrorBody(t, rec.Body.String()); msg != "Benutzer hat noch aktive Ausleihen" {
		t.Errorf("Meldung verfälscht: %q", msg)
	}
}

// TestSendHTTPError_InternalErrorIsSanitized: Ein 500 darf niemals interne Details leaken.
func TestSendHTTPError_InternalErrorIsSanitized(t *testing.T) {
	rec := httptest.NewRecorder()
	SendHTTPError(rec, http.StatusInternalServerError,
		errors.New("pgx: SELECT * FROM geheim WHERE token = 'abc123'"))

	body := rec.Body.String()
	msg := decodeErrorBody(t, body)
	for _, leak := range []string{"pgx", "SELECT", "geheim", "abc123"} {
		if strings.Contains(msg, leak) {
			t.Errorf("interner Detail-Leak im 500-Body (%q): %q", leak, body)
		}
	}
}

// TestSendHTTPError_DBErrorSanitizedEvenOn4xx: Auch bei 4xx wird eine DB-nahe Meldung
// (SQL/Constraint) zensiert — sonst gelangte SQL-Struktur an den Client.
func TestSendHTTPError_DBErrorSanitizedEvenOn4xx(t *testing.T) {
	rec := httptest.NewRecorder()
	SendHTTPError(rec, http.StatusBadRequest,
		errors.New("ERROR: duplicate key value violates unique constraint \"foo_key\""))

	msg := decodeErrorBody(t, rec.Body.String())
	if strings.Contains(msg, "unique constraint") || strings.Contains(msg, "foo_key") {
		t.Errorf("DB-Detail im 4xx-Body geleakt: %q", msg)
	}
}

// TestWrapAndSendHTTPError_SameShape: Beide Fehler-Ausgabepfade müssen exakt dasselbe
// Wire-Format liefern ({"error": ...}), damit der Client sich auf ein Schema verlassen kann.
func TestWrapAndSendHTTPError_SameShape(t *testing.T) {
	// Pfad A: Wrap mit einem *APIError.
	handler := Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
		return Conflict("Konflikt X", errors.New("intern: darf nicht sichtbar sein"))
	})
	recA := httptest.NewRecorder()
	handler(recA, httptest.NewRequest(http.MethodGet, "/x", nil))

	// Pfad B: SendHTTPError direkt.
	recB := httptest.NewRecorder()
	SendHTTPError(recB, http.StatusConflict, errors.New("Konflikt X"))

	if recA.Code != recB.Code {
		t.Errorf("Status weicht ab: Wrap=%d, SendHTTPError=%d", recA.Code, recB.Code)
	}
	msgA := decodeErrorBody(t, recA.Body.String())
	msgB := decodeErrorBody(t, recB.Body.String())
	if msgA != "Konflikt X" || msgB != "Konflikt X" {
		t.Errorf("Meldungen: Wrap=%q, SendHTTPError=%q", msgA, msgB)
	}
	// Der interne Fehler von Pfad A darf nie im Body erscheinen.
	if strings.Contains(recA.Body.String(), "darf nicht sichtbar sein") {
		t.Errorf("Wrap leakt internen Fehler: %q", recA.Body.String())
	}
}

// Ein abgebrochener Browser-Request ist kein Serverfehler (Register B, 07.09.2026):
// Seitenwechsel während GET /api/exemplare/etiketten-offen/anzahl → pgx „context
// canceled“ → bis dahin 500 mit nutzlosem Stacktrace. Jetzt 499, egal ob der Fehler
// context.Canceled wrappt (SendHTTPError) oder nur der Request-Kontext abgebrochen ist
// (Wrap).
func TestSendHTTPError_ClientAbbruchIstKein500(t *testing.T) {
	rec := httptest.NewRecorder()
	SendHTTPError(rec, http.StatusInternalServerError, fmt.Errorf("query: %w", context.Canceled))
	if rec.Code != StatusClientClosedRequest {
		t.Fatalf("Status = %d, want 499", rec.Code)
	}
	if msg := decodeErrorBody(t, rec.Body.String()); !strings.Contains(msg, "abgebrochen") {
		t.Errorf("Meldung = %q, want Hinweis auf Abbruch", msg)
	}
}

func TestWrap_AbgebrochenerKontextIstKein500(t *testing.T) {
	ctx, abbrechen := context.WithCancel(context.Background())
	abbrechen()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	// Der Handler meldet einen Fehler, der context.Canceled NICHT wrappt — der Kontext
	// allein muss reichen.
	Wrap(func(http.ResponseWriter, *http.Request) error { return errors.New("scan: broken pipe") })(rec, req)
	if rec.Code != StatusClientClosedRequest {
		t.Fatalf("Status = %d, want 499", rec.Code)
	}
}

// Gegenprobe: ein echter 500 bleibt ein 500.
func TestWrap_EchterFehlerBleibt500(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rec := httptest.NewRecorder()
	Wrap(func(http.ResponseWriter, *http.Request) error { return errors.New("kaputt") })(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Status = %d, want 500", rec.Code)
	}
}
