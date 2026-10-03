package repository

import (
	"context"
	"slices"
	"testing"
)

// Der Wächter nennt je Löschroutine eine Zeile, in dieser Reihenfolge. Die Selbstprüfung
// zeigt sie so an.
func TestZaehleLoeschRueckstand_EineZeileJeRoutineInFesterReihenfolge(t *testing.T) {
	pool := pgTestPool(t)
	stand, err := NewBetriebszustandRepository(pool).ZaehleLoeschRueckstand(context.Background())
	if err != nil {
		t.Fatalf("ZaehleLoeschRueckstand: %v", err)
	}
	var routinen []string
	for _, z := range stand {
		routinen = append(routinen, z.Routine)
	}
	want := []string{
		"Schüler-Anonymisierung",
		"Abgänger endgültig löschen",
		"Gelöschte Kollegen endgültig löschen",
		"Lesehistorie Schülerbücherei",
		"Lesehistorie Lernmittel",
		"Erledigte Anliegen",
		"Erledigte Klassensatz-Reservierungen",
		"Quittierte Nachbuch-Meldungen",
		"Audit-Aufbewahrung",
	}
	if !slices.Equal(routinen, want) {
		t.Errorf("Routinen %q, erwartet %q", routinen, want)
	}
}
