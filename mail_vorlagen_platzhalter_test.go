package main

// Die Platzhalter einer Mail-Vorlage stimmen an drei Orten überein.
//
// Drei Orte nennen je Typ der Vorlage, welche {{.X}}-Platzhalter gelten:
//
//   1. der Renderer in Go, die einzige Menge, die wirklich ersetzt wird
//      (pdf/mahnbrief.go für MAHNUNG_ELTERN, internal/service/bestellmail_text.go für
//      BESTELLUNG_HAENDLER),
//   2. die Anzeige im Editor der Vorlagen (vorlagenInfo in mailVorlagenInfo.js),
//   3. die Vorgabetexte (schema.sql).
//
// Warum: Zeigt der Editor Platzhalter, die der Renderer nicht ersetzt, schickt eine nach
// Anleitung umformulierte Vorlage dem Händler wörtlich „{{.BuchListe}}". Eine Vorlage ohne
// Renderer lässt sich bearbeiten und speichern, ohne je verschickt zu werden. Diese Prüfung hält
// die drei Orte deckungsgleich.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var platzhalterMuster = regexp.MustCompile(`\{\{\.[A-Za-z]+\}\}`)

// rendererQuellen: je Typ die Go-Datei, deren Zeichenketten die wirksame Menge der Platzhalter
// tragen (Replacer, Split, Ersatzvorlage).
var rendererQuellen = map[string]string{
	"MAHNUNG_ELTERN":      filepath.Join("pdf", "mahnbrief.go"),
	"BESTELLUNG_HAENDLER": filepath.Join("internal", "service", "bestellmail_text.go"),
}

// standardtextQuellen: je Typ die Go-Datei mit dem Text, der gilt, wenn die gespeicherte
// Vorlage fehlt oder leer ist, sofern er nicht beim Renderer steht.
var standardtextQuellen = map[string]string{
	"MAHNUNG_ELTERN": filepath.Join("api", "mahnwesen", "mahnbriefe.go"),
}

// ohneRenderer: Typen aus den Vorgabetexten, die absichtlich keinen Renderer haben, je mit
// Begründung. Ein Eintrag hier ist eine geparkte Lücke, kein Normalzustand.
var ohneRenderer = map[string]string{}

// sammlePlatzhalter zieht alle {{.X}}-Vorkommen aus einem Text als Menge.
func sammlePlatzhalter(text string) map[string]bool {
	out := map[string]bool{}
	for _, m := range platzhalterMuster.FindAllString(text, -1) {
		out[m] = true
	}
	return out
}

func sortierteMenge(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// leseRendererPlatzhalter liest die wirksame Menge aus den Zeichenketten der Datei des Renderers.
// Kommentare zählen nicht (dort wird über Platzhalter geredet, nicht ersetzt); dieselbe Technik
// mit ohneKommentare wie in der Prüfung der Schichtung.
func leseRendererPlatzhalter(t *testing.T, datei string) map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(datei)
	if err != nil {
		t.Fatalf("%s lesen: %v", datei, err)
	}
	return sammlePlatzhalter(ohneKommentare(string(roh)))
}

// leseEditorPlatzhalter parst vorlagenInfo aus frontend/src/lib/components/einstellungen/mailVorlagenInfo.js:
// je Typ-Block die platzhalter-Liste (und dass es den Block überhaupt gibt).
func leseEditorPlatzhalter(t *testing.T) map[string]map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(filepath.Join("frontend", "src", "lib", "components", "einstellungen", "mailVorlagenInfo.js"))
	if err != nil {
		t.Fatalf("mailVorlagenInfo.js lesen: %v", err)
	}
	blockMuster := regexp.MustCompile(`(?s)([A-Z_]+):\s*\{\s*name:.*?platzhalter:\s*\[(.*?)\]`)
	out := map[string]map[string]bool{}
	for _, m := range blockMuster.FindAllStringSubmatch(string(roh), -1) {
		out[m[1]] = sammlePlatzhalter(m[2])
	}
	if len(out) < 2 {
		t.Fatalf("nur %d Typ-Blöcke in vorlagenInfo erkannt — Parser oder Komponente prüfen", len(out))
	}
	return out
}

// leseVorgabetexte liefert je Typ aus den Vorgabetexten die Platzhalter seines Texts.
func leseVorgabetexte(t *testing.T) map[string]map[string]bool {
	t.Helper()
	roh, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("schema.sql lesen: %v", err)
	}
	inhalt := string(roh)
	start := strings.Index(inhalt, "INSERT INTO mail_vorlagen")
	ende := strings.Index(inhalt[start:], "ON CONFLICT (typ)")
	if start < 0 || ende < 0 {
		t.Fatal("die Vorgabetexte von mail_vorlagen in schema.sql nicht gefunden")
	}
	vorgabe := inhalt[start : start+ende]

	typMuster := regexp.MustCompile(`'([A-Z_]+)',`)
	treffer := typMuster.FindAllStringSubmatchIndex(vorgabe, -1)
	out := map[string]map[string]bool{}
	for i, m := range treffer {
		typ := vorgabe[m[2]:m[3]]
		blockEnde := len(vorgabe)
		if i+1 < len(treffer) {
			blockEnde = treffer[i+1][0]
		}
		out[typ] = sammlePlatzhalter(vorgabe[m[0]:blockEnde])
	}
	if len(out) < 2 {
		t.Fatalf("nur %d Vorgabetexte erkannt — Parser prüfen", len(out))
	}
	return out
}

