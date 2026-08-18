package api

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// PII-Matrix-Gate (Abschluss von Befund F2, bewertung/datenbank-pruefbericht.md):
// docs/PII_MATRIX.de.md stuft JEDE Route nach Schülerdaten ein (Stufe 0–3) und
// nennt das Recht davor. Dieses Gate hält Dokument und Code deckungsgleich:
//
//  1. Jede registrierte Route braucht eine Zeile in der Matrix — eine NEUE Route
//     ohne Einstufung wird rot. Genau das war Daniels Kern von F2: "Adresse
//     vergessen zu entfernen" darf nicht lautlos möglich sein.
//  2. Jede Matrix-Zeile braucht ihre Route — das Dokument kann nicht veralten.
//  3. Das dokumentierte Recht muss dem RequirePermission(...) der Registrierung
//     entsprechen — die Matrix kann nicht mehr Schutz behaupten, als da ist.
//  4. Öffentliche Routen (ohne Sitzung erreichbar) müssen Stufe 0 sein.
//
// Die STUFE selbst (was die Antwort wirklich enthält) prüft kein Parser — sie ist
// beim Anlegen der Zeile von Hand am Handler zu verifizieren. Das Gate erzwingt,
// DASS diese Verifikation stattfindet, nicht ihr Ergebnis.

type matrixZeile struct {
	Route  string // "GET /api/vormerkungen"
	Recht  string // "view_students", "öffentlich", "RequireAuthenticated", "inventur:view_books", ...
	Stufe  string // "0".."3"
	Quelle string // Dateiname laut Abschnittsüberschrift im Dokument
}

// Auch Mount-Registrierungen ohne HTTP-Methode ("/api/books", "/uploads/")
// sind Routen und brauchen eine Zeile.
var matrixTabellenzeile = regexp.MustCompile(`^\|\s*` + "`" + `?((?:(?:GET|POST|PUT|PATCH|DELETE) )?/[^|` + "`" + `]*?)` + "`" + `?\s*\|\s*([^|]+?)\s*\|\s*([0-3])\s*\|`)

func leseMatrix(t *testing.T) map[string]matrixZeile {
	t.Helper()
	inhalt, err := os.ReadFile(filepath.Join("..", "docs", "PII_MATRIX.de.md"))
	if err != nil {
		t.Fatalf("docs/PII_MATRIX.de.md lesen: %v", err)
	}
	zeilen := map[string]matrixZeile{}
	quelle := ""
	for _, l := range strings.Split(string(inhalt), "\n") {
		if strings.HasPrefix(l, "## ") {
			quelle = strings.TrimSpace(strings.TrimPrefix(l, "## "))
		}
		m := matrixTabellenzeile.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		route := strings.TrimSpace(m[1])
		if _, doppelt := zeilen[route]; doppelt {
			t.Errorf("Matrix führt %q doppelt", route)
		}
		zeilen[route] = matrixZeile{Route: route, Recht: strings.TrimSpace(m[2]), Stufe: m[3], Quelle: quelle}
	}
	if len(zeilen) < 100 {
		t.Fatalf("nur %d Matrix-Zeilen erkannt — Tabellenformat/Regex prüfen (erwartet >150)", len(zeilen))
	}
	return zeilen
}

// registrierteRouten sammelt Route → Registrierungszeile aus den api-Routendateien
// und dem Inventur-Mux. Dieselbe lexikalische Technik wie das Authz-Coverage-Gate.
func registrierteRouten(t *testing.T) map[string]string {
	t.Helper()
	muster := regexp.MustCompile(`(?:mux|handler\.mux)\.Handle(?:Func)?\("([^"]+)"`)
	routen := map[string]string{}

	dateien, err := filepath.Glob("routes_*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	dateien = append(dateien, "router.go", filepath.Join("..", "inventur", "api_routen.go"))
	for _, datei := range dateien {
		if strings.HasSuffix(datei, "_test.go") {
			continue
		}
		inhalt, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("lesen %s: %v", datei, err)
		}
		for _, zeile := range strings.Split(string(inhalt), "\n") {
			m := muster.FindStringSubmatch(zeile)
			if m == nil {
				continue
			}
			routen[strings.TrimSpace(m[1])] = zeile
		}
	}
	if len(routen) < 100 {
		t.Fatalf("nur %d Routen erkannt — Scanner greift nicht mehr", len(routen))
	}
	return routen
}

