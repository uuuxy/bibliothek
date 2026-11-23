package repository

import (
	"slices"
	"testing"
)

// Das Signaturetikett teilt die Aufschrift am ersten Trenner wie die Liste der Regaladressen
// (sqlSignaturRegaladresse): Das Kürzel kann den Trenner selbst tragen („PoWi / Ich").
func TestSignaturZeilen(t *testing.T) {
	for _, fall := range []struct {
		signatur string
		want     []string
	}{
		{"Sk / Bie", []string{"Sk", "Bie"}},
		{"LMF Deu 7 / Bär", []string{"LMF Deu 7", "Bär"}},
		{"PoWi / Ich / Du", []string{"PoWi", "Ich / Du"}},
		{"Sk", []string{"Sk"}},
		{"Sk / ", []string{"Sk"}},
		{"Pa/KL", []string{"Pa/KL"}},
		{"  Ea  ", []string{"Ea"}},
	} {
		if got := SignaturZeilen(fall.signatur); !slices.Equal(got, fall.want) {
			t.Errorf("SignaturZeilen(%q) = %q, erwartet %q", fall.signatur, got, fall.want)
		}
	}
}
