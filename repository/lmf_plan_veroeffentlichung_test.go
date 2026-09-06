package repository

import (
	"reflect"
	"testing"
)

// Die Einstellung der Eingangsjahrgänge wird von Hand getippt („5, 7"); der Leser muss
// Trennzeichen, führende Nullen und Unsinn wegstecken — und darf NIE leer werden, sonst
// bekäme nach den Ferien niemand Bücher und vor den Ferien gäbe niemand „nur zurück".
func TestEingangsjahrgaengeAus(t *testing.T) {
	faelle := []struct {
		in   string
		soll []int
	}{
		{"5, 7", []int{5, 7}},
		{"7;5", []int{5, 7}},
		{"05 07 7", []int{5, 7}},
		{"5, 7, 11", []int{5, 7, 11}},
		{"", []int{5, 7}},
		{"abc, 0, 14", []int{5, 7}},
		{"9", []int{9}},
	}
	for _, f := range faelle {
		if ist := EingangsjahrgaengeAus(f.in); !reflect.DeepEqual(ist, f.soll) {
			t.Errorf("EingangsjahrgaengeAus(%q) = %v, erwartet %v", f.in, ist, f.soll)
		}
	}
}
