package repository

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"testing"
)

// gemeinsameGruende liest die geteilten Prüffälle der vier Gründe, die auch der Vitest des
// Löschdialogs liest (frontend/src/lib/utils/aussonderungsgruende.test.js).
func gemeinsameGruende(t *testing.T) (vorgabe string, gruende []struct{ Wert, Text string }) {
	t.Helper()
	roh, err := os.ReadFile("../frontend/src/lib/utils/aussonderungsgruende.faelle.json")
	if err != nil {
		t.Fatal(err)
	}
	var pruefung struct {
		Vorgabe string `json:"vorgabe"`
		Gruende []struct {
			Wert string `json:"wert"`
			Text string `json:"text"`
		} `json:"gruende"`
	}
	if err := json.Unmarshal(roh, &pruefung); err != nil {
		t.Fatal(err)
	}
	if len(pruefung.Gruende) != 4 {
		t.Fatalf("len(Gruende) = %d, erwartet 4: liest der Test noch aussonderungsgruende.faelle.json?", len(pruefung.Gruende))
	}
	for _, g := range pruefung.Gruende {
		gruende = append(gruende, struct{ Wert, Text string }{g.Wert, g.Text})
	}
	return pruefung.Vorgabe, gruende
}

// Das Abgangsbuch druckt dieselben Wörter, die der Dialog zur Wahl stellt, und die Tür nimmt genau
// diese vier Gründe an; der leere Grund der Altdaten heißt „ohne Angabe".
func TestAbgangsgrundText(t *testing.T) {
	vorgabe, gruende := gemeinsameGruende(t)
	for _, g := range gruende {
		if got := abgangsgrundText(g.Wert); got != g.Text {
			t.Errorf("abgangsgrundText(%q) = %q, erwartet %q", g.Wert, got, g.Text)
		}
		if !IstAussonderungsgrund(g.Wert) {
			t.Errorf("IstAussonderungsgrund(%q) = false", g.Wert)
		}
	}
	if !IstAussonderungsgrund(vorgabe) {
		t.Errorf("die Vorgabe %q ist kein Grund", vorgabe)
	}
	for _, kein := range []string{"", "aussortiert", "VERSCHENKT", "Alter Bestand"} {
		if IstAussonderungsgrund(kein) {
			t.Errorf("IstAussonderungsgrund(%q) = true", kein)
		}
	}
	if got := abgangsgrundText(""); got != "ohne Angabe" {
		t.Errorf("abgangsgrundText(\"\") = %q, erwartet \"ohne Angabe\"", got)
	}
}

// Die Datenbank lässt an beiden Stellen genau die vier Gründe der geteilten Fälle zu: am Exemplar
// und an der Buchzeile eines gelöschten Exemplars.
func TestAussonderungsgrund_DatenbankKenntDieselbenVier(t *testing.T) {
	pool := pgTestPool(t)
	_, gruende := gemeinsameGruende(t)
	var erwartet []string
	for _, g := range gruende {
		erwartet = append(erwartet, g.Wert)
	}
	slices.Sort(erwartet)
	wert := regexp.MustCompile(`'([A-Z]+)'`)
	for _, name := range []string{"chk_aussonderung_grund", "chk_geloeschte_exemplare_grund"} {
		var definition string
		if err := pool.QueryRow(context.Background(),
			`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = $1`, name).Scan(&definition); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var gefunden []string
		for _, m := range wert.FindAllStringSubmatch(definition, -1) {
			gefunden = append(gefunden, m[1])
		}
		slices.Sort(gefunden)
		gefunden = slices.Compact(gefunden)
		if !slices.Equal(gefunden, erwartet) {
			t.Errorf("%s lässt %v zu, erwartet %v", name, gefunden, erwartet)
		}
	}
}
