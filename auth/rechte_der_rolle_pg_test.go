package auth

import (
	"context"
	"slices"
	"testing"
)

// Die Rechte, die Anmeldung und /me an den Browser schicken, an der echten Tabelle: nur die
// erteilten, nur die der eigenen Rolle, ohne Rücksicht auf die Schreibweise der Rolle. Ohne
// erteiltes Recht ist die Liste leer und nicht nil, sonst stünde im JSON null.
func TestLoadPermissionsForRole_NurErteilteRechteDerRolle(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM role_permissions WHERE role IN ('PROBEROLLE', 'NACHBARROLLE')`); err != nil {
			t.Errorf("Aufräumen role_permissions: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission, allowed) VALUES
			('PROBEROLLE', 'recht_erteilt', true),
			('PROBEROLLE', 'recht_entzogen', false),
			('NACHBARROLLE', 'recht_der_nachbarrolle', true)`); err != nil {
		t.Fatalf("Rechte anlegen: %v", err)
	}

	// Das Konto trägt die Rolle klein geschrieben, die Tabelle groß.
	rechte, err := loadPermissionsForRole(ctx, pool, "proberolle")
	if err != nil {
		t.Fatalf("loadPermissionsForRole: %v", err)
	}
	if !slices.Equal(rechte, []string{"recht_erteilt"}) {
		t.Errorf("Rechte = %v, erwartet nur das erteilte Recht der Rolle", rechte)
	}

	keine, err := loadPermissionsForRole(ctx, pool, "rolle_ohne_zeile")
	if err != nil {
		t.Fatalf("loadPermissionsForRole ohne Zeile: %v", err)
	}
	if keine == nil || len(keine) != 0 {
		t.Errorf("Rechte einer Rolle ohne Zeile = %#v, erwartet eine leere Liste", keine)
	}
}
