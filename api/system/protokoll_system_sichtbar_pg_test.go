package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
)

// TestProtokollZeigtSystemEintraege: /api/audit zeigt auch Einträge des Systems. Sie
// haben keine bearbeiter_id und akteur='SYSTEM' (DSGVO-Bereinigung, Backups, automatische
// Sperren); mit einem INNER JOIN auf benutzer fielen genau diese Zeilen heraus.
//
// Ein Protokoll wird geführt, damit man nachvollziehen kann, was ohne Aufsicht am Bestand und
// an Schülerdaten passiert ist. Fehlte dieser Teil, fiele es nicht auf, weil eine kürzere Liste
// nach nichts aussieht.
func TestProtokollZeigtSystemEintraege(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Database{Pool: pool}}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	systemTabelle := "e2e_system_" + suffix
	nutzerTabelle := "e2e_nutzer_" + suffix

	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_log WHERE tabelle IN ($1, $2)`, systemTabelle, nutzerTabelle)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE email = $1`, "audit-"+suffix+"@example.org")
	})

	// Ein Eintrag mit Bearbeiter ...
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Auda', 'Pruefer', $1, 'mitarbeiter', true) RETURNING id
	`, "audit-"+suffix+"@example.org").Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		VALUES ($1, 'UPDATE', gen_random_uuid(), $2, 'USER')
	`, nutzerTabelle, bearbeiterID); err != nil {
		t.Fatalf("Nutzer-Eintrag anlegen: %v", err)
	}

	// ... und einer ohne, so wie ProtokolliereSystemAktion ihn schreibt.
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		VALUES ($1, 'DELETE', gen_random_uuid(), NULL, 'SYSTEM')
	`, systemTabelle); err != nil {
		t.Fatalf("System-Eintrag anlegen: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.Protokoll().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, Koerper: %s", rec.Code, rec.Body.String())
	}

	var eintraege []protokollEintrag
	if err := json.Unmarshal(rec.Body.Bytes(), &eintraege); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}

	var system, nutzer *protokollEintrag
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

	// Der Eintrag mit Bearbeiter darf durch den LEFT JOIN nichts verloren haben.
	if nutzer == nil {
		t.Fatal("der Eintrag mit Bearbeiter fehlt")
	}
	if nutzer.BearbeiterVorname != "Auda" || nutzer.BearbeiterNachname != "Pruefer" {
		t.Errorf("Bearbeitername unvollstaendig: %q %q", nutzer.BearbeiterVorname, nutzer.BearbeiterNachname)
	}
}
