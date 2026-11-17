package bereitschaft

import (
	"testing"
	"time"
)

func TestStundenText(t *testing.T) {
	for _, fall := range []struct {
		alter time.Duration
		text  string
	}{
		{90 * time.Minute, "1 Stunde"},
		{30 * time.Minute, "0 Stunden"},
		{30 * time.Hour, "30 Stunden"},
	} {
		if got := stundenText(fall.alter); got != fall.text {
			t.Errorf("stundenText(%v) = %q, want %q", fall.alter, got, fall.text)
		}
	}
}
