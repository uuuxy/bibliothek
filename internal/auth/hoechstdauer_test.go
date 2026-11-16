package auth

import (
	"strings"
	"testing"
)

// Die Meldung nennt die Höchstdauer, die gilt; die Zahl steht dort als Text.
func TestMeldungHoechstdauerErreicht_NenntDieHoechstdauer(t *testing.T) {
	stunden := strings.TrimSuffix(hoechstdauerNachPasswort.String(), "h0m0s")
	if !strings.Contains(MeldungHoechstdauerErreicht, " "+stunden+" Stunden ") {
		t.Errorf("MeldungHoechstdauerErreicht = %q, erwartet „%s Stunden“ wie hoechstdauerNachPasswort",
			MeldungHoechstdauerErreicht, stunden)
	}
}
