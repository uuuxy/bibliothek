package benutzer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Rechte-Matrix ist Systemkonfiguration, nicht Tagesgeschäft der Kontoverwaltung.
//
// Genügte manage_users, um sie zu ändern, könnte, wer Konten verwalten darf, der eigenen Rolle
// jedes weitere Recht zuschalten.
// Der Test läuft gegen echtes Postgres, weil genau die Trefferzahl der UPDATE-Anweisung
// geprüft wird; ein Mock würde die zurückgeben, die man ihm vorgibt.
func TestRechteMatrixNurDurchAdmin(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ohneProtokollRecht(t, pool)

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
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions", rumpf, "mitarbeiter-1", auth.RolleMitarbeiter)
		w := httptest.NewRecorder()
		srv.RollenRechtSetzen().ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Status %d, erwartet 403", w.Code)
		}
		if istErlaubt() {
			t.Error("das Recht wurde trotz Ablehnung gesetzt — die Prüfung sitzt hinter dem Schreibzugriff")
		}
	})

	t.Run("Admin darf", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions", rumpf, "admin-1", auth.RolleAdmin)
		w := httptest.NewRecorder()
		srv.RollenRechtSetzen().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200 (%s)", w.Code, w.Body.String())
		}
		if !istErlaubt() {
			t.Error("der Admin hat das Recht gesetzt, in der Datenbank steht es nicht")
		}
	})

	// „Still verworfen, trotzdem 200": Ein Tippfehler im Namen des Rechts trifft keine Zeile.
	// Meldete die Tür Erfolg, sähe die Oberfläche einen gesetzten Haken, der nach dem nächsten
	// Neuladen wieder weg ist.
	t.Run("unbekanntes Recht meldet Fehler statt Erfolg", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions",
			`{"role":"mitarbeiter","permission":"audit_logz","allowed":true}`, "admin-1", auth.RolleAdmin)
		w := httptest.NewRecorder()
		srv.RollenRechtSetzen().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400 — ein Tippfehler darf nicht als Erfolg durchgehen", w.Code)
		}
	})

	t.Run("unbekannte Rolle meldet Fehler statt Erfolg", func(t *testing.T) {
		req := anfrageAls(t, http.MethodPut, "/api/admin/permissions",
			`{"role":"hausmeister","permission":"audit_logs","allowed":true}`, "admin-1", auth.RolleAdmin)
		w := httptest.NewRecorder()
		srv.RollenRechtSetzen().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status %d, erwartet 400", w.Code)
		}
	})
}

// ohneProtokollRecht legt fest, dass die Rolle MITARBEITER audit_logs nicht hat, und räumt die
// Zeile nach dem Test weg.
func ohneProtokollRecht(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
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
		pgtest.RaeumeAuf(t, pool, `DELETE FROM role_permissions WHERE role = 'MITARBEITER' AND permission = 'audit_logs'`)
	})
}

// Eine Änderung der Matrix gilt mit der nächsten Anfrage: Die Tür leert den Cache der Rechte,
// sonst wirkte die alte Entscheidung bis zu 60 s weiter.
func TestRechteMatrix_AenderungGiltSofort(t *testing.T) {
	pool := pgtest.Pool(t)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	ohneProtokollRecht(t, pool)

	mitarbeiter := anfrageAls(t, http.MethodGet, "/api/admin/auditlog", "", "mitarbeiter-1", auth.RolleMitarbeiter)
	if srv.HatRecht(mitarbeiter, "audit_logs") {
		t.Fatal("HatRecht(mitarbeiter, audit_logs) = true vor der Änderung, erwartet false")
	}

	req := anfrageAls(t, http.MethodPut, "/api/admin/permissions",
		`{"role":"mitarbeiter","permission":"audit_logs","allowed":true}`, "admin-1", auth.RolleAdmin)
	w := httptest.NewRecorder()
	srv.RollenRechtSetzen().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("RollenRechtSetzen: Status %d, erwartet 200 (%s)", w.Code, w.Body.String())
	}

	if !srv.HatRecht(mitarbeiter, "audit_logs") {
		t.Error("HatRecht(mitarbeiter, audit_logs) = false nach der Änderung, erwartet true: die alte Entscheidung kam aus dem Cache")
	}
}
