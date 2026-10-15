package api

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Die Oberfläche der Rechte und die Routen müssen dieselbe Menge kennen.
//
// Es gibt zwei Listen von Berechtigungen, und beide steuern etwas Echtes:
//
//   - `frontend/src/lib/rechteKategorien.js` — die Schalter, die ein Admin umlegt,
//     samt Beschriftung; `menu.js` blendet Menüpunkte danach ein.
//   - die `verlangeRecht("…")` an den Routen — was der Server tatsächlich prüft.
//
// Läuft ein Name nur in einer der beiden Listen, ist der Schaden still und geht in
// beide Richtungen:
//
//   - Nur in der Oberfläche: Der Admin legt einen Schalter um, der nichts bewirkt. Verlangt die
//     Statistik-Route etwa `view_students` statt `view_stats`, entzieht nichts, wer die
//     Statistik entzieht, und wer sie erteilt, bekommt einen Menüpunkt, der 403 liefert.
//     Unbemerkt bleibt das, solange in der ausgelieferten Vorgabe beide Werte je Rolle
//     übereinstimmen.
//   - Nur im Server: Ein Recht, das kein Mensch je erteilen kann — die Funktion dahinter
//     ist für jede Rolle unerreichbar, die es nicht aus der Vorgabe mitbekommt.
//
// Der Test prüft die Namen, nicht die Zuordnung. Ob `view_stats` an der richtigen Route hängt,
// bleibt eine Urteilsfrage; dass es überhaupt an einer hängt, ist mechanisch.
func TestRechteInOberflaecheUndRoutenSindDeckungsgleich(t *testing.T) {
	inOberflaeche := rechteAusOberflaeche(t)
	anRouten := rechteAnRouten(t)

	for _, recht := range sortiert(inOberflaeche) {
		if !anRouten[recht] {
			t.Errorf("Recht %q steht in rechteKategorien.js, wird aber von KEINER Route verlangt — "+
				"ein Schalter ohne Wirkung. Entweder an die zuständige Route hängen oder aus der "+
				"Oberfläche nehmen.", recht)
		}
	}
	for _, recht := range sortiert(anRouten) {
		if !inOberflaeche[recht] {
			t.Errorf("Route verlangt %q, aber die Rechte-Oberfläche kennt es nicht — niemand kann es "+
				"erteilen. Zeile in frontend/src/lib/rechteKategorien.js ergänzen.", recht)
		}
	}
}

// TestMenuepunkteVerlangenBekannteRechte schließt die dritte Tür: Ein Menüpunkt, dessen
// `permission` es nirgends gibt, wird für jede Rolle ausgeblendet — die Seite ist gebaut
// und unerreichbar (dieselbe Klasse wie eine tote Route, nur von vorn).
func TestMenuepunkteVerlangenBekannteRechte(t *testing.T) {
	inOberflaeche := rechteAusOberflaeche(t)

	roh, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "menu.js"))
	if err != nil {
		t.Fatalf("menu.js lesen: %v", err)
	}
	muster := regexp.MustCompile(`permission:\s*'([a-z_]+)'`)
	treffer := muster.FindAllStringSubmatch(string(roh), -1)
	// Sammelpunkte tragen `permissions: [ 'a', 'b', … ]` (etwa „Einstellungen").
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
	roh, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "rechteKategorien.js"))
	if err != nil {
		t.Fatalf("rechteKategorien.js lesen: %v", err)
	}
	muster := regexp.MustCompile(`key:\s*'([a-z_]+)'`)
	menge := map[string]bool{}
	for _, m := range muster.FindAllStringSubmatch(string(roh), -1) {
		menge[m[1]] = true
	}
	if len(menge) == 0 {
		t.Fatal("keine Rechte in rechteKategorien.js gefunden — der Detektor greift ins Leere")
	}
	return menge
}

// rechteAnRouten liest die Aufrufe von verlangeRecht aus den Routendateien als Text, wie
// geschuetzte_routen_test.go nebenan.
func rechteAnRouten(t *testing.T) map[string]bool {
	t.Helper()
	dateien, err := filepath.Glob("routen_*.go")
	if err != nil {
		t.Fatalf("glob routen_*.go: %v", err)
	}
	dateien = append(dateien, "router.go")

	muster := regexp.MustCompile(`verlangeRecht\("([a-z_]+)"\)`)
	menge := map[string]bool{}
	for _, d := range dateien {
		// Eine Routenliste, die fehlt, ist ein Fehler: Übergangen, fehlten ihre Rechte still.
		roh, err := os.ReadFile(d)
		if err != nil {
			t.Fatalf("%s lesen: %v", d, err)
		}
		for _, m := range muster.FindAllStringSubmatch(string(roh), -1) {
			menge[m[1]] = true
		}
	}
	if len(menge) == 0 {
		t.Fatal("keine verlangeRecht-Aufrufe gefunden — der Detektor greift ins Leere")
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
// Denselben Begriff gibt es in drei Schreibweisen: das Enum der Datenbank klein (`'admin'`),
// die Konstanten in Go groß (`RolleAdmin Rolle = "ADMIN"`), und die Oberfläche vergleicht genau
// klein (`rolle === 'admin'`). Zusammengehalten wird das nur von `EqualFold` und `UPPER()` im
// Go-Code, in der Oberfläche von gar nichts.
//
// Wandert eine Liste allein, stirbt ein Zweig still: Benennt eine Migration eine Rolle um, etwa
// LEHRER in KOLLEGIUM, und ein Vergleich steht weiter auf dem alten Namen, ist sein Zweig tot.
// Eine Lehrkraft bekäme beim Scan eines freien Buchs „nicht ausgeliehen" statt ihrer Ausleihe in
// den Handapparat, und kein Test merkte es.
//
// Geprüft wird gegen schema.sql: Dort steht das Enum, und die Datenbank entscheidet am Ende.
func TestRollenVokabularIstUeberallDasselbe(t *testing.T) {
	ausSchema := rollenAusSchema(t)

	// 1. Go-Konstanten (auth/token.go) — Großschreibung ist erlaubt, ein anderer Name nicht.
	roh, err := os.ReadFile(filepath.Join("..", "auth", "token.go"))
	if err != nil {
		t.Fatalf("auth/token.go lesen: %v", err)
	}
	konstante := regexp.MustCompile(`Rolle[A-Za-z]+\s+Rolle\s*=\s*"([A-Za-z]+)"`)
	treffer := konstante.FindAllStringSubmatch(string(roh), -1)
	if len(treffer) == 0 {
		t.Fatal("keine Rollen-Konstanten in auth/token.go gefunden — der Detektor greift ins Leere")
	}
	for _, m := range treffer {
		if !ausSchema[strings.ToLower(m[1])] {
			t.Errorf("auth/token.go kennt die Rolle %q, das Enum benutzer_rolle in schema.sql nicht", m[1])
		}
	}

	// 2. Vergleiche in der Oberfläche: Hier gilt Groß- und Kleinschreibung, `===` ist unbarmherzig.
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
