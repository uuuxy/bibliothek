package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/auskunft"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Auskunft über einen Leser MIT Zugangskonto verlangt das Recht für Konten
// (manage_users), nicht nur das Sonderrecht der Schülerverwaltung (28.09.2026).
//
// Seit dem 24.09.2026 gibt es die Auskunft für jeden Leser. Beim Kollegen mit Konto nennt sie
// das Konto, die Kontoereignisse aus dem Verwaltungsprotokoll und jeden selbst bearbeiteten
// Vorgang, Verwaltungseingriffe samt IP-Adresse. Die Route verlangt manage_students_admin: ab
// Werk hat es die Leitung, und laut db/seed.go (RechteOptional) ist es zum Delegieren ans
// Sekretariat gedacht. Konten und Verwaltungsprotokoll zeigt die Anwendung sonst nur mit
// manage_users (GET /api/benutzer, GET /api/admin/auditlog), und das hat die Leitung bewusst
// nicht (Migration 122). Über die Auskunft bekam sie das Tätigkeitsprotokoll jedes Kollegen
// mit Konto, auch das des Administrators.
//
// Gegenproben: Dieselbe Rolle bekommt die Auskunft eines Kollegen OHNE Konto (die Regel hängt
// am Konto, nicht an der Leserart). Der Administrator bekommt die mit Konto. Erteilt der Admin
// der Leitung manage_users, bekommt auch sie sie (geprüft wird das Recht, nicht die Rolle).
func TestDsgvoAuskunft_KontoVerlangtKontenrecht(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Die Rechtezeile der Leitung steht je nach Lauf schon da (Seed anderer Tests). Sie wird
	// für die Probe auf „nein" gesetzt und danach genau so zurückgestellt, wie sie war.
	var vorher *bool
	if err := pool.QueryRow(ctx, `SELECT allowed FROM role_permissions
		WHERE role = 'LEITUNG' AND permission = 'manage_users'`).Scan(&vorher); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Rechtezeile lesen: %v", err)
	}
	setzeKontenrecht := func(erlaubt bool) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO role_permissions (role, permission, allowed)
			VALUES ('LEITUNG', 'manage_users', $1)
			ON CONFLICT (role, permission) DO UPDATE SET allowed = EXCLUDED.allowed`, erlaubt); err != nil {
			t.Fatalf("Rechtezeile setzen: %v", err)
		}
		InvalidatePermissionCache()
	}
	setzeKontenrecht(false)
	t.Cleanup(func() {
		zurueck := `DELETE FROM role_permissions WHERE role = 'LEITUNG' AND permission = 'manage_users'`
		args := []any{}
		if vorher != nil {
			zurueck = `UPDATE role_permissions SET allowed = $1 WHERE role = 'LEITUNG' AND permission = 'manage_users'`
			args = append(args, *vorher)
		}
		if _, err := pool.Exec(context.Background(), zurueck, args...); err != nil {
			t.Errorf("Rechtezeile zurückstellen: %v", err)
		}
		for _, sql := range []string{
			`DELETE FROM audit_log WHERE aktion = 'dsgvo_auskunft'
			    AND bearbeiter_id IN (SELECT id FROM benutzer WHERE email LIKE '%@auskunft-recht.invalid')`,
			`DELETE FROM audit_logs WHERE admin_id IN (SELECT id FROM benutzer WHERE email LIKE '%@auskunft-recht.invalid')`,
			`DELETE FROM benutzer WHERE email LIKE '%@auskunft-recht.invalid'`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
		InvalidatePermissionCache()
	})

	// Konten: Der Trigger trg_benutzer_hat_leserzeile hängt jedem die Leserzeile an.
	konto := func(vorname, email, rolle string) (kontoID, leserID string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ($1, 'Recht', $2, $3, true)
			RETURNING id, leser_id`, vorname, email, rolle).Scan(&kontoID, &leserID); err != nil {
			t.Fatalf("Konto %s anlegen: %v", email, err)
		}
		return kontoID, leserID
	}
	rita, ritaLeser := konto("Rita", "rita@auskunft-recht.invalid", "mitarbeiter")
	lea, _ := konto("Lea", "lea@auskunft-recht.invalid", "leitung")
	ada, _ := konto("Ada", "ada@auskunft-recht.invalid", "admin")
	if _, err := pool.Exec(ctx, `INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse)
		VALUES ($1, 'USER_UPDATE', '{}', '10.1.2.3')`, rita); err != nil {
		t.Fatalf("Verwaltungseingriff anlegen: %v", err)
	}
	var ottoLeser string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art)
		VALUES ('Otto', 'Ohnekonto', 'lehrkraft') RETURNING id`).Scan(&ottoLeser); err != nil {
		t.Fatalf("Kollege ohne Konto anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	rufe := func(h http.HandlerFunc, leserID, kontoID string, rolle auth.Role) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+leserID+"/dsgvo-auskunft", nil)
		req.SetPathValue("id", leserID)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: kontoID, Rolle: rolle}))
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec
	}
	protokolliert := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log
			WHERE aktion = 'dsgvo_auskunft' AND datensatz_id = $1::uuid AND bearbeiter_id = $2`,
			ritaLeser, lea).Scan(&n); err != nil {
			t.Fatalf("Protokoll zählen: %v", err)
		}
		return n
	}

	for name, h := range map[string]http.HandlerFunc{
		"JSON": srv.DsgvoAuskunftHandler(), "PDF": srv.DsgvoAuskunftPDFHandler(),
	} {
		rec := rufe(h, ritaLeser, lea, auth.RoleLeitung)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: die Leitung ohne manage_users bekommt die Auskunft eines Kollegen mit Konto "+
				"(Status %d, erwartet 403)", name, rec.Code)
		}
		for _, verboten := range []string{"rita@auskunft-recht.invalid", "10.1.2.3"} {
			if strings.Contains(rec.Body.String(), verboten) {
				t.Errorf("%s: die Antwort an die Leitung enthält %q", name, verboten)
			}
		}
	}
	if n := protokolliert(); n != 0 {
		t.Errorf("eine verweigerte Auskunft steht %d-mal als erteilt im Protokoll", n)
	}

	if rec := rufe(srv.DsgvoAuskunftHandler(), ottoLeser, lea, auth.RoleLeitung); rec.Code != http.StatusOK {
		t.Errorf("Gegenprobe: die Leitung bekommt die Auskunft eines Kollegen ohne Konto nicht (Status %d) — %s",
			rec.Code, rec.Body.String())
	}

	// Ein gelöschtes Konto zählt wie ein bestehendes (29.09.2026): Die Auskunft nennt dann das
	// frühere Konto samt seinen Einträgen, dieselbe Art Daten. Der Ausweis hält die Leserzeile,
	// wenn das Konto über Benutzer & Rechte fällt.
	fritz, fritzLeser := konto("Fritz", "fritz@auskunft-recht.invalid", "mitarbeiter")
	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = 'AR-FRITZ-1' WHERE id = $1`, fritzLeser); err != nil {
		t.Fatalf("Ausweis für Fritz: %v", err)
	}
	if err := repository.NewAuditRepository(pool).DeleteUser(ctx, fritz, ada); err != nil {
		t.Fatalf("Konto von Fritz löschen: %v", err)
	}
	for name, h := range map[string]http.HandlerFunc{
		"JSON": srv.DsgvoAuskunftHandler(), "PDF": srv.DsgvoAuskunftPDFHandler(),
	} {
		rec := rufe(h, fritzLeser, lea, auth.RoleLeitung)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: die Leitung ohne manage_users bekommt die Auskunft eines Kollegen mit gelöschtem Konto "+
				"(Status %d, erwartet 403)", name, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "fritz@auskunft-recht.invalid") {
			t.Errorf("%s: die Antwort an die Leitung enthält die Adresse des gelöschten Kontos", name)
		}
	}

	rec := rufe(srv.DsgvoAuskunftHandler(), ritaLeser, ada, auth.RoleAdmin)
	if rec.Code != http.StatusOK {
		t.Fatalf("Gegenprobe: der Administrator bekommt die Auskunft nicht (Status %d) — %s", rec.Code, rec.Body.String())
	}
	var a auskunft.DsgvoAuskunftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if a.Zugangskonto == nil || !strings.Contains(rec.Body.String(), "10.1.2.3") {
		t.Error("Gegenprobe: die Auskunft des Administrators nennt Konto und IP-Adresse nicht — " +
			"die Probe oben wäre ohne Aussage")
	}

	setzeKontenrecht(true)
	if rec := rufe(srv.DsgvoAuskunftHandler(), ritaLeser, lea, auth.RoleLeitung); rec.Code != http.StatusOK {
		t.Errorf("Gegenprobe: die Leitung mit manage_users bekommt die Auskunft nicht (Status %d)", rec.Code)
	}
	if n := protokolliert(); n != 1 {
		t.Errorf("die erteilte Auskunft steht %d-mal im Protokoll, erwartet einmal", n)
	}
}
