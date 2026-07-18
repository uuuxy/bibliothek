package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// TestLockStudentHandler_RequiresReasonAndSetsBlockReason prüft den Lock-Endpoint end-to-end:
// Sperren ohne Grund wird abgelehnt (400), Sperren mit Grund setzt is_manually_blocked und
// block_reason, Entsperren räumt beides wieder. Vorher konnte der Toggle eine grundlose
// "Zombie-Sperre" anlegen.
func TestLockStudentHandler_RequiresReasonAndSetsBlockReason(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('LK-1', 'Max', 'Muster', '7a', 2030) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/admin/students/"+id+"/lock", strings.NewReader(body))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		srv.LockStudentHandler().ServeHTTP(rec, req)
		return rec
	}
	leseSperre := func() (bool, *string) {
		t.Helper()
		var blocked bool
		var reason *string
		if err := pool.QueryRow(ctx,
			`SELECT is_manually_blocked, block_reason FROM schueler WHERE id = $1`, id).Scan(&blocked, &reason); err != nil {
			t.Fatal(err)
		}
		return blocked, reason
	}

	// Sperren ohne Grund → 400, keine Änderung.
	if rec := call(`{"is_locked":true}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Sperre ohne Grund erwartet 400, war %d: %s", rec.Code, rec.Body.String())
	}
	if blocked, _ := leseSperre(); blocked {
		t.Error("Schüler wurde trotz fehlendem Grund gesperrt")
	}

	// Sperren mit Grund → 200, block_reason gesetzt.
	if rec := call(`{"is_locked":true,"reason":"Wiederholt nicht zurückgegeben"}`); rec.Code != http.StatusOK {
		t.Fatalf("Sperre mit Grund erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if blocked, reason := leseSperre(); !blocked || reason == nil || *reason != "Wiederholt nicht zurückgegeben" {
		t.Errorf("Sperre nicht korrekt gesetzt: blocked=%v reason=%v", blocked, reason)
	}

	// Entsperren → 200, block_reason geräumt (keine Systemsperre vorhanden).
	if rec := call(`{"is_locked":false}`); rec.Code != http.StatusOK {
		t.Fatalf("Entsperren erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if blocked, reason := leseSperre(); blocked || reason != nil {
		t.Errorf("Entsperren nicht korrekt: blocked=%v reason=%v", blocked, reason)
	}
}
