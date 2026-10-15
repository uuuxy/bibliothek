package api

import "testing"

// Der Bestätigungs-Token darf nie in ein Logfile geraten. Säße die Maskierung nur an der Zeile
// für 500, stünde der verschickte Link trotzdem im Klartext im Log: Die Zeile je Anfrage in
// router.go protokolliert jeden Aufruf mit vollem Pfad. Beide Pfadformen tragen das Geheimnis:
// die Seite selbst und ihre Aufrufe der Schnittstelle.
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
