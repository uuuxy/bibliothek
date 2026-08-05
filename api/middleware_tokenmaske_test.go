package api

import "testing"

// Der Bestätigungs-Token darf nie in ein Logfile geraten.
//
// Anlass ist ein Fund aus dem Smoke-Test gegen den laufenden Server: Die Maskierung saß
// zuerst nur an der 500er-Zeile, während die Request-Zeile in router.go JEDEN Aufruf mit
// vollem Pfad protokolliert — der verschickte Link stand fünfmal im Klartext im Log.
// Beide Pfadformen tragen das Geheimnis: die Seite selbst und ihre API-Aufrufe.
func TestMaskiereToken(t *testing.T) {
	faelle := map[string]string{
		"/api/public/bestellung/GEHEIM123":                                   "/api/public/bestellung/***",
		"/api/public/bestellung/GEHEIM123/bestaetigen":                       "/api/public/bestellung/***/bestaetigen",
		"/api/public/bestellung/GEHEIM123/etiketten/klein":                   "/api/public/bestellung/***/etiketten/klein",
		"/bestellung/GEHEIM123":                                              "/bestellung/***",
		"/api/bestellungen/11111111-1111-1111-1111-111111111111/bestaetigen": "/api/bestellungen/11111111-1111-1111-1111-111111111111/bestaetigen",
		"/api/public/opac/suche":                                             "/api/public/opac/suche",
		"/health":                                                            "/health",
	}

	for pfad, erwartet := range faelle {
		if got := maskiereToken(pfad); got != erwartet {
			t.Errorf("maskiereToken(%q) = %q, want %q", pfad, got, erwartet)
		}
	}
}
