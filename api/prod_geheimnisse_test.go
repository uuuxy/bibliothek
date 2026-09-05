package api

import "testing"

// Die Vorgabe ist die sichere Richtung: Nur eine ausdrückliche „false"-Zeile oder eine
// Spielwiesen-Umgebung ohne gesetzte Variable schaltet die Absicherung aus. Mit der
// alten Regel (`== "true"`) sind die drei production-Zeilen rot (rot gesehen).
func TestErzwingeProdGeheimnisse(t *testing.T) {
	faelle := []struct {
		appEnv, roh string
		will        bool
	}{
		{"production", "", true},
		{"production", "true", true},
		{"production", "false", false},
		{"production", "ja", true},
		{"", "", true},
		{"local", "", false},
		{"development", "", false},
		{"test", "", false},
		{"local", "true", true},
		{"local", "false", false},
	}
	for _, f := range faelle {
		if got := ErzwingeProdGeheimnisse(f.appEnv, f.roh); got != f.will {
			t.Errorf("ErzwingeProdGeheimnisse(%q, %q) = %v; want %v", f.appEnv, f.roh, got, f.will)
		}
	}
}
