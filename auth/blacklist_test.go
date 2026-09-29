package auth

import "testing"

// TestHashToken: revoked_tokens hält widerrufene Tokens als SHA-256 in Hex, und
// IsBlacklisted vergleicht mit demselben Wert. Änderte ein Update die Form, fände
// IsBlacklisted die vorher widerrufenen Tokens nicht mehr — sie wären bis zu ihrem Ablauf
// (höchstens zwölf Stunden) wieder gültig, ohne dass ein anderer Test rot wird. Die Werte
// sind die bekannten SHA-256-Prüfsummen der Eingaben.
func TestHashToken(t *testing.T) {
	faelle := []struct {
		name, token, erwartet string
	}{
		{"leer", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"kurz", "test_token", "cc0af97287543b65da2c7e1476426021826cab166f1e063ed012b855ff819656"},
		// Form eines JWT: Kopf {"alg":"HS256"}, Inhalt {"sub":"konto"}, Signatur.
		{
			"JWT-Form",
			"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJrb250byJ9.c2lnbmF0dXI",
			"c27154c4e606a8f5df52e425838f15480b0627f346cd09d569016342f2e778ad",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := hashToken(f.token); got != f.erwartet {
				t.Errorf("hashToken(%q) = %s, erwartet %s", f.token, got, f.erwartet)
			}
		})
	}
}
