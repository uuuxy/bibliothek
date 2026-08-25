package api

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Rechte-Paritäts-Gate: Die Rechte-Oberfläche und die Routen müssen DIESELBE Menge
// kennen.
//
// Es gibt zwei Listen von Berechtigungen, und beide steuern etwas Echtes:
//
//   - `frontend/src/lib/permissionMetadata.js` — die Schalter, die ein Admin umlegt,
//     samt Beschriftung; `menu.js` blendet Menüpunkte danach ein.
//   - die `RequirePermission("…")` an den Routen — was der Server tatsächlich prüft.
//
// Läuft ein Name nur in einer der beiden Listen, ist der Schaden still und geht in
// beide Richtungen:
//
//   - Nur in der Oberfläche: Der Admin legt einen Schalter um, der nichts bewirkt. Genau
//     das war `view_stats` bis zum 23.08.2026 — die Statistik-Route verlangte
//     `view_students`. Wer die Statistik entzog, entzog nichts; wer sie erteilte, bekam
//     einen Menüpunkt, der 403 lieferte. Aufgefallen ist es nur, weil in der
//     ausgelieferten Vorgabe beide Werte je Rolle zufällig übereinstimmten.
//   - Nur im Server: Ein Recht, das kein Mensch je erteilen kann — die Funktion dahinter
//     ist für jede Rolle unerreichbar, die es nicht aus der Vorgabe mitbekommt.
//
// Das Gate prüft die Namen, nicht die Zuordnung. Ob `view_stats` an der RICHTIGEN Route
// hängt, bleibt eine Urteilsfrage; dass es überhaupt an einer hängt, ist mechanisch.
func TestRechteInOberflaecheUndRoutenSindDeckungsgleich(t *testing.T) {
	inOberflaeche := rechteAusOberflaeche(t)
	anRouten := rechteAnRouten(t)

	for _, recht := range sortiert(inOberflaeche) {
		if !anRouten[recht] {
			t.Errorf("Recht %q steht in permissionMetadata.js, wird aber von KEINER Route verlangt — "+
				"ein Schalter ohne Wirkung. Entweder an die zuständige Route hängen oder aus der "+
				"Oberfläche nehmen.", recht)
		}
	}
	for _, recht := range sortiert(anRouten) {
		if !inOberflaeche[recht] {
			t.Errorf("Route verlangt %q, aber die Rechte-Oberfläche kennt es nicht — niemand kann es "+
				"erteilen. Zeile in frontend/src/lib/permissionMetadata.js ergänzen.", recht)
		}
	}
}

// TestMenuepunkteVerlangenBekannteRechte schließt die dritte Tür: Ein Menüpunkt, dessen
// `permission` es nirgends gibt, wird für JEDE Rolle ausgeblendet — die Seite ist gebaut
// und unerreichbar (dieselbe Klasse wie eine tote Route, nur von vorn).
func TestMenuepunkteVerlangenBekannteRechte(t *testing.T) {
	inOberflaeche := rechteAusOberflaeche(t)

	roh, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "menu.js"))
	if err != nil {
		t.Fatalf("menu.js lesen: %v", err)
	}
	muster := regexp.MustCompile(`permission:\s*'([a-z_]+)'`)
	treffer := muster.FindAllStringSubmatch(string(roh), -1)
	// Sammelpunkte tragen `permissions: [ 'a', 'b', … ]` (seit 24.08.2026, „Einstellungen").
	sammel := regexp.MustCompile(`(?s)permissions:\s*\[(.*?)\]`)
	einzel := regexp.MustCompile(`'([a-z_]+)'`)
	for _, block := range sammel.FindAllStringSubmatch(string(roh), -1) {
		treffer = append(treffer, einzel.FindAllStringSubmatch(block[1], -1)...)
	}
	if len(treffer) == 0 {
		t.Fatal("kein einziger Menüpunkt mit permission gefunden — der Detektor greift ins Leere")
	}
	for _, m := range treffer {
		if !inOberflaeche[m[1]] {
			t.Errorf("Menüpunkt verlangt %q, das die Rechte-Oberfläche nicht kennt — der Punkt "+
				"bleibt für jede Rolle unsichtbar", m[1])
		}
	}
}

func rechteAusOberflaeche(t *testing.T) map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "permissionMetadata.js"))
	if err != nil {
		t.Fatalf("permissionMetadata.js lesen: %v", err)
	}
	muster := regexp.MustCompile(`key:\s*'([a-z_]+)'`)
	menge := map[string]bool{}
	for _, m := range muster.FindAllStringSubmatch(string(roh), -1) {
		menge[m[1]] = true
	}
	if len(menge) == 0 {
		t.Fatal("keine Rechte in permissionMetadata.js gefunden — der Detektor greift ins Leere")
	}
	return menge
}

