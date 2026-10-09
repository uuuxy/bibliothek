package api

import (
	"bibliothek/pkg/leserart"
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Die Arten eines Lesers stehen an vier Stellen: in der Datenbank (chk_leser_art), im Server
// (pkg/leserart, dsgvoLeserart), im Browser (leserArt.js) und in den Prüffällen, die Server
// und Browser beide lesen (leserArt.faelle.json). Eine Art, die an einer Stelle fehlt, hieße
// im Browser „Schüler", käme beim Server als „Unbekannte Art" zurück oder bekäme ein Konto,
// das sie nicht haben soll. Wort, Grenze zum Kollegium und Zugang je Art hält der Test in
// pkg/leserart gegen die Prüffälle; hier stehen die Datenbank und die Auskunft.
func TestLeserArten_WieInDerDatenbankUndImBrowser(t *testing.T) {
	const faelleDatei = "../frontend/src/lib/leserArt.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Arten []struct {
			Art string `json:"art"`
		} `json:"arten"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatalf("Prüffälle: %v", err)
	}
	if len(pruefung.Arten) < 7 {
		t.Fatalf("%d Arten — erwartet mindestens 7: liest der Test noch auf leserArt.faelle.json?",
			len(pruefung.Arten))
	}

	var ausDatei []string
	for _, a := range pruefung.Arten {
		ausDatei = append(ausDatei, a.Art)
		if !leserart.Bekannt(a.Art) {
			t.Errorf("%s: steht in den Prüffällen, der Server kennt die Art nicht", a.Art)
		}
		if dsgvoLeserart(a.Art) == a.Art {
			t.Errorf("%s: Die Auskunft schreibt die Art nicht aus", a.Art)
		}
	}

	// Die Datenbank: die Werte aus der CHECK-Bedingung, so wie Postgres sie ausgibt.
	pool := pgTestPool(t)
	var def string
	if err := pool.QueryRow(context.Background(), `
		SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = 'chk_leser_art'`).Scan(&def); err != nil {
		t.Fatalf("chk_leser_art lesen: %v", err)
	}
	var inDerDatenbank []string
	for _, m := range regexp.MustCompile(`'([a-z]+)'`).FindAllStringSubmatch(def, -1) {
		inDerDatenbank = append(inDerDatenbank, m[1])
		if !leserart.Bekannt(m[1]) {
			t.Errorf("chk_leser_art erlaubt %q, der Server kennt die Art nicht", m[1])
		}
	}
	slices.Sort(inDerDatenbank)
	sortiert := slices.Sorted(slices.Values(ausDatei))
	if !slices.Equal(inDerDatenbank, sortiert) {
		t.Errorf("chk_leser_art erlaubt %v, die Prüffälle nennen %v", inDerDatenbank, sortiert)
	}

	// Liest die Browser-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../frontend/src/lib/leserArt.test.js")
	if err != nil {
		t.Fatalf("Vitest lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./leserArt.faelle.json") {
		t.Error("leserArt.test.js liest leserArt.faelle.json nicht mehr ein")
	}
}
