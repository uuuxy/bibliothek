package api

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"bibliothek/repository"
)

// Die Arten eines Lesers stehen an vier Stellen: in der Datenbank (chk_leser_art), im Server
// (leserArtenListe, repository.ArtMitKonto, dsgvoLeserart), im Browser (leserArt.js) und in
// den Prüffällen, die Server und Browser beide lesen (leserArt.faelle.json). Seit Migration
// 153 sind es sieben; eine Art, die an einer Stelle fehlt, hieße im Browser „Schüler", käme
// beim Server als „Unbekannte Art" zurück oder bekäme ein Konto, das sie nicht haben soll.
func TestLeserArten_WieInDerDatenbankUndImBrowser(t *testing.T) {
	const faelleDatei = "../frontend/src/lib/leserArt.faelle.json"
	roh, err := os.ReadFile(faelleDatei)
	if err != nil {
		t.Fatalf("Prüffälle lesen: %v", err)
	}
	var pruefung struct {
		Arten []struct {
			Art       string `json:"art"`
			Text      string `json:"text"`
			Kollegium bool   `json:"kollegium"`
			MitKonto  bool   `json:"mitKonto"`
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
		if ist := leserArtBezeichnung(a.Art); ist != a.Text {
			t.Errorf("%s: leserArtBezeichnung = %q, erwartet %q", a.Art, ist, a.Text)
		}
		if ist := !istSchuelerArt(a.Art); ist != a.Kollegium {
			t.Errorf("%s: Kollegium = %v, erwartet %v", a.Art, ist, a.Kollegium)
		}
		if ist := repository.ArtMitKonto(a.Art); ist != a.MitKonto {
			t.Errorf("%s: repository.ArtMitKonto = %v, erwartet %v", a.Art, ist, a.MitKonto)
		}
		if dsgvoLeserart(a.Art) == a.Art {
			t.Errorf("%s: Die Auskunft schreibt die Art nicht aus", a.Art)
		}
	}

	var imServer []string
	for _, a := range leserArtenListe {
		imServer = append(imServer, a.art)
	}
	if !slices.Equal(imServer, ausDatei) {
		t.Errorf("Server %v, Prüffälle %v — Menge und Reihenfolge müssen gleich sein", imServer, ausDatei)
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
