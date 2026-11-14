package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Alle Postfächer des Kollegiums liegen auf derselben Domain: „mueller" muss reichen.
// Vollständige Adressen bleiben unangetastet, damit Empfänger und Konten anderer Domains
// möglich bleiben.
func TestErgaenzeSchulDomain(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "schule.example")

	tests := []struct{ eingabe, want string }{
		{"mueller", "mueller@schule.example"},
		{"m.mueller", "m.mueller@schule.example"},
		{"extern@schulamt.hessen.de", "extern@schulamt.hessen.de"}, // vollständig → unverändert
		{"", ""},
	}
	for _, tt := range tests {
		if got := ErgaenzeSchulDomain(tt.eingabe); got != tt.want {
			t.Errorf("ErgaenzeSchulDomain(%q) = %q, erwartet %q", tt.eingabe, got, tt.want)
		}
	}
}

// Ohne eingetragene Domain wird nicht geraten: Die Eingabe bleibt und scheitert an der
// Adressprüfung oder am Konto mit einer klaren Meldung.
func TestErgaenzeSchulDomain_OhneDomainKeineRaterei(t *testing.T) {
	t.Setenv(selbstanmeldeDomainVariable, "")
	if got := ErgaenzeSchulDomain("mueller"); got != "mueller" {
		t.Fatalf("ErgaenzeSchulDomain = %q, erwartet unverändert %q", got, "mueller")
	}
}

// Die Anmeldeseite bekommt die Domain als Wert, ohne Anmeldung; ohne Domain einen leeren Text.
func TestDomain_NenntDieDomainDerSchule(t *testing.T) {
	for _, tt := range []struct{ eingetragen, ldapURL, want string }{
		{"Schule.Example", "", `{"domain":"schule.example","ldap":false}`},
		{"", "", `{"domain":"","ldap":false}`},
		{"", "ldaps://dc.schule.test", `{"domain":"","ldap":true}`},
	} {
		t.Setenv(selbstanmeldeDomainVariable, tt.eingetragen)
		t.Setenv("LDAP_URL", tt.ldapURL)
		rec := httptest.NewRecorder()
		Domain()(rec, httptest.NewRequest(http.MethodGet, "/api/auth/domain", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Status %d, erwartet 200", rec.Code)
		}
		if got := rec.Body.String(); got != tt.want && got != tt.want+"\n" {
			t.Errorf("Antwort bei %q = %q, erwartet %s", tt.eingetragen, got, tt.want)
		}
	}
}
