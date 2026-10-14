package testdaten

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetzeGenauEinRecht stellt die MITARBEITER-Rolle in role_permissions auf exakt ein erlaubtes
// Recht; ein leeres Recht nimmt ihr alle. Den Zwischenspeicher der Rechteprüfung leert es
// nicht: Tests von Türen rufen tuertest.SetzeGenauEinRecht.
func SetzeGenauEinRecht(t *testing.T, pool *pgxpool.Pool, recht string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role = 'MITARBEITER'`); err != nil {
		t.Fatalf("Rechte leeren: %v", err)
	}
	if recht != "" {
		if _, err := pool.Exec(ctx, `
			INSERT INTO role_permissions (role, permission, allowed) VALUES ('MITARBEITER', $1, true)`,
			recht); err != nil {
			t.Fatalf("Recht %q setzen: %v", recht, err)
		}
	}
}
