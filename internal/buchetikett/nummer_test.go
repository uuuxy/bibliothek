package buchetikett

import (
	"slices"
	"testing"
)

// Kein Etikett trägt eine LMF-Nummer; jede andere Form schon, auch die 13 Ziffern eines
// Littera-Etiketts und eine von Hand eingetragene Nummer.
func TestIstDruckbar(t *testing.T) {
	faelle := []struct {
		nummer string
		want   bool
	}{
		{"B-10001", true},
		{"1016340039562", true},
		{"101634", true},
		{"SYS-100001", true},
		{"LMF-9783060158423-001", false},
		{"LMF-", false},
		// Eine Signatur heißt so („LMF-Deutsch 5"), eine Nummer nur mit der Vorsilbe am Anfang.
		{"X-LMF-1", true},
	}
	for _, f := range faelle {
		if got := IstDruckbar(f.nummer); got != f.want {
			t.Errorf("IstDruckbar(%q) = %v, erwartet %v", f.nummer, got, f.want)
		}
	}
}

// Jede abgelehnte Nummer steht einmal in der Liste, in der Reihenfolge ihres ersten Vorkommens.
func TestNichtDruckbare(t *testing.T) {
	got := NichtDruckbare([]string{"B-1", "LMF-2", "1016340039562", "LMF-1", "LMF-2"})
	if want := []string{"LMF-2", "LMF-1"}; !slices.Equal(got, want) {
		t.Errorf("NichtDruckbare = %q, erwartet %q", got, want)
	}
	if got := NichtDruckbare([]string{"B-1"}); len(got) != 0 {
		t.Errorf("NichtDruckbare ohne LMF-Nummer = %q, erwartet keine", got)
	}
}