func TestVorlagenPlatzhalterSindDeckungsgleich(t *testing.T) {
	editor := leseEditorPlatzhalter(t)
	vorgaben := leseVorgabetexte(t)

	// 1) Je Typ mit Renderer: Anzeige im Editor ≡ wirksame Menge des Renderers, und der
	//    Vorgabetext benutzt nichts, was der Renderer nicht ersetzt.
	for typ, datei := range rendererQuellen {
		renderer := leseRendererPlatzhalter(t, datei)
		if len(renderer) == 0 {
			t.Fatalf("%s: keine Platzhalter in %s gefunden — Detektor greift nicht mehr", typ, datei)
		}
		ui, ok := editor[typ]
		if !ok {
			t.Errorf("%s fehlt in vorlagenInfo (mailVorlagenInfo.js) — der Editor zeigt dann nichts an.", typ)
			continue
		}
		for p := range renderer {
			if !ui[p] {
				t.Errorf("%s: Renderer ersetzt %s, der Editor verschweigt es — unentdeckbares Feature.", typ, p)
			}
		}
		for p := range ui {
			if !renderer[p] {
				t.Errorf("%s: Editor bietet %s an, der Renderer (%s) ersetzt es nicht — der Platzhalter stünde wörtlich im Versand.", typ, p, datei)
			}
		}
		for p := range vorgaben[typ] {
			if !renderer[p] {
				t.Errorf("%s: Der Vorgabetext (schema.sql) enthält %s, der Renderer ersetzt es nicht.", typ, p)
			}
		}
		if quelle, hat := standardtextQuellen[typ]; hat {
			standard := leseRendererPlatzhalter(t, quelle)
			if len(standard) == 0 {
				t.Fatalf("%s: kein Platzhalter im Standardtext in %s gefunden — Detektor greift nicht mehr", typ, quelle)
			}
			for p := range standard {
				if !renderer[p] {
					t.Errorf("%s: Der Standardtext (%s) enthält %s, der Renderer ersetzt es nicht.", typ, quelle, p)
				}
			}
		}
	}

	// 2) Jeder Typ aus den Vorgabetexten braucht einen Renderer oder eine begründete Ausnahme,
	//    und die Ausnahme darf im Editor nichts anbieten.
	for typ := range vorgaben {
		_, hatRenderer := rendererQuellen[typ]
		grund, istAusnahme := ohneRenderer[typ]
		switch {
		case hatRenderer && istAusnahme:
			t.Errorf("%s steht in rendererQuellen und ohneRenderer — eines von beidem stimmt nicht.", typ)
		case !hatRenderer && !istAusnahme:
			t.Errorf("Der Vorgabetext %s hat keinen Renderer und keine begründete Ausnahme — tote Vorlage.", typ)
		case istAusnahme && strings.TrimSpace(grund) == "":
			t.Errorf("Ausnahme %s ohne Begründung.", typ)
		case istAusnahme:
			if ui, ok := editor[typ]; ok && len(ui) > 0 {
				t.Errorf("%s hat keinen Renderer, der Editor bietet aber Platzhalter an: %v", typ, sortierteMenge(ui))
			}
		}
	}

	// 3) Rückrichtung: Ausnahmen und Blöcke im Editor ohne Typ in den Vorgabetexten veralten
	//    lautlos.
	for typ := range ohneRenderer {
		if _, ok := vorgaben[typ]; !ok {
			t.Errorf("ohneRenderer führt %s, die Vorgabetexte kennen den Typ nicht (mehr) — Eintrag entfernen.", typ)
		}
	}
	for typ := range editor {
		if _, ok := vorgaben[typ]; !ok {
			t.Errorf("vorlagenInfo führt %s, die Vorgabetexte kennen den Typ nicht (mehr) — Block entfernen oder Vorgabetext ergänzen.", typ)
		}
	}
}

// Gegenprobe am Detektor: Ein Muster, das nichts findet, meldet ewig „alles gut".
func TestVorlagenPlatzhalterDetektorGreift(t *testing.T) {
	if m := sammlePlatzhalter(`x := "{{.Vorname}} und {{.Frist}}"`); !m["{{.Vorname}}"] || !m["{{.Frist}}"] {
		t.Error("Platzhalter-Muster erkennt String-Literale nicht")
	}
	if m := sammlePlatzhalter(ohneKommentare("// {{.Kommentar}}\nx := 1")); len(m) != 0 {
		t.Error("Kommentar-Platzhalter dürfen nicht zählen")
	}
	if leseRendererPlatzhalter(t, rendererQuellen["BESTELLUNG_HAENDLER"])["{{.BuchListe}}"] {
		t.Error("Händler-Renderer dürfte {{.BuchListe}} nie führen — Detektor liest offenbar die falsche Quelle")
	}
}
