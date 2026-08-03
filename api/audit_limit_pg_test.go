package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/db"
)

// TestAuditLogIstBegrenzt belegt, dass /api/audit nicht die gesamte Tabelle ausliefert.
//
// Vorher stand in der Abfrage kein LIMIT. Mit 247.000 Zeilen ergab das eine Antwort von
// 72 MB: Der Server lieferte sie in 0,6 s, aber der Browser kam mit dem Parsen und dem
// Bauen ebenso vieler Tabellenzeilen nicht mehr zurecht — die Seite blieb leer, und der
// E2E-Test für die System-Logs lief in den Timeout. Das Logbuch wächst im Betrieb
// unbegrenzt weiter, der Fall tritt also zwangsläufig ein.
//
// Geprüft wird an einer echten Datenbank mit MEHR Zeilen, als die Grenze zulässt. Ein
// Test gegen eine kleine Tabelle wäre wertlos — er wäre auch ohne LIMIT grün.
func TestAuditLogIstBegrenzt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Ein Bearbeiter, an dem der JOIN hängt. Ohne ihn liefert die Abfrage gar nichts,
	// und der Test wäre auch bei kaputter Grenze grün.
	var bearbeiterID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Audit', 'Limit', 'audit-limit@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id
	`).Scan(&bearbeiterID); err != nil {
		t.Fatalf("Bearbeiter anlegen: %v", err)
	}

	// Absichtlich mehr Zeilen als die Grenze, mit fallenden Zeitstempeln.
	const ueberschuss = auditLogMaxZeilen + 250
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, timestamp)
		SELECT 'buecher', 'DELETE', gen_random_uuid(), $1, 'USER',
		       CURRENT_TIMESTAMP - (g || ' seconds')::interval
		FROM generate_series(1, $2) AS g
	`, bearbeiterID, ueberschuss); err != nil {
		t.Fatalf("Logzeilen anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	rec := httptest.NewRecorder()
	srv.GetAuditLogsHandler()(rec, httptest.NewRequest(http.MethodGet, "/api/audit", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, erwartet 200 — Antwort: %s", rec.Code, rec.Body.String())
	}

	var eintraege []AuditLogEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &eintraege); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}

	if len(eintraege) != auditLogMaxZeilen {
		t.Fatalf("%d Einträge geliefert, erwartet genau %d — ohne LIMIT wären es %d",
			len(eintraege), auditLogMaxZeilen, ueberschuss)
	}

	// Und es sind die JÜNGSTEN. Eine Grenze, die die ältesten liefert, wäre schlimmer als
	// keine: Das Logbuch zeigte dauerhaft nur die Vergangenheit, und niemand bemerkte es.
	if eintraege[0].Timestamp.Before(eintraege[1].Timestamp) {
		t.Fatalf("Reihenfolge falsch: erster Eintrag (%s) ist älter als der zweite (%s)",
			eintraege[0].Timestamp, eintraege[1].Timestamp)
	}
}
