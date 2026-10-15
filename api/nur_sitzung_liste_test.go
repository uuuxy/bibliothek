package api

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Welche Routen kommen mit einer bloßen Sitzung aus, ohne Fachrecht?
//
// Das ist die kleine, aber empfindliche Fläche, die jede angemeldete Rolle öffnen kann, auch
// Kollegium (Portal) und Helfer (Kiosk). Ein Satz im Kommentar über VerlangeAnmeldung, der die
// Routen zählt, liegt leicht falsch: Die drei Routen des Portals im Paket inventur bekommen die
// Middleware hereingereicht (konfig.VerlangeAnmeldung), statt sie als Methode zu rufen, und ein
// Zähler, der nur nach der Methode von api/ sucht, übersieht sie.
//
// Deshalb zählt eine Ratsche statt eines Kommentars. Jede Zeile hier ist eine bewusste
// Entscheidung: Der Inhalt muss PII-Stufe 0 sein (keine Schülerdaten), sonst gehört die Route
// hinter ein Recht.
func TestNurSitzungRoutenSindBenannt(t *testing.T) {
	nurSitzung := map[string]string{
		"GET /events":                       "SSE-Stream; der authStore baut ihn direkt nach dem Login auf. Ein Fachrecht davor erzeugt statt einer Absage eine Reconnect-Schleife plus Offline-Overlay.",
		"GET /api/einstellungen/sitzung":    "Anzeige-Einstellungen der Sitzung (Schulname, Logo, Fristen) — jeder angemeldete Client braucht sie zum Zeichnen.",
		"GET /api/lmf-termine":              "veröffentlichter LMF-Plan: Datum, Stunde, Klassen, Vermerk — der Plan hängt für das ganze Kollegium aus.",
		"GET /api/lmf-termine/pdf":          "derselbe Plan als PDF (der Entwurf hat eine eigene Route hinter edit_books).",
		"GET /api/portal/klassensaetze":     "Klassensätze im Portal: Buch- und Zähldaten. view_books würde der Rolle den ganzen Medienkatalog öffnen.",
		"GET /api/portal/lernmittel":        "Schulbücher je Fach für die Fachsprecher: Zahlen und Titel.",
		"GET /api/portal/lernmittel/export": "derselbe Bestand als PDF-Export.",
	}

	// Zwei Schreibweisen, ein Sinn: `s.VerlangeAnmeldung()(…)` in api/, und die
	// hereingereichte Fassung `konfig.VerlangeAnmeldung(…)` im Paket inventur.
	registrierung := regexp.MustCompile(`(?:tueren\.)?mux\.Handle(?:Func)?\("([^"]+)"[^)]*VerlangeAnmeldung`)

	dateien, err := filepath.Glob("routen_*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	dateien = append(dateien, "router.go", filepath.Join("..", "inventur", "api_routen.go"))

	gefunden := map[string]bool{}
	for _, datei := range dateien {
		inhalt, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("%s lesen: %v", datei, err)
		}
		for _, zeile := range strings.Split(string(inhalt), "\n") {
			if treffer := registrierung.FindStringSubmatch(zeile); treffer != nil {
				gefunden[treffer[1]] = true
			}
		}
	}

	if len(gefunden) == 0 {
		t.Fatal("keine einzige Route mit VerlangeAnmeldung gefunden — die Suchform passt nicht mehr, " +
			"und dieses Gate wäre still grün")
	}
	var neu, weg []string
	for route := range gefunden {
		if _, benannt := nurSitzung[route]; !benannt {
			neu = append(neu, route)
		}
	}
	for route := range nurSitzung {
		if !gefunden[route] {
			weg = append(weg, route)
		}
	}
	sort.Strings(neu)
	sort.Strings(weg)
	if len(neu) > 0 {
		t.Errorf("Neue Route ohne Fachrecht, nur mit Sitzung: %v\n"+
			"Jede angemeldete Rolle kann sie öffnen — auch KOLLEGIUM und HELFER. Wenn das gewollt ist: "+
			"hier mit Begründung eintragen (Inhalt muss PII-Stufe 0 sein) und die PII-Matrix nachziehen.", neu)
	}
	if len(weg) > 0 {
		t.Errorf("Diese Routen stehen in der Liste, sind aber nicht (mehr) so registriert: %v\n"+
			"Eintrag entfernen — sonst behauptet die Liste eine Fläche, die es nicht gibt.", weg)
	}
}
