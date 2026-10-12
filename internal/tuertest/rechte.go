package tuertest

import (
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetzeGenauEinRecht stellt die MITARBEITER-Rolle auf exakt ein erlaubtes Recht.
// So zeigt jede Antwort, was das Recht ihrer Matrix-Zeile ALLEIN öffnet.
func SetzeGenauEinRecht(t *testing.T, pool *pgxpool.Pool, recht string) {
	t.Helper()
	testdaten.GenauEinRecht(t, pool, recht)
	tuer.InvalidatePermissionCache()
}
