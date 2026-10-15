package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAlleRoutenSindGeschuetzt hält versehentlich ungeschützte Endpunkte fern. Jede in
// router.go oder routen_*.go angemeldete Route erfüllt eine der drei Bedingungen:
//
//  1. Sie trägt eine Prüfung des Zugriffs (verlangeRecht oder VerlangeAnmeldung).
//  2. Sie reicht an inventurTueren weiter (dessen Routen prüft
//     TestPIIMatrixRechtStimmtMitCodeUeberein).
//  3. Sie steht bewusst auf der Liste der öffentlichen Routen unten.
//
// Eine neue Route ohne Schutz macht diesen Test rot; so geht kein Endpunkt unbemerkt ohne
// Zugriffskontrolle in Betrieb. Der Test liest die Quelltexte der Anmeldungen als Text; das ist
// bewusst einfach und braucht keinen Aufbau des Routers zur Laufzeit.
func TestAlleRoutenSindGeschuetzt(t *testing.T) {
	// Bewusst öffentliche bzw. selbst-authentifizierende Routen (Pfad ohne HTTP-Methode).
	// Jede Ergänzung hier ist eine bewusste Entscheidung über die Sicherheit.
	oeffentlicheRouten := map[string]string{
		"/api/public/opac/suche": "öffentlicher Katalog (nur Titel/Autor/Verfügbarkeit, keine PII)",
		"/api/monitor/slides":    "öffentlicher Bibliotheks-Monitor (nur Buchdaten)",
		"/api/images/cover":      "öffentlicher Cover-Proxy (SSRF-Host-Allowlist in api/katalog/cover.go)",
		"/api/csrf-token":        "CSRF-Bootstrap-Endpunkt",
		// Bestätigungs-Link an den Lieferanten. Keine Anmeldung, aber auch kein offener
		// Endpunkt: Der 256-Bit-Token aus der Bestellmail ist der Ausweis und öffnet
		// ausschließlich seine Bestellung. Sichtbar sind Lieferant, Datum, Kundennummer und
		// Titelzeilen, dieselben Angaben, die der Lieferant ohnehin als Mailanhang hat; keine
		// PII, keine Preise, kein Zugriff auf den Bestand.
		"/api/public/bestellung/{token}":                     "Bestätigungs-Link: Bestellansicht des Lieferanten (Token = Ausweis, nur diese Bestellung)",
		"/api/public/bestellung/{token}/etiketten/{groesse}": "Bestätigungs-Link: Etikettenbogen dieser Bestellung (identisch zum Mailanhang)",
		"/api/public/bestellung/{token}/bestaetigen":         "Bestätigungs-Link: einmalige Bestätigung durch den Lieferanten (atomar, danach 409)",
		"/uploads/":         "hochgeladene Cover-Bilder (öffentlich lesbar für Katalog/Monitor; Schülerfotos liegen NICHT hier, die stehen AES-verschlüsselt in der DB)",
		"/api/auth/refresh": "Auth-Endpunkt (validiert das Token selbst)",
		"/api/auth/me":      "Auth-Endpunkt (validiert das Token selbst)",
		"/api/auth/logout":  "Auth-Endpunkt",
		// Sperre nach Inaktivität: Beide prüfen das Token selbst und wirken nur auf die
		// Anmeldung, mit der die Anfrage kommt. Hinter VerlangeAnmeldung ginge Entsperren
		// nicht — die Tür lehnt eine gesperrte Anmeldung ab.
		"/api/auth/sperren":    "Auth-Endpunkt (validiert das Token selbst, sperrt nur die eigene Anmeldung)",
		"/api/auth/entsperren": "Auth-Endpunkt (validiert das Token selbst, prüft das Passwort; Rate-Limit-Middleware)",
		"/login":               "Login (Rate-Limit-Middleware, es existiert noch kein Token)",
		"/health":              "Health-Check",
		"/swagger/":            "API-Doku (nur bei APP_ENV=local/development registriert)",
		"/swagger":             "API-Doku (nur bei APP_ENV=local/development registriert)",
		"/favicon.ico":         "statisches Asset",
		"/":                    "SPA-Fallback (statisches Frontend)",
	}

	registrierung := regexp.MustCompile(`mux\.Handle(?:Func)?\("([^"]+)"`)
	methodenPraefix := regexp.MustCompile(`^(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) `)

	dateien, err := filepath.Glob("routen_*.go")
	if err != nil {
		t.Fatalf("glob routen_*.go: %v", err)
	}
	dateien = append(dateien, "router.go")

	geprueft := 0
	for _, datei := range dateien {
		inhalt, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("lesen %s: %v", datei, err)
		}
		for _, ausdruck := range registrierungsAusdruecke(string(inhalt), "mux.") {
			m := registrierung.FindStringSubmatch(ausdruck)
			if m == nil {
				continue
			}
			geprueft++
			muster := m[1]
			pfad := methodenPraefix.ReplaceAllString(muster, "")

			geschuetzt := strings.Contains(ausdruck, "verlangeRecht(") ||
				// VerlangeAnmeldung verlangt eine gültige Sitzung, aber kein Fachrecht: die
				// richtige Schwelle für Endpunkte, die jede angemeldete Rolle öffnet
				// (nur_sitzung_liste_test.go nennt sie). Bewusst hier und nicht auf der Liste der
				// öffentlichen Routen: Ohne Sitzung kommt niemand durch.
				strings.Contains(ausdruck, "VerlangeAnmeldung(") ||
				strings.Contains(ausdruck, "inventurTueren")
			if geschuetzt {
				continue
			}
			if _, ok := oeffentlicheRouten[pfad]; ok {
				continue
			}
			t.Errorf("Route %q (in %s) hat KEINEN Autorisierungs-Wrapper und steht nicht auf der Public-Allowlist.\n"+
				"→ Entweder mit verlangeRecht(...) schützen (VerlangeAnmeldung(...), wenn jede Sitzung genügt), oder — falls bewusst öffentlich — "+
				"mit Begründung in oeffentlicheRouten (geschuetzte_routen_test.go) aufnehmen.", muster, datei)
		}
	}

	// Untergrenze: Findet der Scanner (etwa nach einer geänderten Form der Anmeldung)
	// plötzlich fast nichts, ist der Test faktisch abgeschaltet; dann lieber laut scheitern als
	// still grün.
	if geprueft < 50 {
		t.Fatalf("nur %d Routen erkannt — der Scanner greift vermutlich nicht mehr (erwartet >100). "+
			"Registrierungs-Syntax/Regex prüfen.", geprueft)
	}
}

