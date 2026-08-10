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

	// Katalog JA (Betreiber-Entscheidung 30.07.2026): Der Helfer an der Theke muss
	// nachsehen können, ob ein Titel da ist, ohne jede Frage weiterzureichen. Rein
	// lesend — die Grenze zu Personendaten zieht view_students, nicht view_books.
	if a, found := allowed("HELFER", "view_books"); !found || !a {
		t.Errorf("HELFER muss view_books haben (Theken-Auskunft), war allowed=%v found=%v", a, found)
	}

	// Keine Regression: Die operative Rolle behält perform_actions, nachdem die
	// Kiosk-Routen von view_students auf perform_actions umgestellt wurden.
	if a, found := allowed("MITARBEITER", "perform_actions"); !found || !a {
		t.Errorf("MITARBEITER muss perform_actions behalten (sonst Ausleihe kaputt), war allowed=%v found=%v", a, found)
	}

	// Kollegium: NEIN — und das ist die Umkehrung dessen, was hier bis zum 10.08.2026
	// stand. Die Rolle (bis Migration 069 „lehrer") galt damals als operativ und musste
	// perform_actions behalten. Seit 972bf52 ist sie eine reine Portal-Rolle: Eine
	// Lehrkraft meldet sich an, um einen Klassensatz zu reservieren, und das läuft über
	// create_reservations. perform_actions öffnet /api/action — Ausleihe und Rückgabe am
	// Kiosk, im Menü unsichtbar, per URL aber erreichbar.
	//
	// Diese Zeile stand acht Stunden lang falsch herum, ohne dass es auffiel: Der Test
	// hängt an TEST_DATABASE_URL, und der Pre-Push-Hook setzt die Variable nicht — ohne
	// sie überspringt sich der ganze Block STILL. Ein Gate, das sich selbst abschaltet,
	// sobald seine Umgebung fehlt, meldet grün und hat nichts geprüft.
	if a, _ := allowed("KOLLEGIUM", "perform_actions"); a {
		t.Error("KOLLEGIUM darf KEIN perform_actions haben (gäbe Zugriff auf die Kiosk-Endpunkte /api/action)")
	}
	if a, found := allowed("KOLLEGIUM", "create_reservations"); !found || !a {
		t.Errorf("KOLLEGIUM muss create_reservations haben (der Zweck der Portal-Rolle), war allowed=%v found=%v", a, found)
	}
}
