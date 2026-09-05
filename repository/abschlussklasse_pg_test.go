package repository

import (
	"context"
	"testing"
)

// Die eine Abschlussregel, Klasse für Klasse am echten Postgres — denn sie ist SQL, und
// ein Go-Zwilling bewiese nur sich selbst. Was hier steht, ist die Wahrheit, gegen die
// Versetzung und Abgängerliste laufen.
func TestAbschlussklasseSQL_Regel(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	faelle := []struct {
		klasse    string
		abschluss bool
		warum     string
	}{
		{"9H1", true, "Hauptschulzweig endet mit 9"},
		{"09h2", true, "führende Null und Kleinschreibung ändern nichts"},
		{"9H", true, "ohne Zugnummer"},
		{"10H1", true, "freiwilliges 10. Hauptschuljahr — geht danach ebenfalls"},
		{"10R1", true, "Realschulzweig endet mit 10"},
		{"10R3", true, "dritter Zug"},
		{"13", true, "Oberstufe ohne Zweigbuchstaben"},
		{"13G2", true, "Gymnasialzweig endet in der Oberstufe"},
		{"10G1", false, "Gymnasialzweig bleibt über 10 hinaus"},
		{"8H1", false, "Hauptschulzweig, ein Jahr vor dem Ende"},
		{"9R1", false, "Realschulzweig, ein Jahr vor dem Ende"},
		{"5F1", false, "Förderstufe"},
		{"6G2", false, "Förderstufe im Gymnasialzweig"},
		{"12", false, "Q1/Q2 — noch nicht am Ende"},
		{"E1", false, "Einführungsphase ohne Jahrgangsziffer — die Schule führt die Oberstufe als 13"},
		{"Q4", false, "Q-Phase ohne Jahrgangsziffer"},
		{"ABG", false, "die Konvention der schon Abgegangenen"},
		{"", false, "leer"},
	}
	for _, f := range faelle {
		var ist bool
		if err := pool.QueryRow(ctx, "SELECT "+AbschlussklasseSQL("$1::text"), f.klasse).Scan(&ist); err != nil {
			t.Fatalf("Klasse %q: %v", f.klasse, err)
		}
		if ist != f.abschluss {
			t.Errorf("Klasse %q: Abschlussklasse=%v, erwartet %v (%s)", f.klasse, ist, f.abschluss, f.warum)
		}
	}
}
