package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
)

// Die Rechte-Matrix ist Systemkonfiguration, nicht Tagesgeschäft der Kontoverwaltung.
//
// Vorher genügte manage_users, um sie zu ändern — und damit schloss sich der Kreis:
// Wer Konten verwalten durfte, konnte der EIGENEN Rolle jedes weitere Recht zuschalten.
// Der Test läuft gegen echtes Postgres, weil genau die Trefferzahl der UPDATE-Anweisung
// geprüft wird; ein Mock würde die zurückgeben, die man ihm vorgibt.
func TestRechteMatrixNurDurchAdmin(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS role_permissions (
			role VARCHAR(50) NOT NULL,
			permission VARCHAR(100) NOT NULL,
			allowed BOOLEAN NOT NULL DEFAULT false,
			PRIMARY KEY (role, permission)
		)
	`); err != nil {
		t.Fatalf("role_permissions anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed) VALUES ('MITARBEITER', 'audit_logs', false)
		ON CONFLICT (role, permission) DO UPDATE SET allowed = false
	`); err != nil {
		t.Fatalf("Ausgangszustand setzen: %v", err)
	}
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM role_permissions WHERE role = 'MITARBEITER' AND permission = 'audit_logs'`)
	})

	istErlaubt := func() bool {
		var erlaubt bool
		if err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE role = 'MITARBEITER' AND permission = 'audit_logs'`,
		).Scan(&erlaubt); err != nil {
			t.Fatalf("Zustand lesen: %v", err)
		}
		return erlaubt
	}

	rumpf := `{"role":"mitarbeiter","permission":"audit_logs","allowed":true}`

	t.Run("Mitarbeiter mit manage_users wird abgewiesen", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions", rumpf, "mitarbeiter-1", auth.RoleMitarbeiter)
		w := httptest.NewRecorder()
		srv.UpdatePermissionsHandler().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Status %d, erwartet 403", w.Code)
		}
		if istErlaubt() {
			t.Error("das Recht wurde trotz Ablehnung gesetzt — die Prüfung sitzt hinter dem Schreibzugriff")
		}
	})

	t.Run("Admin darf", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions", rumpf, "admin-1", auth.RoleAdmin)
		w := httptest.NewRecorder()
		srv.UpdatePermissionsHandler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200 (%s)", w.Code, w.Body.String())
		}
		if !istErlaubt() {
			t.Error("der Admin hat das Recht gesetzt, in der Datenbank steht es nicht")
		}
	})

	// Bugklasse „still verworfen, trotzdem 200": Ein Tippfehler im Rechtenamen traf keine
	// Zeile, der Endpunkt meldete aber Erfolg. In der Oberfläche sah das aus wie ein
	// gesetzter Haken, der nach dem nächsten Neuladen wieder weg war.
	t.Run("unbekanntes Recht meldet Fehler statt Erfolg", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions",
			`{"role":"mitarbeiter","permission":"audit_logz","allowed":true}`, "admin-1", auth.RoleAdmin)
		w := httptest.NewRecorder()
		srv.UpdatePermissionsHandler().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400 — ein Tippfehler darf nicht als Erfolg durchgehen", w.Code)
		}
	})

	t.Run("unbekannte Rolle meldet Fehler statt Erfolg", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions",
			`{"role":"hausmeister","permission":"audit_logs","allowed":true}`, "admin-1", auth.RoleAdmin)
		w := httptest.NewRecorder()
		srv.UpdatePermissionsHandler().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400", w.Code)
		}
	})
}
