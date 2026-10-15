package api

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// docs/PII_MATRIX.de.md stuft jede Route nach Schülerdaten ein (Stufe 0–3) und nennt das Recht
// davor. Diese Tests halten Dokument und Code deckungsgleich:
//
//  1. Jede angemeldete Route braucht eine Zeile in der Matrix; eine neue Route ohne
//     Einstufung wird rot. „Adresse vergessen zu entfernen" darf nicht lautlos möglich sein.
//  2. Jede Matrix-Zeile braucht ihre Route; das Dokument kann nicht veralten.
//  3. Das dokumentierte Recht muss dem verlangeRecht(...) der Anmeldung entsprechen; die
//     Matrix kann nicht mehr Schutz behaupten, als da ist.
//  4. Öffentliche Routen (ohne Sitzung erreichbar) müssen Stufe 0 sein.
//
// Die Stufe selbst (was die Antwort wirklich enthält) prüft für GET-Routen
// pii_antwort_pg_test.go mit Kanarienwerten am echten Router. Für Routen ohne GET bleibt sie
// beim Anlegen der Zeile von Hand an der Tür nachzuprüfen: Diese Tests erzwingen, dass die
// Einordnung stattfindet, pii_antwort_pg_test.go misst ihr Ergebnis auf dem Lesepfad.

type matrixZeile struct {
	Route  string // "GET /api/vormerkungen"
	Recht  string // "view_students", "öffentlich", "Sitzung", "inventur:view_books", ...
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

// registrierteRouten sammelt Route → Zeile der Anmeldung aus den Routendateien von api/ und
// dem Mux der Inventur. Dieselbe Technik wie geschuetzte_routen_test.go, aber je Zeile; deshalb
// steht jede Anmeldung in einer Zeile.
func registrierteRouten(t *testing.T) map[string]string {
	t.Helper()
	muster := regexp.MustCompile(`(?:mux|handler\.mux)\.Handle(?:Func)?\("([^"]+)"`)
	routen := map[string]string{}

	dateien, err := filepath.Glob("routen_*.go")
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
			// Keine Prüfung des Zugriffs davor: Token-Routen weisen sich über den Token im
			// Link aus, selbst prüfende (auth/*) prüfen das Sitzungscookie in der Tür.
			if strings.Contains(zeile, "verlangeRecht(") ||
				strings.Contains(zeile, "VerlangeAnmeldung(") {
				t.Errorf("%s: Matrix sagt %q, Code hat einen Wrapper: %s", route, z.Recht, strings.TrimSpace(zeile))
			}
			continue
		case "Sitzung":
			// api/ ruft die Methode VerlangeAnmeldung, das Paket inventur das hereingereichte Feld
			// VerlangeAnmeldung seiner Konfiguration.
			if !strings.Contains(zeile, "VerlangeAnmeldung(") {
				t.Errorf("%s: Matrix dokumentiert Recht %q, die Registrierung sagt etwas anderes: %s", route, z.Recht, strings.TrimSpace(zeile))
			}
			continue
		case "inventur-Mux":
			erwartet = "inventurTueren" // Schutz liegt im inneren Mux, dessen Routen stehen einzeln in der Matrix
		case "inventur:view_books":
			erwartet = "VerlangeLesen("
		case "inventur:delete_books":
			// Bestand vernichten ist ein eigenes Recht, getrennt von edit_books: Hinge der
			// Massenlöschweg an edit_books, wäre er offener als das Löschen eines einzelnen
			// Titels, das delete_books verlangt.
			erwartet = "VerlangeLoeschen("
		case "inventur:edit_books":
			// Die Schreibrouten teilen sich eine vorgebaute Tür:
			// verwaltung := konfig.VerlangeAendern(...); die Variable ist die Prüfung des Zugriffs.
			if !strings.Contains(zeile, "VerlangeAendern(") && !strings.Contains(zeile, "verwaltung") {
				t.Errorf("%s: Matrix dokumentiert Recht %q, die Registrierung sagt etwas anderes: %s", route, z.Recht, strings.TrimSpace(zeile))
			}
			continue
		default:
			erwartet = fmt.Sprintf("verlangeRecht(%q)", z.Recht)
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
