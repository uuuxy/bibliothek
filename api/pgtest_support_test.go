package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// routerMitSitzung legt ein Admin-Konto an, erzeugt dafür eine Sitzung und baut den GANZEN
// Router. Jeder Test, der eine Tür am Live-Pfad prüfen will, nimmt ihn: Recht, CSRF und
// Middleware hängen am Router, nicht am Handler, und genau dort saßen in diesem Projekt
// schon Fehler, die der nackte Handler nicht zeigte.
//
// Die E-Mail gehört dem Test (ON CONFLICT hält den Aufruf wiederholbar), damit zwei Tests
// sich nicht dasselbe Konto teilen und seine Protokolleinträge zählen.
func routerMitSitzung(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	return routerMitSitzungUeber(t, pool, pool, email, vorname, nachname)
}

// routerMitSitzungUeber ist routerMitSitzung mit einem eigenen Pool für den Server. Ein Test
// wickelt damit den echten Pool ein und lässt eine einzelne Abfrage scheitern; Konto und
// Sitzung entstehen am echten.
func routerMitSitzungUeber(t *testing.T, pool *pgxpool.Pool, serverPool db.PgxPoolIface, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	authenticator, err := auth.NewAuthenticator("pg-test-sitzungsgeheimnis-32-zeichen!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, $3, 'admin', true)
		ON CONFLICT (lower(email)) DO UPDATE SET aktiv = true
		RETURNING id`, vorname, nachname, email).Scan(&adminID); err != nil {
		t.Fatalf("Konto %s anlegen: %v", email, err)
	}
	sitzung, err = authenticator.GenerateToken(adminID, "PG-TEST-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	return adminID, sitzung, NewServer(&db.Database{Pool: serverPool}, authenticator, sse.NewBroker(), false).Routes()
}