// rechteAnRouten liest die RequirePermission-Aufrufe aus den Registrierungsdateien —
// lexikalisch, wie das Routen-Schutz-Gate nebenan (routes_authz_coverage_test.go).
func rechteAnRouten(t *testing.T) map[string]bool {
	t.Helper()
	dateien, err := filepath.Glob("routes_*.go")
	if err != nil {
		t.Fatalf("glob routes_*.go: %v", err)
	}
	dateien = append(dateien, "router.go", "mail_routes.go")

	muster := regexp.MustCompile(`RequirePermission\("([a-z_]+)"\)`)
	menge := map[string]bool{}
	for _, d := range dateien {
		roh, err := os.ReadFile(d)
		if err != nil {
			if strings.Contains(err.Error(), "no such file") {
				continue
			}
			t.Fatalf("%s lesen: %v", d, err)
		}
		for _, m := range muster.FindAllStringSubmatch(string(roh), -1) {
			menge[m[1]] = true
		}
	}
	if len(menge) == 0 {
		t.Fatal("keine RequirePermission-Aufrufe gefunden — der Detektor greift ins Leere")
	}
	return menge
}

func sortiert(m map[string]bool) []string {
	aus := make([]string, 0, len(m))
	for k := range m {
		aus = append(aus, k)
	}
	sort.Strings(aus)
	return aus
}

// TestRollenVokabularIstUeberallDasselbe hält die dritte Liste zusammen, die niemand
// hält: die Rollen.
//
// Denselben Begriff gibt es in drei Schreibweisen — DB-Enum klein (`'admin'`),
// Go-Konstanten groß (`RoleAdmin Role = "ADMIN"`), und das Frontend vergleicht exakt
// klein (`rolle === 'admin'`, rund zwei Dutzend Stellen). Zusammengehalten wird das nur
// von `EqualFold` und `UPPER()` im Go-Code — im Frontend von gar nichts.
//
// Was passiert, wenn eine Liste allein wandert, ist belegt: Migration 069 benannte
// LEHRER in KOLLEGIUM um, und der Vergleich in loan_return.go stand wochenlang auf dem
// wörtlichen "LEHRER". Der Zweig war tot; eine Lehrkraft bekam beim Scan eines freien
// Buchs "nicht ausgeliehen" statt ihrer Handapparat-Ausleihe. Gefunden hat das erst eine
// Vermessung, kein Test.
//
// Geprüft wird gegen schema.sql — dort steht das Enum, und die Datenbank ist die
// Instanz, die am Ende entscheidet.
func TestRollenVokabularIstUeberallDasselbe(t *testing.T) {
	ausSchema := rollenAusSchema(t)

	// 1. Go-Konstanten (auth/jwt.go) — Großschreibung ist erlaubt, ein anderer Name nicht.
	roh, err := os.ReadFile(filepath.Join("..", "auth", "jwt.go"))
	if err != nil {
		t.Fatalf("auth/jwt.go lesen: %v", err)
	}
	konstante := regexp.MustCompile(`Role[A-Za-z]+\s+Role\s*=\s*"([A-Za-z]+)"`)
	treffer := konstante.FindAllStringSubmatch(string(roh), -1)
	if len(treffer) == 0 {
		t.Fatal("keine Rollen-Konstanten in auth/jwt.go gefunden — der Detektor greift ins Leere")
	}
	for _, m := range treffer {
		if !ausSchema[strings.ToLower(m[1])] {
			t.Errorf("auth/jwt.go kennt die Rolle %q, das Enum benutzer_rolle in schema.sql nicht", m[1])
		}
	}

	// 2. Frontend-Vergleiche — hier gilt Groß-/Kleinschreibung, `===` ist unbarmherzig.
	vergleich := regexp.MustCompile(`(?:rolle|role)\s*(?:\?\.\w+\(\))?\s*===\s*'([a-z_]+)'`)
	gefunden := 0
	for _, pfad := range svelteUndJs(t) {
		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatalf("%s lesen: %v", pfad, err)
		}
		for _, m := range vergleich.FindAllStringSubmatch(string(inhalt), -1) {
			gefunden++
			if !ausSchema[m[1]] {
				t.Errorf("%s vergleicht die Rolle gegen %q — das Enum benutzer_rolle kennt "+
					"diesen Wert nicht, der Zweig ist tot", pfad, m[1])
			}
		}
	}
	if gefunden == 0 {
		t.Fatal("kein einziger Rollen-Vergleich im Frontend gefunden — der Detektor greift ins Leere")
	}
}

func rollenAusSchema(t *testing.T) map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(filepath.Join("..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql lesen: %v", err)
	}
	zeile := regexp.MustCompile(`CREATE TYPE benutzer_rolle AS ENUM \(([^)]*)\)`)
	m := zeile.FindStringSubmatch(string(roh))
	if m == nil {
		t.Fatal("Enum benutzer_rolle nicht in schema.sql gefunden — der Detektor greift ins Leere")
	}
	menge := map[string]bool{}
	for _, w := range regexp.MustCompile(`'([a-z_]+)'`).FindAllStringSubmatch(m[1], -1) {
		menge[w[1]] = true
	}
	return menge
}

func svelteUndJs(t *testing.T) []string {
	t.Helper()
	var aus []string
	wurzel := filepath.Join("..", "frontend", "src")
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(pfad, ".svelte") || (strings.HasSuffix(pfad, ".js") && !strings.HasSuffix(pfad, ".test.js")) {
			aus = append(aus, pfad)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("frontend/src durchlaufen: %v", err)
	}
	return aus
}
