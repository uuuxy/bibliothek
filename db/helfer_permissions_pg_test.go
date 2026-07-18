package db

import (
	"context"
	"testing"
)

// TestHelferPermissions_KioskJaMahnwesenNein sichert die Helfer-Rechte ab: Die
// Kiosk-Rolle darf am Terminal arbeiten (perform_actions: Ausleihe/Rückgabe/Scan/Suche),
// aber NICHT auf Schülerlisten und Mahnwesen/Bulk-Mahndruck zugreifen (view_students).
// Vor dem Fix waren ALLE Helfer-Rechte false — die Rolle war unbenutzbar, ein Helfer
// konnte nicht einmal die Kiosk-Ausleihe durchführen, für die es die Rolle gibt.
func TestHelferPermissions_KioskJaMahnwesenNein(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	d := &Database{Pool: pool}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}

	allowed := func(role, perm string) (val bool, found bool) {
		err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE UPPER(role) = UPPER($1) AND permission = $2`,
			role, perm).Scan(&val)
		return val, err == nil
	}

	// Helfer: Kiosk JA.
	if a, found := allowed("HELFER", "perform_actions"); !found || !a {
		t.Errorf("HELFER muss perform_actions haben (Kiosk-Kernfunktion), war allowed=%v found=%v", a, found)
	}
	// Helfer: Mahnwesen/Schülerdatei NEIN. view_students gated den Bulk-Mahndruck
	// (= Mahnstufen-Eskalation) und die Schülerlisten — nichts für einen Kiosk-Helfer.
	if a, _ := allowed("HELFER", "view_students"); a {
		t.Error("HELFER darf KEIN view_students haben (gäbe Zugriff auf Mahnwesen/Bulk-Mahndruck)")
	}
	if a, _ := allowed("HELFER", "manage_users"); a {
		t.Error("HELFER darf KEIN manage_users haben")
	}

	// Keine Regression: Die anderen operativen Rollen behalten perform_actions, nachdem
	// die Kiosk-Routen von view_students auf perform_actions umgestellt wurden.
	for _, role := range []string{"LEHRER", "MITARBEITER"} {
		if a, found := allowed(role, "perform_actions"); !found || !a {
			t.Errorf("%s muss perform_actions behalten (sonst Ausleihe kaputt), war allowed=%v found=%v", role, a, found)
		}
	}
}
