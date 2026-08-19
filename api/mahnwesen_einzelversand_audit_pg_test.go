package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// TestEinzelMahnVersandWirdAuditiert schließt die Audit-Asymmetrie aus dem IDOR-Sweep
// (19.08.2026): Der Massenversand (BULK_OVERDUE_MAIL) schrieb einen Audit-Eintrag, der
// Einzelversand POST /api/mahnwesen/senden — der ebenfalls an eine frei eingetippte
// Adresse geht — nicht. Für die DSGVO-Rechenschaftspflicht (Art. 5 (2)) muss auch er
// festhalten, wer die Mahnliste welcher Klasse an WEN geschickt hat.
func TestEinzelMahnVersandWirdAuditiert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Ein überfälliges Buch in Klasse 5b, damit der Versand echten Inhalt hat.
	ausleihbaresExemplar(t, pool, "Mahn-Roman", "B-EMV-1")
	schueler := schuelerAnlegen(t, pool, "Mira", "5b", "S-EMV-1")
	bearbeiter := adminFuerAudit(t, pool)
	ausleiheUeberDenDienst(t, pool, "B-EMV-1", schueler, bearbeiter)
	inDieVergangenheit(t, pool, schueler, 20)

	sitzungen := mailAbfangen(t)
	srv := &Server{DB: &db.Database{Pool: pool}}
	mahnRepo := repository.NewMahnwesenRepository(pool)

	ziel := "vertretung@schule.invalid"
	req := httptest.NewRequest(http.MethodPost, "/api/mahnwesen/senden",
		strings.NewReader(`{"klasse":"5b","email":"`+ziel+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(ctx, auth.ClaimsContextKey,
		&auth.Claims{UserID: bearbeiter, Rolle: auth.RoleAdmin}))

	rec := httptest.NewRecorder()
	srv.SendMahnwesenHandler(mahnRepo)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Einzelversand: Status %d — %s", rec.Code, rec.Body.String())
	}
	warteAufMail(t, sitzungen) // sicherstellen, dass wirklich versandt wurde

	var details string
	if err := pool.QueryRow(ctx,
		`SELECT details::text FROM audit_logs WHERE aktion = 'EINZEL_MAHN_MAIL' AND admin_id = $1`,
		bearbeiter).Scan(&details); err != nil {
		t.Fatalf("kein EINZEL_MAHN_MAIL-Audit gefunden: %v", err)
	}
	// Die Ziel-Adresse muss im Klartext im Audit stehen — das ist der ganze Zweck.
	if !strings.Contains(details, ziel) {
		t.Errorf("Audit muss die Empfänger-Adresse im Klartext führen, war: %s", details)
	}
	if !strings.Contains(details, "5b") {
		t.Errorf("Audit muss die Klasse nennen, war: %s", details)
	}
	// Gültiges JSON (jsonb-Spalte).
	var geparst map[string]any
	if err := json.Unmarshal([]byte(details), &geparst); err != nil {
		t.Errorf("Audit-Details sind kein gültiges JSON: %v", err)
	}
}
