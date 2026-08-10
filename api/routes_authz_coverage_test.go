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
		// Bestätigungs-Link an den Lieferanten (Migration 063). Kein Login, aber auch kein
		// offener Endpunkt: Der 256-Bit-Token aus der Bestellmail ist der Ausweis und
		// öffnet ausschließlich SEINE Bestellung. Sichtbar sind Lieferant, Datum,
		// Kundennummer und Titelzeilen — dieselben Angaben, die der Lieferant ohnehin als
		// Mailanhang hat; keine PII, keine Preise, kein Zugriff auf den Bestand.
		"/api/public/bestellung/{token}":                     "Bestätigungs-Link: Bestellansicht des Lieferanten (Token = Ausweis, nur diese Bestellung)",
		"/api/public/bestellung/{token}/etiketten/{groesse}": "Bestätigungs-Link: Etikettenbogen dieser Bestellung (identisch zum Mailanhang)",
		"/api/public/bestellung/{token}/bestaetigen":         "Bestätigungs-Link: einmalige Bestätigung durch den Lieferanten (atomar, danach 409)",
		"/uploads/":         "hochgeladene Cover-Bilder (öffentlich lesbar für Katalog/Monitor; Schülerfotos liegen NICHT hier, die stehen AES-verschlüsselt in der DB)",
		"/api/auth/refresh": "Auth-Endpunkt (validiert das Token selbst)",
		"/api/auth/me":      "Auth-Endpunkt (validiert das Token selbst)",
		"/api/auth/logout":  "Auth-Endpunkt",
		"/login":            "Login (Rate-Limit-Middleware, es existiert noch kein Token)",
		"/health":           "Health-Check",
		"/swagger/":         "API-Doku (nur bei APP_ENV=local/development registriert)",
		"/swagger":          "API-Doku (nur bei APP_ENV=local/development registriert)",
		"/favicon.ico":      "statisches Asset",
		"/":                 "SPA-Fallback (statisches Frontend)",
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
				// RequireAuthenticated verlangt eine gültige Sitzung, aber kein Fachrecht —
				// die richtige Schwelle für Endpunkte, die JEDER angemeldete Client öffnet
				// (heute nur der SSE-Stream /events). Bewusst hier und nicht auf der
				// Public-Allowlist: Ohne Sitzung kommt niemand durch.
				strings.Contains(zeile, "RequireAuthenticated(") ||
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

// TestInventurDelegationIstWirklichGeschuetzt prüft die Annahme nach, auf der
// Bedingung 2 des Gates oben beruht.
//
// Der Test oben wertet jede Zeile mit "invHandler" als geschützt — begründet damit,
// der Inventur-Mux setze intern RequireViewBooks/RequireEditBooks. Für eine seiner
// Routen stimmte das nicht: /uploads/ wird dort von einem nackten http.FileServer
// bedient. Das Gate meldete also "geschützt" für einen Endpunkt, den es nie geprüft
// hatte — ein grüner Test, dessen Grün nichts bedeutete.
//
// Statt der Zusicherung zu glauben, liest dieser Test die Registrierungen des
// Inventur-Mux und verlangt für jede OHNE Autorisierungs-Wrapper einen bewussten
// Eintrag auf der Public-Allowlist. Kommt dort eine ungeschützte Route hinzu, wird
// dieser Test rot — nicht erst der Betrieb.
func TestInventurDelegationIstWirklichGeschuetzt(t *testing.T) {
	inhalt, err := os.ReadFile(filepath.Join("..", "inventur", "api_routen.go"))
	if err != nil {
		t.Fatalf("inventur/api_routen.go lesen: %v", err)
	}

	registrierung := regexp.MustCompile(`handler\.mux\.Handle(?:Func)?\("([^"]+)"`)
	methodenPraefix := regexp.MustCompile(`^(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) `)

	geprueft := 0
	for _, zeile := range strings.Split(string(inhalt), "\n") {
		m := registrierung.FindStringSubmatch(zeile)
		if m == nil {
			continue
		}
		geprueft++
		if strings.Contains(zeile, "config.Require") || strings.Contains(zeile, "adminH") {
			continue
		}
		pfad := methodenPraefix.ReplaceAllString(m[1], "")
		if _, ok := publicAllowlistInventur[pfad]; ok {
			continue
		}
		t.Errorf("Inventur-Route %q trägt keinen Autorisierungs-Wrapper und steht nicht auf der "+
			"Public-Allowlist.\n→ Das Gate in TestAlleRoutenSindGeschuetzt hält jede Delegation an "+
			"invHandler pauschal für geschützt; diese Route wäre dort unbemerkt durchgerutscht.", m[1])
	}

	if geprueft < 5 {
		t.Fatalf("nur %d Inventur-Registrierungen erkannt — der Scanner greift vermutlich nicht mehr "+
			"(erwartet >10). Registrierungs-Syntax/Regex prüfen.", geprueft)
	}
}

// publicAllowlistInventur nennt die Routen des Inventur-Mux, die bewusst ohne
// Autorisierung ausgeliefert werden. JEDE Ergänzung ist eine Sicherheitsentscheidung.
var publicAllowlistInventur = map[string]string{
	"/uploads/": "hochgeladene Cover-Bilder, öffentlich lesbar für Katalog/Monitor. " +
		"Kein Verzeichnislisting (neuteredFileSystem); Schülerfotos liegen nicht hier, " +
		"sondern AES-verschlüsselt in der Datenbank.",
}