// TestInventurDelegationIstWirklichGeschuetzt prüft die Annahme nach, auf der Bedingung 2 des
// Tests oben beruht.
//
// Der Test oben wertet jede Anmeldung mit inventurTueren als geschützt, weil der Mux der
// Inventur intern VerlangeLesen oder VerlangeAendern setze. Für /uploads/ stimmt das nicht:
// Die Route bedient dort ein nackter http.FileServer. Der Test oben meldete also „geschützt" für
// einen Endpunkt, den er nie geprüft hat.
//
// Statt der Zusicherung zu glauben, liest dieser Test die Anmeldungen im Mux der Inventur und
// verlangt für jede ohne Prüfung des Zugriffs einen bewussten Eintrag in
// oeffentlicheRoutenInventur. Kommt dort eine ungeschützte Route hinzu, wird dieser Test rot,
// nicht erst der Betrieb.
func TestInventurDelegationIstWirklichGeschuetzt(t *testing.T) {
	inhalt, err := os.ReadFile(filepath.Join("..", "inventur", "api_routen.go"))
	if err != nil {
		t.Fatalf("inventur/api_routen.go lesen: %v", err)
	}

	registrierung := regexp.MustCompile(`tueren\.mux\.Handle(?:Func)?\("([^"]+)"`)
	methodenPraefix := regexp.MustCompile(`^(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) `)

	geprueft := 0
	for _, ausdruck := range registrierungsAusdruecke(string(inhalt), "tueren.mux.") {
		m := registrierung.FindStringSubmatch(ausdruck)
		if m == nil {
			continue
		}
		geprueft++
		if strings.Contains(ausdruck, "konfig.Verlange") || strings.Contains(ausdruck, "verwaltung") {
			continue
		}
		pfad := methodenPraefix.ReplaceAllString(m[1], "")
		if _, ok := oeffentlicheRoutenInventur[pfad]; ok {
			continue
		}
		t.Errorf("Inventur-Route %q trägt keinen Autorisierungs-Wrapper und steht nicht auf der "+
			"Public-Allowlist.\n→ Das Gate in TestAlleRoutenSindGeschuetzt hält jede Delegation an "+
			"inventurTueren pauschal für geschützt; diese Route wäre dort unbemerkt durchgerutscht.", m[1])
	}

	if geprueft < 5 {
		t.Fatalf("nur %d Inventur-Registrierungen erkannt — der Scanner greift vermutlich nicht mehr "+
			"(erwartet >10). Registrierungs-Syntax/Regex prüfen.", geprueft)
	}
}

