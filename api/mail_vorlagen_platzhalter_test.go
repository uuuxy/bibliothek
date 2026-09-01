package api

// Platzhalter-Paritäts-Gate (Sweep „gebaut, aber nie verdrahtet", 01.09.2026).
//
// Drei Orte behaupten je Vorlagen-Typ, welche {{.X}}-Platzhalter gelten:
//
//   1. der GO-RENDERER — die einzige Menge, die wirklich ersetzt wird
//      (reports_pdf.go für MAHNUNG_ELTERN, bestellmail_text.go für
//      BESTELLUNG_HAENDLER),
//   2. die ANZEIGE im Vorlagen-Editor (vorlagenInfo in MailTemplates.svelte),
//   3. die SEED-TEXTE (schema.sql).
//
// Bis zum 01.09.2026 zeigte der Editor für JEDE Vorlage dieselben vier
// Platzhalter — für die Händler-Mail ersetzte der Renderer keinen einzigen
// davon: Wer die Vorlage laut Anleitung umformulierte, schickte dem Händler
// wörtlich „{{.BuchListe}}". Und die Seed-Vorlage BESTELLUNG_EINGETROFFEN hat
// überhaupt keinen Renderer — sie ließ sich bearbeiten und speichern, ohne je
// verschickt zu werden. Dieses Gate hält die drei Orte deckungsgleich; die
// bekannte tote Vorlage steht als begründete Ausnahme unten und fliegt raus,
// sobald die Betreiber-Entscheidung fällt (bauen oder austragen).

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var platzhalterMuster = regexp.MustCompile(`\{\{\.[A-Za-z]+\}\}`)

// rendererQuellen: je Typ die Go-Datei, deren STRING-LITERALE die wirksame
// Platzhalter-Menge tragen (Replacer, Split, Fallback-Vorlage).
var rendererQuellen = map[string]string{
	"MAHNUNG_ELTERN":      "reports_pdf.go",
	"BESTELLUNG_HAENDLER": "bestellmail_text.go",
}

// ohneRenderer: Seed-Typen, die BEWUSST keinen Renderer haben — mit Begründung.
// Ein Eintrag hier ist ein geparkter Befund, kein Normalzustand. Der bisher
// einzige Fall (BESTELLUNG_EINGETROFFEN) ist mit Migration 092 ausgetragen:
// Die Abholbereit-Benachrichtigung ist der Abholfach-Hinweis am Theken-Terminal
// (OmniboxService), keine Mail — Betreiber-Entscheidung 01.09.2026.
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

// leseRendererPlatzhalter liest die wirksame Menge aus den String-Literalen der
// Renderer-Datei. Kommentare zählen nicht (dort wird über Platzhalter GEREDET,
// nicht ersetzt) — dieselbe ohneKommentare-Technik wie das Schichtungs-Gate.
func leseRendererPlatzhalter(t *testing.T, datei string) map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(datei)
	if err != nil {
		t.Fatalf("%s lesen: %v", datei, err)
	}
	return sammlePlatzhalter(ohneKommentare(string(roh)))
}

// leseEditorPlatzhalter parst vorlagenInfo aus MailVorlagenPlatzhalter.svelte:
// je Typ-Block die platzhalter-Liste (und dass es den Block überhaupt gibt).
func leseEditorPlatzhalter(t *testing.T) map[string]map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(filepath.Join("..", "frontend", "src", "lib", "MailVorlagenPlatzhalter.svelte"))
	if err != nil {
		t.Fatalf("MailVorlagenPlatzhalter.svelte lesen: %v", err)
	}
	blockMuster := regexp.MustCompile(`(?s)([A-Z_]+):\s*\{\s*verwendung:.*?platzhalter:\s*\[(.*?)\]`)
	out := map[string]map[string]bool{}
	for _, m := range blockMuster.FindAllStringSubmatch(string(roh), -1) {
		out[m[1]] = sammlePlatzhalter(m[2])
	}
	if len(out) < 2 {
		t.Fatalf("nur %d Typ-Blöcke in vorlagenInfo erkannt — Parser oder Komponente prüfen", len(out))
	}
	return out
}

// leseSeedVorlagen liefert je Seed-Typ die Platzhalter seines Vorlagentexts.
func leseSeedVorlagen(t *testing.T) map[string]map[string]bool {
	t.Helper()
	roh, err := os.ReadFile(filepath.Join("..", "schema.sql"))
	if err != nil {
		t.Fatalf("schema.sql lesen: %v", err)
	}
	inhalt := string(roh)
	start := strings.Index(inhalt, "INSERT INTO mail_vorlagen")
	ende := strings.Index(inhalt[start:], "ON CONFLICT (typ)")
	if start < 0 || ende < 0 {
		t.Fatal("mail_vorlagen-Seed in schema.sql nicht gefunden")
	}
	seed := inhalt[start : start+ende]

	typMuster := regexp.MustCompile(`'([A-Z_]+)',`)
	treffer := typMuster.FindAllStringSubmatchIndex(seed, -1)
	out := map[string]map[string]bool{}
	for i, m := range treffer {
		typ := seed[m[2]:m[3]]
		blockEnde := len(seed)
		if i+1 < len(treffer) {
			blockEnde = treffer[i+1][0]
		}
		out[typ] = sammlePlatzhalter(seed[m[0]:blockEnde])
	}
	if len(out) < 2 {
		t.Fatalf("nur %d Seed-Vorlagen erkannt — Parser prüfen", len(out))
	}
	return out
}

