package inventur

import "testing"

func TestConstantsValues(t *testing.T) {
	faelle := []struct {
		name     string
		actual   string
		expected string
	}{
		{"routeNotFoundMsg", meldungUnbekannteRoute, "Unbekannter Endpunkt"},
		{"routeClassBooks", routeKlassensaetze, "/api/admin/class-books"},
		{"langFrench", spracheFranzoesisch, "Französisch"},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if f.actual != f.expected {
				t.Errorf("expected %q, got %q", f.expected, f.actual)
			}
		})
	}
}
