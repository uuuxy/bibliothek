package betrag

import "testing"

func TestText(t *testing.T) {
	for _, f := range []struct {
		betrag float64
		want   string
	}{
		{12.5, "12,50"},
		{0, "0,00"},
		{19.9, "19,90"},
		{1234.5, "1234,50"},
		{-3.2, "-3,20"},
		// Kaufmännisch: der halbe Cent rundet auf, auch wo die Zahl sich binär genau schreiben lässt.
		{0.125, "0,13"},
		{2.675, "2,68"},
		// Knapp unter null bleibt null, ohne Vorzeichen.
		{-0.001, "0,00"},
	} {
		if got := Text(f.betrag); got != f.want {
			t.Errorf("Text(%v) = %q, erwartet %q", f.betrag, got, f.want)
		}
	}
}

func TestEuro(t *testing.T) {
	if got := Euro(41.5); got != "41,50 €" {
		t.Errorf("Euro(41.5) = %q, erwartet „41,50 €“", got)
	}
}
