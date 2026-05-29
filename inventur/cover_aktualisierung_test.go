package inventur

import "testing"

func TestFallbackString(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{
			name:     "non-empty value, empty fallback",
			value:    "primary",
			fallback: "",
			want:     "primary",
		},
		{
			name:     "empty value, non-empty fallback",
			value:    "",
			fallback: "secondary",
			want:     "secondary",
		},
		{
			name:     "both non-empty",
			value:    "primary",
			fallback: "secondary",
			want:     "primary",
		},
		{
			name:     "both empty",
			value:    "",
			fallback: "",
			want:     "",
		},
		{
			name:     "whitespace value is considered non-empty",
			value:    "   ",
			fallback: "secondary",
			want:     "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fallbackString(tt.value, tt.fallback); got != tt.want {
				t.Errorf("fallbackString() = %v, want %v", got, tt.want)
			}
		})
	}
}
