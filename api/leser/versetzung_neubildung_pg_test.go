package leser

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"

	"github.com/google/go-cmp/cmp"
)

// Die Klassenleitung rückt mit der Versetzung eine Stufe hoch, außer von 6 nach 7 und von
// 10 in die Oberstufe. Dort bildet die Schule die Klassen neu, und die neuen Klassen bekommen
// neue Klassenleitungen. Zählte die Versetzung auch diese Zuordnungen hoch, entstünden
// „06F1" → „07F1", „06G1" → „07G1", „10G1" → „11G1", jeweils mit der alten Lehrkraft.
func TestVersetzungKlassenleitungEntfaelltBeiNeubildung(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	testdaten.LeereBestandsdaten(t, pool)
	if _, err := pool.Exec(ctx, `DELETE FROM klassen_lehrer_mapping`); err != nil {
		t.Fatalf("Mapping leeren: %v", err)
	}
	// Ein Schüler, damit der Lauf etwas zu versetzen hat.
	testdaten.LegeSchuelerMitNamenAn(t, pool, "NB-S-1", "Nora", "Neubildung", "05F1", 2034)
	for klasse, mail := range map[string]string{
		"05F1": "a@schule.example", // 5 → 6: rückt hoch
		"06F1": "b@schule.example", // Förderstufe → Zweige: entfällt
		"06G1": "c@schule.example", // 6 → 7 auch im Gymnasialzweig: entfällt
		"09G1": "d@schule.example", // 9 → 10: rückt auf den frei gewordenen Namen
		"10G1": "e@schule.example", // 10 → Oberstufe: entfällt
		"10R1": "f@schule.example", // Abschlussklasse: entfällt
		"12T1": "g@schule.example", // innerhalb der Oberstufe: rückt hoch
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ($1, $2)`, klasse, mail); err != nil {
			t.Fatalf("Mapping %s: %v", klasse, err)
		}
	}

	echt := versetzungAusfuehren(t, pool, false)
	if echt.ZuordnungenVersetzt != 3 || echt.ZuordnungenEntfernt != 4 || len(echt.Zuordnungskonflikte) != 0 {
		t.Fatalf("Lauf: erwartet 3 hochgerückt, 4 entfallen, keine Konflikte — %+v", echt)
	}

	rows, err := pool.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
	if err != nil {
		t.Fatalf("Zuordnungen lesen: %v", err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var klasse, mail string
		if err := rows.Scan(&klasse, &mail); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		got[klasse] = mail
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Zuordnungen lesen: %v", err)
	}
	erwartet := map[string]string{
		"06F1": "a@schule.example",
		"10G1": "d@schule.example",
		"13T1": "g@schule.example",
	}
	if !cmp.Equal(got, erwartet) {
		t.Errorf("Zuordnungen nach der Versetzung = %v, erwartet %v", got, erwartet)
	}
}
