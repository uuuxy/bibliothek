package leserart

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// Server und Browser lesen dieselben Prüffälle. Ohne Datenbank prüft dieser Test Wort, Grenze
// zum Kollegium, Zugang und Reihenfolge je Art; die Werte der Datenbank hält
// api/leser_art_pg_test.go dagegen.
func TestArten_WieInDenPrueffaellen(t *testing.T) {
	const faelleDatei = "../../frontend/src/lib/leserArt.faelle.json"
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
		t.Fatalf("%d Arten, erwartet mindestens 7: liest der Test noch auf leserArt.faelle.json?",
			len(pruefung.Arten))
	}

	var ausDatei []string
	for _, a := range pruefung.Arten {
		ausDatei = append(ausDatei, a.Art)
		if !Bekannt(a.Art) {
			t.Errorf("%s: gilt als unbekannte Art", a.Art)
		}
		if ist := Bezeichnung(a.Art); ist != a.Text {
			t.Errorf("%s: Bezeichnung = %q, erwartet %q", a.Art, ist, a.Text)
		}
		if ist := IstKollegium(a.Art); ist != a.Kollegium {
			t.Errorf("%s: IstKollegium = %v, erwartet %v", a.Art, ist, a.Kollegium)
		}
		if ist := IstSchueler(a.Art); ist == a.Kollegium {
			t.Errorf("%s: IstSchueler = %v, im Kollegium = %v", a.Art, ist, a.Kollegium)
		}
		if ist := MitKonto(a.Art); ist != a.MitKonto {
			t.Errorf("%s: MitKonto = %v, erwartet %v", a.Art, ist, a.MitKonto)
		}
		wort := a.Text
		if !a.Kollegium {
			wort = "5a"
		}
		if ist := KlasseOderArt("5a", a.Art); ist != wort {
			t.Errorf("%s: KlasseOderArt = %q, erwartet %q", a.Art, ist, wort)
		}
	}
	var imServer []string
	for _, a := range arten {
		imServer = append(imServer, a.art)
	}
	if !slices.Equal(imServer, ausDatei) {
		t.Errorf("Server %v, Prüffälle %v: Menge und Reihenfolge müssen gleich sein", imServer, ausDatei)
	}
	if ist, soll := Moegliche(), "Schüler, Lehrkraft, LiV, Praktikum, Sekretariat, U-plus, Fachbereich"; ist != soll {
		t.Errorf("Moegliche = %q, erwartet %q", ist, soll)
	}

	// Liest die Browser-Seite dieselbe Datei? Sonst prüfte jede Seite nur sich selbst.
	vitest, err := os.ReadFile("../../frontend/src/lib/leserArt.test.js")
	if err != nil {
		t.Fatalf("Vitest lesen: %v", err)
	}
	if !strings.Contains(string(vitest), "./leserArt.faelle.json") {
		t.Error("leserArt.test.js liest leserArt.faelle.json nicht mehr ein")
	}
}

// Eine leere Art und eine unbekannte: Die beiden Fragen nach dem Schüler antworten bei leerer
// Art verschieden, und das ist an den Aufrufern so gewollt.
func TestArten_LeerUndUnbekannt(t *testing.T) {
	if IstSchueler("") {
		t.Error("IstSchueler(\"\") = true: Eine Eingabe ohne Art gälte als Schüler")
	}
	if IstKollegium("") {
		t.Error("IstKollegium(\"\") = true: Eine Zeile aus der Sicht schueler gälte als Kollege")
	}
	if ist := KlasseOderArt("7b", ""); ist != "7b" {
		t.Errorf("KlasseOderArt ohne Art = %q, erwartet die Klasse", ist)
	}
	if Bekannt("") || Bekannt("lehrer") {
		t.Error("eine leere oder unbekannte Art gilt als bekannt")
	}
	if MitKonto("") || MitKonto("lehrer") {
		t.Error("eine leere oder unbekannte Art bekäme ein Konto")
	}
	if ist := Bezeichnung("lehrer"); ist != "lehrer" {
		t.Errorf("Bezeichnung einer unbekannten Art = %q, erwartet den gespeicherten Wert", ist)
	}
	if !IstKollegium("lehrer") {
		t.Error("eine unbekannte Art gilt nicht als Kollegium, obwohl sie gesetzt und keine Schüler-Art ist")
	}
}
