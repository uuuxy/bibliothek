package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAlleRoutenSindGeschuetzt ist ein Sicherheits-Gate gegen versehentlich
// ungeschützte Endpunkte. Jede in router.go / routes_*.go registrierte HTTP-Route
// MUSS eine der drei Bedingungen erfüllen:
//
//  1. einen Autorisierungs-Wrapper tragen (RequirePermission / RequireRoles),
//  2. an den inventur-invHandler delegieren (der intern RequireViewBooks/-EditBooks setzt), ODER
//  3. bewusst auf der öffentlichen Allowlist unten stehen.
//
// Eine neu hinzugefügte Route ohne Schutz lässt diesen Test rot werden — damit kann
// kein Endpunkt unbemerkt ohne Zugriffskontrolle live gehen. Der Test parst die
// Registrierungs-Quelltexte lexikalisch; das ist bewusst simpel und deterministisch
// (kein Laufzeit-Router-Aufbau nötig).
func TestAlleRoutenSindGeschuetzt(t *testing.T) {
	// Bewusst öffentliche bzw. selbst-authentifizierende Routen (Pfad ohne HTTP-Methode).
	// JEDE Ergänzung hier ist eine bewusste Sicherheitsentscheidung — Reviewer aufgepasst.
	publicAllowlist := map[string]string{
		"/api/public/opac/suche": "öffentlicher Katalog (nur Titel/Autor/Verfügbarkeit, keine PII)",
		"/api/monitor/slides":    "öffentlicher Bibliotheks-Monitor (nur Buchdaten)",
		"/api/images/cover":      "öffentlicher Cover-Proxy (SSRF-Host-Allowlist in image_caching.go)",
		"/api/csrf-token":        "CSRF-Bootstrap-Endpunkt",
		"/api/auth/refresh":      "Auth-Endpunkt (validiert das Token selbst)",
		"/api/auth/me":           "Auth-Endpunkt (validiert das Token selbst)",
		"/api/auth/logout":       "Auth-Endpunkt",
		"/login":                 "Login (Rate-Limit-Middleware, es existiert noch kein Token)",
		"/health":                "Health-Check",
		"/swagger/":              "API-Doku (nur bei APP_ENV=local/development registriert)",
		"/swagger":               "API-Doku (nur bei APP_ENV=local/development registriert)",
		"/favicon.ico":           "statisches Asset",
		"/":                      "SPA-Fallback (statisches Frontend)",
	}

	registrierung := regexp.MustCompile(`mux\.Handle(?:Func)?\("([^"]+)"`)
	methodenPraefix := regexp.MustCompile(`^(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) `)

	dateien, err := filepath.Glob("routes_*.go")
	if err != nil {
		t.Fatalf("glob routes_*.go: %v", err)
	}
	dateien = append(dateien, "router.go")

	geprueft := 0
	for _, datei := range dateien {
		inhalt, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("lesen %s: %v", datei, err)
		}
		for _, zeile := range strings.Split(string(inhalt), "\n") {
			m := registrierung.FindStringSubmatch(zeile)
			if m == nil {
				continue
			}
			geprueft++
			muster := m[1]
			pfad := methodenPraefix.ReplaceAllString(muster, "")

			geschuetzt := strings.Contains(zeile, "RequirePermission(") ||
				strings.Contains(zeile, "RequireRoles(") ||
				strings.Contains(zeile, "invHandler")
			if geschuetzt {
				continue
			}
			if _, ok := publicAllowlist[pfad]; ok {
				continue
			}
			t.Errorf("Route %q (in %s) hat KEINEN Autorisierungs-Wrapper und steht nicht auf der Public-Allowlist.\n"+
				"→ Entweder mit RequirePermission(...)/RequireRoles(...) schützen, oder — falls bewusst öffentlich — "+
				"mit Begründung in publicAllowlist (routes_authz_coverage_test.go) aufnehmen.", muster, datei)
		}
	}

	// Sanity-Floor: Findet der Scanner (durch geänderte Registrierungs-Syntax o. Ä.)
	// plötzlich fast nichts, ist der Gate faktisch abgeschaltet — dann lieber laut
	// scheitern als still grün. Aktuell sind es ~110 Routen.
	if geprueft < 50 {
		t.Fatalf("nur %d Routen erkannt — der Scanner greift vermutlich nicht mehr (erwartet >100). "+
			"Registrierungs-Syntax/Regex prüfen.", geprueft)
	}
}
