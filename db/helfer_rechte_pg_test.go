package db

import (
	"context"
	"testing"
)

// Der Helfer arbeitet an der Theke (perform_actions: Ausleihe, Rückgabe, Scan, Suche), sieht aber
// keine Schülerlisten und kein Mahnwesen (view_students). Ohne perform_actions wäre die Rolle
// unbenutzbar: Nicht einmal die Ausleihe ginge, für die es sie gibt.
func TestHelferRechte_ThekeJaMahnwesenNein(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	d := &Verbindung{Pool: pool}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
	}

	allowed := func(role, perm string) (val bool, found bool) {
		err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE UPPER(role) = UPPER($1) AND permission = $2`,
			role, perm).Scan(&val)
		return val, err == nil
	}

	// Helfer: Theke ja.
	if a, found := allowed("HELFER", "perform_actions"); !found || !a {
		t.Errorf("HELFER muss perform_actions haben (Kiosk-Kernfunktion), war allowed=%v found=%v", a, found)
	}
	// Helfer: Mahnwesen und Schülerdatei nein. view_students öffnet den Druck vieler Mahnungen
	// auf einmal, der die Mahnstufe erhöht, und die Schülerlisten.
	if a, _ := allowed("HELFER", "view_students"); a {
		t.Error("HELFER darf KEIN view_students haben (gäbe Zugriff auf Mahnwesen/Bulk-Mahndruck)")
	}
	if a, _ := allowed("HELFER", "manage_users"); a {
		t.Error("HELFER darf KEIN manage_users haben")
	}

	// Katalog ja: Der Helfer an der Theke sieht nach, ob ein Titel da ist, ohne jede Frage
	// weiterzureichen. Das Recht liest nur; die Grenze zu Personendaten zieht view_students.
	if a, found := allowed("HELFER", "view_books"); !found || !a {
		t.Errorf("HELFER muss view_books haben (Theken-Auskunft), war allowed=%v found=%v", a, found)
	}

	// Die Rolle MITARBEITER braucht perform_actions für die Routen der Theke.
	if a, found := allowed("MITARBEITER", "perform_actions"); !found || !a {
		t.Errorf("MITARBEITER muss perform_actions behalten (sonst Ausleihe kaputt), war allowed=%v found=%v", a, found)
	}

	// Kollegium nein: Eine Lehrkraft meldet sich an, um einen Klassensatz zu reservieren, und das
	// läuft über create_reservations. perform_actions öffnete /api/action, Ausleihe und Rückgabe
	// an der Theke, im Menü unsichtbar, über die Adresse aber erreichbar. Dass dieser Test in CI
	// nicht still übersprungen wird, hält TestDBTestsLaufenInCI.
	if a, _ := allowed("KOLLEGIUM", "perform_actions"); a {
		t.Error("KOLLEGIUM darf KEIN perform_actions haben (gäbe Zugriff auf die Kiosk-Endpunkte /api/action)")
	}
	if a, found := allowed("KOLLEGIUM", "create_reservations"); !found || !a {
		t.Errorf("KOLLEGIUM muss create_reservations haben (der Zweck der Portal-Rolle), war allowed=%v found=%v", a, found)
	}
}
