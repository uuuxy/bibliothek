package db

import (
	"context"
	"testing"
)

// Eine gewachsene Anlage, die einer Rolle manage_users erteilt hat, behält Einstellungen,
// Versetzung, Auskunft und Papierkorb: Die beiden Rechte, die daraus hervorgingen, erben den
// Wert. Ohne das Erbe legte SchreibeRechteVorgabe sie mit der Vorgabe false an, und die Rolle
// verlöre sie still.
func TestRechteVererbung_AufteilungDesRechtsBenutzerVerwalten(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	d := &Verbindung{Pool: pool}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
	}

	allowed := func(role, perm string) bool {
		var v bool
		if err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE role = $1 AND permission = $2`, role, perm).Scan(&v); err != nil {
			t.Fatalf("%s/%s fehlt: %v", role, perm, err)
		}
		return v
	}

	// Frische Anlage: Die Vorgabe gilt, MITARBEITER bekommt die beiden Rechte nicht.
	if allowed("MITARBEITER", "manage_settings") || allowed("MITARBEITER", "manage_students_admin") {
		t.Fatal("frische Anlage: MITARBEITER darf die neuen Rechte nicht ab Werk haben")
	}

	// Eine Anlage vor der Aufteilung: manage_users erteilt, die beiden Zeilen fehlen noch.
	if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = true WHERE role = 'MITARBEITER' AND permission = 'manage_users'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role = 'MITARBEITER' AND permission IN ('manage_settings', 'manage_students_admin')`); err != nil {
		t.Fatal(err)
	}

	// Das Update ist der zweite Start.
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe (Update): %v", err)
	}
	if !allowed("MITARBEITER", "manage_settings") {
		t.Error("MITARBEITER hatte manage_users und verliert nach dem Update still die Einstellungen (manage_settings=false)")
	}
	if !allowed("MITARBEITER", "manage_students_admin") {
		t.Error("MITARBEITER hatte manage_users und verliert nach dem Update still Versetzung/DSGVO (manage_students_admin=false)")
	}
	// Ein weiterer Start ändert nichts und fasst Bestehendes nicht an.
	if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = false WHERE role = 'MITARBEITER' AND permission = 'manage_settings'`); err != nil {
		t.Fatal(err)
	}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatal(err)
	}
	if allowed("MITARBEITER", "manage_settings") {
		t.Error("Vererbung darf eine bewusst gesetzte Zeile nicht überschreiben")
	}
}

// Das Zusammenführen zweier Datensätze hat ein eigenes Recht (merge_students), das den Wert von
// manage_students_admin erbt: Eine Anlage, die dieses Sonderrecht ans Sekretariat abgegeben hat,
// behält den Knopf nach dem Update.
func TestRechteVererbung_Zusammenfuehren(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	d := &Verbindung{Pool: pool}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
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
	// Eine Anlage, die das Sonderrecht abgegeben hat; die Zeile für merge_students fehlt noch.
	if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = true WHERE role = 'MITARBEITER' AND permission = 'manage_students_admin'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE permission = 'merge_students'`); err != nil {
		t.Fatal(err)
	}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe (Update): %v", err)
	}
	if !allowed("MITARBEITER", "merge_students") {
		t.Error("MITARBEITER hatte manage_students_admin und verliert nach dem Update still das Zusammenführen")
	}
	if !allowed("ADMIN", "merge_students") || allowed("HELFER", "merge_students") {
		t.Error("Vererbung: ADMIN behält, HELFER bekommt nichts")
	}
}