func TestVorlagenPlatzhalterSindDeckungsgleich(t *testing.T) {
	editor := leseEditorPlatzhalter(t)
	seeds := leseSeedVorlagen(t)

	// 1) Je Renderer-Typ: Editor-Anzeige ≡ wirksame Renderer-Menge, und die
	//    Seed-Vorlage benutzt nichts, was der Renderer nicht ersetzt.
	for typ, datei := range rendererQuellen {
		renderer := leseRendererPlatzhalter(t, datei)
		if len(renderer) == 0 {
			t.Fatalf("%s: keine Platzhalter in %s gefunden — Detektor greift nicht mehr", typ, datei)
		}
		ui, ok := editor[typ]
		if !ok {
			t.Errorf("%s fehlt in vorlagenInfo (MailTemplates.svelte) — der Editor zeigt dann nichts an.", typ)
			continue
		}
		for p := range renderer {
			if !ui[p] {
				t.Errorf("%s: Renderer ersetzt %s, der Editor verschweigt es — unentdeckbares Feature.", typ, p)
			}
		}
		for p := range ui {
			if !renderer[p] {
				t.Errorf("%s: Editor bietet %s an, der Renderer (%s) ersetzt es NICHT — der Platzhalter stünde wörtlich im Versand.", typ, p, datei)
			}
		}
		for p := range seeds[typ] {
			if !renderer[p] {
				t.Errorf("%s: Seed-Vorlage (schema.sql) enthält %s, der Renderer ersetzt es nicht.", typ, p)
			}
		}
	}

	// 2) Jeder Seed-Typ braucht einen Renderer ODER eine begründete Ausnahme —
	//    und die Ausnahme darf im Editor nichts anbieten.
	for typ := range seeds {
		_, hatRenderer := rendererQuellen[typ]
		grund, istAusnahme := ohneRenderer[typ]
		switch {
		case hatRenderer && istAusnahme:
			t.Errorf("%s steht in rendererQuellen UND ohneRenderer — eines von beidem stimmt nicht.", typ)
		case !hatRenderer && !istAusnahme:
			t.Errorf("Seed-Vorlage %s hat keinen Renderer und keine begründete Ausnahme — tote Vorlage (Klasse F1).", typ)
		case istAusnahme && strings.TrimSpace(grund) == "":
			t.Errorf("Ausnahme %s ohne Begründung.", typ)
		case istAusnahme:
			if ui, ok := editor[typ]; ok && len(ui) > 0 {
				t.Errorf("%s hat keinen Renderer, der Editor bietet aber Platzhalter an: %v", typ, sortierteMenge(ui))
			}
		}
	}

	// 3) Rückrichtung: Ausnahmen und Editor-Blöcke ohne Seed-Typ veralten lautlos.
	for typ := range ohneRenderer {
		if _, ok := seeds[typ]; !ok {
			t.Errorf("ohneRenderer führt %s, der Seed kennt den Typ nicht (mehr) — Eintrag entfernen.", typ)
		}
	}
	for typ := range editor {
		if _, ok := seeds[typ]; !ok {
			t.Errorf("vorlagenInfo führt %s, der Seed kennt den Typ nicht (mehr) — Block entfernen oder Seed ergänzen.", typ)
		}
	}
}

// Gegenprobe am Detektor (Regel 2 des Sweep-Registers): Ein Muster, das nichts
// findet, meldet ewig „alles gut".
func TestVorlagenPlatzhalterDetektorGreift(t *testing.T) {
	if m := sammlePlatzhalter(`x := "{{.Vorname}} und {{.Frist}}"`); !m["{{.Vorname}}"] || !m["{{.Frist}}"] {
		t.Error("Platzhalter-Muster erkennt String-Literale nicht")
	}
	if m := sammlePlatzhalter(ohneKommentare("// {{.Kommentar}}\nx := 1")); len(m) != 0 {
		t.Error("Kommentar-Platzhalter dürfen nicht zählen")
	}
	if leseRendererPlatzhalter(t, "bestellmail_text.go")["{{.BuchListe}}"] {
		t.Error("Händler-Renderer dürfte {{.BuchListe}} nie führen — Detektor liest offenbar die falsche Quelle")
	}
}
