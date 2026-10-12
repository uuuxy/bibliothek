package api

import (
	"net/http"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// routerMitSitzung legt ein Admin-Konto an, erzeugt dafür eine Sitzung und baut den ganzen
// Router. Jeder Test, der eine Tür am Live-Pfad prüfen will, nimmt ihn: Recht, CSRF und
// Middleware hängen am Router, nicht am Handler.
//
// Konto und Sitzung legt tuertest.AdminSitzung an; die E-Mail gehört dem Test.
func routerMitSitzung(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	return routerMitSitzungUeber(t, pool, pool, email, vorname, nachname)
}

// routerMitSitzungUeber ist routerMitSitzung mit einem eigenen Pool für den Server. Ein Test
// wickelt damit den echten Pool ein und lässt eine einzelne Abfrage scheitern; Konto und
// Sitzung entstehen am echten.
func routerMitSitzungUeber(t *testing.T, pool *pgxpool.Pool, serverPool db.PgxPoolIface, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	adminID, sitzung, authenticator := tuertest.AdminSitzung(t, pool, email, vorname, nachname)
	return adminID, sitzung, NewServer(&db.Database{Pool: serverPool}, authenticator, sse.NewBroker(), false).Routes()
}
