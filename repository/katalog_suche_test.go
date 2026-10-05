package repository

import "testing"

func TestMaskiereLikeJoker(t *testing.T) {
	faelle := map[string]string{
		"Harry":      "Harry",
		"100%":       `100\%`,
		"a_b":        `a\_b`,
		`back\slash`: `back\\slash`,
		"%_%":        `\%\_\%`,
	}
	for in, will := range faelle {
		if got := maskiereLikeJoker(in); got != will {
			t.Errorf("maskiereLikeJoker(%q) = %q; want %q", in, got, will)
		}
	}
}
