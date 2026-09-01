package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// Regressionstest zum Fund vom 01.09.2026, entdeckt vom PII-POST-Gate bei seinem
// ersten Lauf: ActionBatchHandler rief DecodeAndValidate auf ActionBatchRequest —
// einem SLICE. go-playground/validator kann nur Structs prüfen; damit kam JEDER
// wohlgeformte Batch-Body (das Frontend schickt ein nacktes Array, siehe
// offlineSync.svelte.js) als 400 „validator: …" zurück. Der Offline-Sync der
// Theke scheiterte still bei jedem Anlauf und ließ seine Warteschlange ewig
// liegen — kein Test hatte die Route je aufgerufen (Erreichbarkeits-Klasse
// „nie verdrahtet"). Die Item-Prüfung leistet processSingleBatchItem je Eintrag;
// der Handler darf deshalb nur dekodieren, nicht Struct-validieren.
func TestActionBatch_ArrayBodyWirdAngenommen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Setenv("RATE_LIMIT", "100000")
	ctx := context.Background()

	// role_permissions über den Live-Aufbau — MITARBEITER trägt perform_actions
	// aus der Seed-Vorgabe.
	if err := (&db.Database{Pool: pool}).InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
	authenticator, err := auth.NewAuthenticator(
		"action-batch-testgeheimnis-mind-32-bytes!!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	srv := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false)
	router := srv.Routes()

	var mitarbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ('BATCH-MA-1', 'Betti', 'Batchwart', 'batch-ma@example.org', 'mitarbeiter', true)
		RETURNING id`).Scan(&mitarbeiterID); err != nil {
		t.Fatalf("Mitarbeiter anlegen: %v", err)
	}
	token, err := authenticator.GenerateToken(mitarbeiterID, "BATCH-MA-1", auth.RoleMitarbeiter)
	if err != nil {
		t.Fatalf("Session-Token: %v", err)
	}
	seedSchueler(t, pool, "SBK-BATCH-1", "Batchtest", "05A")

	req := httptest.NewRequest(http.MethodPost, "/api/action/batch",
		strings.NewReader(`[{"query":"SBK-BATCH-1"}]`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "batch-csrf"})
	req.Header.Set("X-CSRF-Token", "batch-csrf")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Batch mit Array-Body: Status %d, erwartet 200 — genau so scheiterte der Offline-Sync still: %s",
			rec.Code, rec.Body.String())
	}
	var resp ActionBatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort kein JSON: %v", err)
	}
	if len(resp.Results) != 1 || !resp.Results[0].Success {
		t.Fatalf("Batch-Item nicht verarbeitet: %+v", resp.Results)
	}
	if resp.Results[0].Data == nil || !strings.Contains(rec.Body.String(), "Batchtest") {
		t.Errorf("Scan-Antwort ohne Schüler-DTO: %+v", resp.Results[0])
	}

	// Leerer Query-Eintrag: kein 400 für den ganzen Stapel, sondern ein
	// Fehler-Item — die Prüfung sitzt seit dem Fix ausschließlich je Eintrag.
	req2 := httptest.NewRequest(http.MethodPost, "/api/action/batch",
		strings.NewReader(`[{"query":""}]`))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: "batch-csrf"})
	req2.Header.Set("X-CSRF-Token", "batch-csrf")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("Batch mit leerem Query: Status %d, erwartet 200 mit Fehler-Item", rec2.Code)
	}
	var resp2 ActionBatchResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("Antwort kein JSON: %v", err)
	}
	if len(resp2.Results) != 1 || resp2.Results[0].Success || resp2.Results[0].Status != http.StatusBadRequest {
		t.Errorf("leerer Query muss ein 400-Item ergeben, war: %+v", resp2.Results)
	}
}