func TestPIIMatrixDecktJedeRoute(t *testing.T) {
	matrix := leseMatrix(t)
	routen := registrierteRouten(t)

	for route := range routen {
		if _, ok := matrix[route]; !ok {
			t.Errorf("Route %q ist nicht in docs/PII_MATRIX.de.md eingestuft.\n"+
				"→ Handler lesen, Schülerdaten-Stufe (0–3) bestimmen und eine Tabellenzeile ergänzen.", route)
		}
	}
	for route, z := range matrix {
		if _, ok := routen[route]; !ok {
			t.Errorf("Matrix-Zeile %q (Abschnitt %s) hat keine registrierte Route mehr — Zeile entfernen oder Pfad korrigieren.", route, z.Quelle)
		}
	}
}

// TestPIIMatrixRechtStimmtMitCodeUeberein: Die Matrix darf keinen Schutz
// behaupten, den die Registrierung nicht trägt — und umgekehrt keinen laxeren
// dokumentieren, als da ist (sonst liest der Datenschutz-Prüfer falsche Fakten).
func TestPIIMatrixRechtStimmtMitCodeUeberein(t *testing.T) {
	matrix := leseMatrix(t)
	routen := registrierteRouten(t)

	for route, z := range matrix {
		zeile, ok := routen[route]
		if !ok {
			continue // meldet schon der Test oben
		}
		var erwartet string
		switch z.Recht {
		case "öffentlich", "Token", "selbst-prüfend":
			// Kein Authz-Wrapper: Token-Routen weisen sich über den Link-Token aus,
			// selbst-prüfende (auth/*) validieren das Session-Cookie im Handler.
			if strings.Contains(zeile, "RequirePermission(") || strings.Contains(zeile, "RequireRoles(") || strings.Contains(zeile, "RequireAuthenticated(") {
				t.Errorf("%s: Matrix sagt %q, Code hat einen Wrapper: %s", route, z.Recht, strings.TrimSpace(zeile))
			}
			continue
		case "Sitzung":
			erwartet = "RequireAuthenticated("
		case "inventur-Mux":
			erwartet = "invHandler" // Schutz liegt im inneren Mux, dessen Routen stehen einzeln in der Matrix
		case "inventur:view_books":
			erwartet = "RequireViewBooks("
		case "inventur:edit_books":
			// Die Schreibrouten teilen sich einen vorgebauten Handler:
			// adminH := config.RequireEditBooks(...) — die Variable IST der Wrapper.
			if !strings.Contains(zeile, "RequireEditBooks(") && !strings.Contains(zeile, "adminH") {
				t.Errorf("%s: Matrix dokumentiert Recht %q, die Registrierung sagt etwas anderes: %s", route, z.Recht, strings.TrimSpace(zeile))
			}
			continue
		default:
			erwartet = fmt.Sprintf("RequirePermission(%q)", z.Recht)
		}
		if !strings.Contains(zeile, erwartet) {
			t.Errorf("%s: Matrix dokumentiert Recht %q, die Registrierung sagt etwas anderes: %s", route, z.Recht, strings.TrimSpace(zeile))
		}
	}
}

// TestPIIMatrixOeffentlichIstStufe0: Was ohne Sitzung erreichbar ist, darf keine
// Schülerdaten führen. Diese Regel ist absolut — eine öffentliche Route mit
// Stufe ≥1 ist kein Dokumentationsfehler, sondern ein Datenschutzvorfall.
func TestPIIMatrixOeffentlichIstStufe0(t *testing.T) {
	for route, z := range leseMatrix(t) {
		ohneSitzung := z.Recht == "öffentlich" || z.Recht == "Token" || z.Recht == "selbst-prüfend"
		if ohneSitzung && z.Stufe != "0" {
			t.Errorf("%s ist ohne Fachrecht erreichbar (%s), aber als Stufe %s eingestuft — solche Routen müssen Stufe 0 sein", route, z.Recht, z.Stufe)
		}
	}
}
