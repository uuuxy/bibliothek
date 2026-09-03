package db

import (
	"context"
	"testing"
)

// TestRechteVererbung_ManageUsersSplit: Eine gewachsene Anlage, die einer Rolle
// manage_users erteilt hatte, behält nach der Aufteilung (24.08.2026) Einstellungen,
// Versetzung, DSGVO-Auskunft und Purge — die beiden neuen Rechte erben den alten Wert.
// Ohne die Vererbung legte der Seed sie mit der Vorgabe false an: still verloren, und
// die Rolle sah stattdessen neu die Benutzerverwaltung im Menü.
func TestRechteVererbung_ManageUsersSplit(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	d := &Database{Pool: pool}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}

	allowed := func(role, perm string) bool {
		var v bool
		if err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE role = $1 AND permission = $2`, role, perm).Scan(&v); err != nil {
			t.Fatalf("%s/%s fehlt: %v", role, perm, err)
		}
		return v
	}

	// Frische Anlage: Vorgabe gilt, Mitarbeiter bekommt die neuen Rechte NICHT.
	if allowed("MITARBEITER", "manage_settings") || allowed("MITARBEITER", "manage_students_admin") {
		t.Fatal("frische Anlage: MITARBEITER darf die neuen Rechte nicht ab Werk haben")
	}

	// Anlage vor dem Split nachstellen: manage_users erteilt, die neuen Zeilen gibt es noch nicht.
	if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = true WHERE role = 'MITARBEITER' AND permission = 'manage_users'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role = 'MITARBEITER' AND permission IN ('manage_settings', 'manage_students_admin')`); err != nil {
		t.Fatal(err)
	}

	// Update = zweiter Start.
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions (Update): %v", err)
	}
	if !allowed("MITARBEITER", "manage_settings") {
		t.Error("MITARBEITER hatte manage_users und verliert nach dem Update still die Einstellungen (manage_settings=false)")
	}
	if !allowed("MITARBEITER", "manage_students_admin") {
		t.Error("MITARBEITER hatte manage_users und verliert nach dem Update still Versetzung/DSGVO (manage_students_admin=false)")
	}
	// Ein erneuter Start ändert nichts mehr (idempotent) und fasst Bestehendes nicht an.
	if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = false WHERE role = 'MITARBEITER' AND permission = 'manage_settings'`); err != nil {
		t.Fatal(err)
	}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatal(err)
	}
	if allowed("MITARBEITER", "manage_settings") {
		t.Error("Vererbung darf eine bewusst gesetzte Zeile nicht überschreiben")
	}
}

// TestRechteVererbung_MergeStudents: Zusammenführen war bis zum 03.09.2026 Teil von
// manage_students_admin. Eine Anlage, die dieses Sonderrecht ans Sekretariat delegiert
// hatte, behält den Knopf nach dem Update — merge_students erbt den alten Wert je Rolle.
func TestRechteVererbung_MergeStudents(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	d := &Database{Pool: pool}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
	allowed := func(role, perm string) bool {
		var v bool
		if err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE role = $1 AND permission = $2`, role, perm).Scan(&v); err != nil {
			t.Fatalf("%s/%s fehlt: %v", role, perm, err)
		}
		return v
	}
	if !allowed("ADMIN", "merge_students") || allowed("MITARBEITER", "merge_students") || allowed("HELFER", "merge_students") {
		t.Fatal("frische Anlage: merge_students ab Werk nur für ADMIN")
	}
	// Anlage vor der Herauslösung: Sonderrecht delegiert, die neue Zeile gibt es noch nicht.
	if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = true WHERE role = 'MITARBEITER' AND permission = 'manage_students_admin'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE permission = 'merge_students'`); err != nil {
		t.Fatal(err)
	}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions (Update): %v", err)
	}
	if !allowed("MITARBEITER", "merge_students") {
		t.Error("MITARBEITER hatte manage_students_admin und verliert nach dem Update still das Zusammenführen")
	}
	if !allowed("ADMIN", "merge_students") || allowed("HELFER", "merge_students") {
		t.Error("Vererbung: ADMIN behält, HELFER bekommt nichts")
	}
}