// oeffentlicheRoutenInventur nennt die Routen des Inventur-Mux, die bewusst ohne
// Autorisierung ausgeliefert werden. Jede Ergänzung ist eine Entscheidung über die Sicherheit.
var oeffentlicheRoutenInventur = map[string]string{
	"/uploads/": "hochgeladene Cover-Bilder, öffentlich lesbar für Katalog/Monitor. " +
		"Kein Verzeichnislisting und kein Unterordner (neuteredFileSystem); Schülerfotos " +
		"liegen nicht hier, sondern AES-verschlüsselt in der Datenbank.",
}

// registrierungsAusdruecke liefert jeden Aufruf `<praefix>Handle(` oder `<praefix>HandleFunc(`
// als einen Ausdruck, von der öffnenden bis zur zugehörigen schließenden Klammer, über
// Zeilengrenzen hinweg; Klammern in Zeichenketten zählen nicht.
//
// Läsen die Tests nur die Zeile der Anmeldung, gälte eine Anmeldung, die zwischen Adresse und
// Prüfung des Zugriffs umbricht, als ungeschützt. Ein Umbruch kann nie ein falsches Grün
// erzeugen, nur ein falsches Rot, aber eines, das in die falsche Richtung schickt („schütze die
// Route", sie ist geschützt).
func registrierungsAusdruecke(inhalt, praefix string) []string {
	var ausdruecke []string
	for _, marker := range []string{praefix + "Handle(", praefix + "HandleFunc("} {
		rest := inhalt
		for {
			i := strings.Index(rest, marker)
			if i < 0 {
				break
			}
			ende := klammerEnde(rest, i+len(marker)-1)
			if ende < 0 {
				ausdruecke = append(ausdruecke, rest[i:])
				break
			}
			ausdruecke = append(ausdruecke, rest[i:ende+1])
			rest = rest[ende+1:]
		}
	}
	return ausdruecke
}

// klammerEnde liefert den Index der Klammer, die die öffnende an Position auf schließt;
// -1, wenn sie nicht schließt. Zeichenketten ("…" mit Escapes, `…`) werden übersprungen.
func klammerEnde(text string, auf int) int {
	tiefe := 0
	var inString byte
	for j := auf; j < len(text); j++ {
		c := text[j]
		switch {
		case inString != 0:
			if c == '\\' && inString == '"' {
				j++
				continue
			}
			if c == inString {
				inString = 0
			}
		case c == '"' || c == '`':
			inString = c
		case c == '(':
			tiefe++
		case c == ')':
			tiefe--
			if tiefe == 0 {
				return j
			}
		}
	}
	return -1
}

// TestRegistrierungsAusdruecke_LesenUeberZeilengrenzen ist die Probe für die beiden Tests
// oben: Bricht eine Anmeldung nach der Adresse um, steht die Prüfung des Zugriffs trotzdem im
// selben Ausdruck.
func TestRegistrierungsAusdruecke_LesenUeberZeilengrenzen(t *testing.T) {
	quelle := "\tmux.Handle(\"GET /api/x\",\n" +
		"\t\ts.verlangeRecht(\"view_x\")(h))\n" +
		"\tmux.HandleFunc(\"GET /y (frei)\", frei) // \")\" in der Adresse zählt nicht\n"
	aus := registrierungsAusdruecke(quelle, "mux.")
	if len(aus) != 2 {
		t.Fatalf("%d Ausdrücke statt 2: %q", len(aus), aus)
	}
	gefunden := map[string]bool{}
	for _, a := range aus {
		gefunden[a] = true
	}
	umgebrochen := "mux.Handle(\"GET /api/x\",\n\t\ts.verlangeRecht(\"view_x\")(h))"
	if !gefunden[umgebrochen] {
		t.Errorf("der umgebrochene Ausdruck fehlt oder endet an der Zeile: %q", aus)
	}
	if !gefunden["mux.HandleFunc(\"GET /y (frei)\", frei)"] {
		t.Errorf("Klammer in der Adresse verschiebt das Ende: %q", aus)
	}
}
