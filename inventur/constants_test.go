package inventur

import "testing"

func TestConstantsValues(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"routeNotFoundMsg", routeNotFoundMsg, "route nicht gefunden"},
		{"routeClassBooks", routeClassBooks, "/api/admin/class-books"},
		{"langFrench", langFrench, "Französisch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.actual)
			}
		})
	}
}
