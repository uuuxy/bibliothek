package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Sperren hat GENAU EINE Tür.
//
// Es gab zwei: den Sperr-Endpunkt (PATCH /api/admin/students/{id}/lock), der einen Grund
// verlangt und ihn beim Entsperren wieder konsistent räumt — und den allgemeinen
// Schüler-PATCH, der dieselben zwei Spalten ohne jede Prüfung schrieb.
//
// Gemessen am laufenden Handler, bevor die zweite Tür zuging:
//
//	Sperr-Endpunkt ohne Grund   -> 400  "Für eine manuelle Sperre ist ein Grund erforderlich."
//	Allgemeiner PATCH ohne Grund -> 500  "Ein interner Datenbankfehler ist aufgetreten."
//
// Der 500er kam von chk_schueler_block_reason, und der Sanitizer ersetzt jede
// 500-Meldung — auf dem Bildschirm stand also nichts, was weiterhilft. Zwei Türen zu
// demselben Zustand, von denen nur eine die Regeln kennt: Die falsche geht irgendwann auf.
func TestSperreNurUeberDenSperrEndpunkt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		VALUES ('Tür','Test','07a','TUER-1', 2030) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	fahre := func(body string, h http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec
	}
	auditRepo := repository.NewAuditRepository(pool)

	gesperrt := func() bool {
		t.Helper()
		var b bool
		if err := pool.QueryRow(ctx,
			`SELECT coalesce(is_manually_blocked, false) FROM schueler WHERE id = $1`, id).Scan(&b); err != nil {
			t.Fatalf("Sperrzustand lesen: %v", err)
		}
		return b
	}

	// 1. Der allgemeine PATCH kann NICHT mehr sperren. Das Feld ist keins mehr, also
	//    bleibt vom Rumpf nichts Änderbares übrig — 400 statt eines 500ers aus der Tiefe.
	rec := fahre(`{"is_manually_blocked":true}`, srv.PatchStudentHandler(auditRepo))
	if rec.Code == http.StatusInternalServerError {
		t.Errorf("PATCH mit Sperrfeld gibt %d — der alte 500er aus chk_schueler_block_reason ist zurück.\n"+
			"→ Sperren gehört ausschliesslich in LockStudentHandler; dort gibt es einen Grund und eine Meldung.",
			rec.Code)
	}
	if gesperrt() {
		t.Fatal("der allgemeine PATCH hat gesperrt — die zweite Tür ist wieder offen")
	}

	// 2. Und der Sperr-Endpunkt erklärt weiterhin, was fehlt, statt abzustürzen.
	rec = fahre(`{"is_locked":true}`, srv.LockStudentHandler())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Sperr-Endpunkt ohne Grund: Status %d, erwartet 400 — %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Grund") {
		t.Errorf("die Meldung nennt den fehlenden Grund nicht: %s", rec.Body.String())
	}
	if gesperrt() {
		t.Error("abgelehnt, aber trotzdem gesperrt")
	}

	// 3. Mit Grund geht es — sonst prüfte der Test nur, dass nichts funktioniert.
	rec = fahre(`{"is_locked":true,"reason":"Bücher nicht zurückgegeben"}`, srv.LockStudentHandler())
	if rec.Code != http.StatusOK {
		t.Fatalf("Sperren mit Grund: Status %d — %s", rec.Code, rec.Body.String())
	}
	if !gesperrt() {
		t.Error("mit Grund gesperrt, aber die Sperre steht nicht in der Datenbank")
	}
}
