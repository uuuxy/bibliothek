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
	"bibliothek/repository"
)

// TestListUsersLiefertEchteRechte sichert Audit-Punkt 8 ab.
//
// GET /api/benutzer meldete die Rechte aus einer hartkodierten Liste, die sich
// "analog zum Login" nannte, es aber nicht war: Sie nannte manage_settings,
// print_classes und view_media — Namen, die es im Rechtesystem ueberhaupt nicht gibt
// (kein RequirePermission, kein Seed). Wer die Benutzerverwaltung ansah, bekam also
// erfundene Angaben, waehrend Login und /api/auth/me die echten Rechte laden.
//
// Der Test prueft beides: dass die erfundenen Namen verschwunden sind UND dass ein
// tatsaechlich vergebenes Recht ankommt.
func TestListUsersLiefertEchteRechte(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	email := "rechte-" + suffix + "@example.org"

	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM benutzer WHERE email = $1`, email)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO benutzer (barcode_id, vorname, nachname, email, rolle, aktiv)
		VALUES ($1, 'Rechte', 'Pruefer', $2, 'mitarbeiter', true)
	`, "RECHT-"+suffix, email); err != nil {
		t.Fatalf("Benutzer anlegen: %v", err)
	}

	// role_permissions wird NICHT von schema.sql angelegt, sondern zur Laufzeit von
	// db/seed.go (CREATE TABLE IF NOT EXISTS beim Serverstart). Die Test-Datenbank kennt
	// sie deshalb nicht — dieselbe Definition hier, damit der Test eigenstaendig laeuft.
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

	// Ein Recht, von dem wir wissen, dass es echt ist und der Rolle gehoert.
	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed)
		VALUES ('MITARBEITER', 'view_books', true)
		ON CONFLICT (role, permission) DO UPDATE SET allowed = true
	`); err != nil {
		t.Fatalf("Recht setzen: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.ListUsersHandler(repository.NewUserRepository(pool)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/benutzer", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, Koerper: %s", rec.Code, rec.Body.String())
	}

	var benutzer []UserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &benutzer); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}

	var unserer *UserResponse
	for i := range benutzer {
		if benutzer[i].Email == email {
			unserer = &benutzer[i]
		}
	}
	if unserer == nil {
		t.Fatal("angelegter Benutzer fehlt in der Liste")
	}

	// Die erfundenen Namen duerfen nirgends mehr auftauchen — bei KEINEM Benutzer.
	erfunden := map[string]bool{"manage_settings": true, "print_classes": true, "view_media": true}
	for _, b := range benutzer {
		for _, p := range b.Permissions {
			if erfunden[p] {
				t.Errorf("erfundenes Recht %q in der Benutzerliste (Rolle %s)", p, b.Rolle)
			}
		}
	}

	// Und das echte Recht muss ankommen.
	if !enthaelt(unserer.Permissions, "view_books") {
		t.Errorf("echtes Recht view_books fehlt, geliefert wurde: %v", unserer.Permissions)
	}
}

func enthaelt(werte []string, gesucht string) bool {
	for _, w := range werte {
		if w == gesucht {
			return true
		}
	}
	return false
}
