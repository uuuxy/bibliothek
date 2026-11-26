package db

import (
	"context"
	"testing"
)

// beginneMitFrischenRechten stellt die Rechte, um die es geht, auf die frische Anlage, vor dem Test und
// nach ihm: Die Tests ändern sie, und ein anderer Test läse sonst diesen Stand als Vorgabe.
func beginneMitFrischenRechten(t *testing.T, d *Verbindung) {
	t.Helper()
	ctx := context.Background()
	setze := func() {
		if _, err := d.Pool.Exec(ctx, `DELETE FROM role_permissions WHERE permission = ANY($1)`,
			[]string{"manage_users", "manage_settings", "manage_students_admin", "merge_students"}); err != nil {
			t.Fatalf("Rechte entfernen: %v", err)
		}
		if err := d.SchreibeRechteVorgabe(ctx); err != nil {
			t.Fatalf("SchreibeRechteVorgabe: %v", err)
		}
	}
	setze()
	t.Cleanup(setze)
}

// Eine gewachsene Anlage, die einer Rolle manage_users erteilt hat, behält Einstellungen,
// Versetzung, Auskunft und Papierkorb: Die beiden Rechte, die daraus hervorgingen, erben den
// Wert. Ohne das Erbe legte SchreibeRechteVorgabe sie mit der Vorgabe false an, und die Rolle
// verlöre sie still.
func TestRechteVererbung_AufteilungDesRechtsBenutzerVerwalten(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	d := &Verbindung{Pool: pool}
	beginneMitFrischenRechten(t, d)

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
	beginneMitFrischenRechten(t, d)
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

// Mahnen, Forderungen und Verlängern haben eigene Rechte, die je Rolle den Wert ihres Ursprungs
// erben, in beide Richtungen: Eine Rolle, der der Admin den Ursprung genommen hat, bekommt die
// Aktion beim Update nicht dazu, und eine, der er ihn gegeben hat, verliert sie nicht.
func TestRechteVererbung_AbgespalteneRechte(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	d := &Verbindung{Pool: pool}
	ursprungVon := map[string]string{"send_dunning": "create_orders", "manage_claims": "edit_students", "extend_loans": "edit_books"}
	betroffen := []string{"send_dunning", "manage_claims", "extend_loans", "create_orders", "edit_students", "edit_books"}
	setze := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE permission = ANY($1)`, betroffen); err != nil {
			t.Fatalf("Rechte entfernen: %v", err)
		}
		if err := d.SchreibeRechteVorgabe(ctx); err != nil {
			t.Fatalf("SchreibeRechteVorgabe: %v", err)
		}
	}
	setze()
	t.Cleanup(setze)
	allowed := func(role, perm string) bool {
		var v bool
		if err := pool.QueryRow(ctx,
			`SELECT allowed FROM role_permissions WHERE role = $1 AND permission = $2`, role, perm).Scan(&v); err != nil {
			t.Fatalf("%s/%s fehlt: %v", role, perm, err)
		}
		return v
	}

	// Frische Anlage: Das neue Recht hat, wer das Ursprungsrecht hat.
	for neu, ursprung := range ursprungVon {
		for _, rolle := range []string{"ADMIN", "LEITUNG", "MITARBEITER", "KOLLEGIUM", "HELFER"} {
			if allowed(rolle, neu) != allowed(rolle, ursprung) {
				t.Errorf("frische Anlage: %s/%s = %v, aber %s = %v", rolle, neu, allowed(rolle, neu), ursprung, allowed(rolle, ursprung))
			}
		}
	}

	// Eine Anlage vor der Aufteilung: Den Mitarbeitern sind die Ursprungsrechte genommen, dem
	// Helfer gegeben; die Zeilen der neuen Rechte fehlen noch.
	for _, ursprung := range ursprungVon {
		if _, err := pool.Exec(ctx, `UPDATE role_permissions SET allowed = (role = 'HELFER')
			WHERE role IN ('MITARBEITER', 'HELFER') AND permission = $1`, ursprung); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE permission IN ('send_dunning', 'manage_claims', 'extend_loans')`); err != nil {
		t.Fatal(err)
	}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe (Update): %v", err)
	}
	for neu, ursprung := range ursprungVon {
		if allowed("MITARBEITER", neu) {
			t.Errorf("MITARBEITER hatte %s nicht und bekommt nach dem Update %s dazu", ursprung, neu)
		}
		if !allowed("HELFER", neu) {
			t.Errorf("HELFER hatte %s und verliert nach dem Update %s", ursprung, neu)
		}
	}
}
