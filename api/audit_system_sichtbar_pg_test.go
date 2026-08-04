package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
)

// TestAuditLogZeigtSystemEintraege sichert einen Befund aus dem Audit vom 01.08.2026 ab:
//
// /api/audit verband audit_log per INNER JOIN mit benutzer. Systemgesteuerte Vorgaenge
// schreiben aber bewusst OHNE bearbeiter_id und mit akteur='SYSTEM' — DSGVO-Bereinigung,
// Backups, automatische Sperren. Der Inner Join hat genau diese Zeilen aussortiert.
//
// Das ist kein Schoenheitsfehler: Ein Pruefprotokoll wird gefuehrt, damit man
// nachvollziehen kann, was ohne Aufsicht am Bestand und an Schuelerdaten passiert ist.
// Ausgerechnet dieser Teil fehlte in der Anzeige — und zwar lautlos, weil eine kuerzere
// Liste nach nichts aussieht.
func TestAuditLogZeigtSystemEintraege(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	systemTabelle := "e2e_system_" + suffix
	nutzerTabelle := "e2e_nutzer_" + suffix

	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_log WHERE tabelle IN ($1, $2)`, systemTabelle, nutzerTabelle)
		aufraeumen(t, pool, `DELETE FROM benutzer WHERE email = $1`, "audit-"+suffix+"@example.org")
	})

	// Ein Eintrag MIT Bearbeiter ...
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Auda', 'Pruefer', $2, 'mitarbeiter', true) RETURNING id
	`, "AUD-"+suffix, "audit-"+suffix+"@example.org").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		VALUES ($1, 'UPDATE', gen_random_uuid(), $2, 'USER')
	`, nutzerTabelle, bearbeiterID); err != nil {
		t.Fatalf("Nutzer-Eintrag anlegen: %v", err)
	}

	// ... und einer OHNE, so wie LogSystemAktion ihn schreibt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		VALUES ($1, 'DELETE', gen_random_uuid(), NULL, 'SYSTEM')
	`, systemTabelle); err != nil {
		t.Fatalf("System-Eintrag anlegen: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.GetAuditLogsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, Koerper: %s", rec.Code, rec.Body.String())
	}

	var eintraege []AuditLogEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &eintraege); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}

	var system, nutzer *AuditLogEntry
	for i := range eintraege {
		switch eintraege[i].Tabelle {
		case systemTabelle:
			system = &eintraege[i]
		case nutzerTabelle:
			nutzer = &eintraege[i]
		}
	}

	if system == nil {
		t.Fatal("der SYSTEM-Eintrag fehlt in der Antwort — genau das war der Befund")
	}
	if system.Akteur != "SYSTEM" {
		t.Errorf("Akteur des System-Eintrags: erwartet SYSTEM, war %q", system.Akteur)
	}
	if system.BearbeiterID != "" {
		t.Errorf("System-Eintrag sollte keinen Bearbeiter tragen, war %q", system.BearbeiterID)
	}

	// Der Eintrag MIT Bearbeiter darf durch den LEFT JOIN nichts verloren haben.
	if nutzer == nil {
		t.Fatal("der Eintrag mit Bearbeiter fehlt")
	}
	if nutzer.BearbeiterVorname != "Auda" || nutzer.BearbeiterNachname != "Pruefer" {
		t.Errorf("Bearbeitername unvollstaendig: %q %q", nutzer.BearbeiterVorname, nutzer.BearbeiterNachname)
	}
}
