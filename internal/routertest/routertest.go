// Package routertest baut für Tests den ganzen Router des Servers mit einem angemeldeten
// Admin-Konto. Recht, CSRF und Middleware hängen am Router, nicht an der Tür; ein Test, der eine
// Tür am Weg der Anfrage prüfen will, nimmt ihn.
//
// Das Paket bindet api/ ein. Ein Test im Paket api oder im Paket eines Themas kann es deshalb
// nicht nutzen, Go weist den Kreis ab; es ist für die Testpakete in den Ordnern der Themen
// (package thema_test). Die deadcode-Baseline führt seine Funktionen mit Begründung.
package routertest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api"
	"bibliothek/db"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MitSitzung legt ein Admin-Konto an, stellt ihm eine Sitzung aus und baut den ganzen Router.
// Die Adresse gehört dem Test; die Sitzung schickt er als Cookie session_token mit.
func MitSitzung(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	return MitSitzungUeber(t, pool, pool, email, vorname, nachname)
}

// MitSitzungUeber ist MitSitzung mit einem eigenen Pool für den Server. Ein Test wickelt damit
// den echten Pool ein und lässt eine einzelne Abfrage scheitern; Konto und Sitzung entstehen am
// echten.
func MitSitzungUeber(t *testing.T, pool *pgxpool.Pool, serverPool db.PgxPoolIface, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	adminID, sitzung, anmeldung := tuertest.AdminSitzung(t, pool, email, vorname, nachname)
	return adminID, sitzung, api.NewServer(&db.Database{Pool: serverPool}, anmeldung, sse.NewBroker(), false).Routes()
}

// MitAufruf legt ein Admin-Konto mit Sitzung an und liefert seine Kennung und einen Aufruf über
// den ganzen Router.
func MitAufruf(t *testing.T, pool *pgxpool.Pool) (adminID string, rufe func(t *testing.T, methode, pfad, rumpf string) *httptest.ResponseRecorder) {
	t.Helper()
	adminID, sitzung, router := MitSitzung(t, pool, "verwaltung-protokoll@example.org", "Vera", "Verwaltung")
	return adminID, tuertest.RouterAufruf(router, sitzung)
}
