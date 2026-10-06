package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Wer ändert, was eine Rolle darf, steht im Protokoll: Die Matrix entscheidet an jeder Route,
// wer welche Daten sieht. Konten (USER_UPDATE) und Einstellungen (UPDATE_SETTINGS) schrieben
// ihren Eintrag schon; an der Matrix fehlte er, und nach einer Änderung ließ sich nicht
// nachsehen, wer einer Rolle wann ein Recht gegeben hat. Eine abgelehnte Änderung schreibt
// keinen Eintrag.
func TestRechteMatrix_AenderungStehtImProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()

	authenticator, err := auth.NewAuthenticator(
		"rechte-protokoll-testgeheimnis-32-bytes!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Rita', 'Rechte', 'rechte-protokoll@example.org', 'admin', true)
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	sitzung, err := authenticator.GenerateToken(adminID, "RECHTE-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	router := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false).Routes()

	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed) VALUES ('HELFER', 'view_students', false)
		ON CONFLICT (role, permission) DO UPDATE SET allowed = false`); err != nil {
		t.Fatalf("Ausgangszustand setzen: %v", err)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `UPDATE role_permissions SET allowed = false WHERE role = 'HELFER' AND permission = 'view_students'`)
		aufraeumen(t, pool, `DELETE FROM audit_logs WHERE aktion = 'RECHT_GEAENDERT'`)
	})

	aendere := func(rumpf string) int {
		req := httptest.NewRequest(http.MethodPut, "/api/admin/permissions", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session_token", Value: sitzung})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, mitCSRF(req))
		return rec.Code
	}
	eintraege := func() int {
		return zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'RECHT_GEAENDERT'`)
	}

	if code := aendere(`{"role":"helfer","permission":"view_studentz","allowed":true}`); code != http.StatusBadRequest {
		t.Fatalf("unbekanntes Recht: Status %d, erwartet 400", code)
	}
	if n := eintraege(); n != 0 {
		t.Errorf("%d Einträge zu einer Änderung, die es nicht gab", n)
	}

	if code := aendere(`{"role":"helfer","permission":"view_students","allowed":true}`); code != http.StatusOK {
		t.Fatalf("Recht erteilen: Status %d, erwartet 200", code)
	}
	var wer, rolle, recht string
	var erlaubt bool
	if err := pool.QueryRow(ctx, `
		SELECT admin_id::text, details->>'rolle', details->>'recht', (details->>'erlaubt')::boolean
		FROM audit_logs WHERE aktion = 'RECHT_GEAENDERT'`).Scan(&wer, &rolle, &recht, &erlaubt); err != nil {
		t.Fatalf("Protokolleintrag lesen (genau einer erwartet): %v", err)
	}
	if wer != adminID || rolle != "HELFER" || recht != "view_students" || !erlaubt {
		t.Errorf("Eintrag nennt wer=%s rolle=%s recht=%s erlaubt=%v; erwartet %s, HELFER, view_students, true",
			wer, rolle, recht, erlaubt, adminID)
	}

	if code := aendere(`{"role":"helfer","permission":"view_students","allowed":false}`); code != http.StatusOK {
		t.Fatalf("Recht entziehen: Status %d, erwartet 200", code)
	}
	if n := eintraege(); n != 2 {
		t.Errorf("%d Einträge nach Erteilen und Entziehen, erwartet 2", n)
	}
}
