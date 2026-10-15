package inventur

import "testing"

func TestConstantsValues(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"routeNotFoundMsg", meldungUnbekannteRoute, "route nicht gefunden"},
		{"routeClassBooks", routeKlassensaetze, "/api/admin/class-books"},
		{"langFrench", spracheFranzoesisch, "Französisch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.actual)
			}
		})
	}
}
