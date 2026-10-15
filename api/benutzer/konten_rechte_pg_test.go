package benutzer

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
	"bibliothek/repository"
)

// TestKonten_LiefernEchteRechte: GET /api/benutzer nennt die Rechte aus role_permissions wie
// die Anmeldung und /api/auth/me, keine feste Liste. Eine feste Liste nannte manage_settings,
// print_classes und view_media, Namen, die es im Rechtesystem nicht gibt.
//
// Der Test prüft beides: dass diese Namen fehlen und dass ein vergebenes Recht ankommt.
func TestKonten_LiefernEchteRechte(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	email := "rechte-" + suffix + "@example.org"

	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE email = $1`, email)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Rechte', 'Pruefer', $1, 'mitarbeiter', true)
	`, email); err != nil {
		t.Fatalf("Benutzer anlegen: %v", err)
	}

	// Ein Recht, von dem wir wissen, dass es echt ist und der Rolle gehört.
	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed)
		VALUES ('MITARBEITER', 'view_books', true)
		ON CONFLICT (role, permission) DO UPDATE SET allowed = true
	`); err != nil {
		t.Fatalf("Recht setzen: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.Konten(repository.NewKontoAbfragen(pool)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/benutzer", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d, Koerper: %s", rec.Code, rec.Body.String())
	}

	var benutzer []kontoAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &benutzer); err != nil {
		t.Fatalf("Antwort unlesbar: %v", err)
	}

	var unserer *kontoAntwort
	for i := range benutzer {
		if benutzer[i].Email == email {
			unserer = &benutzer[i]
		}
	}
	if unserer == nil {
		t.Fatal("angelegter Benutzer fehlt in der Liste")
	}

	// Die erfundenen Namen dürfen nirgends auftauchen, bei keinem Konto.
	erfunden := map[string]bool{"manage_settings": true, "print_classes": true, "view_media": true}
	for _, b := range benutzer {
		for _, p := range b.Rechte {
			if erfunden[p] {
				t.Errorf("erfundenes Recht %q in der Benutzerliste (Rolle %s)", p, b.Rolle)
			}
		}
	}

	// Und das echte Recht muss ankommen.
	if !enthaelt(unserer.Rechte, "view_books") {
		t.Errorf("echtes Recht view_books fehlt, geliefert wurde: %v", unserer.Rechte)
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
