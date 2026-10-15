package tuertest

import (
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetzeGenauEinRecht lässt der Rolle MITARBEITER genau ein Recht und leert den Zwischenspeicher
// der Rechteprüfung. So zeigt jede Antwort, was das Recht ihrer Zeile in der Matrix allein öffnet.
func SetzeGenauEinRecht(t *testing.T, pool *pgxpool.Pool, recht string) {
	t.Helper()
	testdaten.SetzeGenauEinRecht(t, pool, recht)
	tuer.LeereRechteCache()
}
