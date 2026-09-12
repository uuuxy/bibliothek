package api

import (
	"os"
	"regexp"
	"testing"
)

// Gate gegen eine Zahl, die niemand nachzieht: FACHKONZEPT.md sagt, wie viele Bereiche die
// Betriebsbereitschaft prüft, und Pruefe sagt es noch einmal — als Liste.
//
// Anlass (Raster-Durchgang 12.09.2026): Bis zum 11.09.2026 stand im FACHKONZEPT „Geprüft
// werden fünfzehn Bereiche", während Pruefe schon sechzehn Befunde lieferte. Aufgefallen
// ist das nur, weil derselbe Commit den siebzehnten Bereich baute und die Zahl anfasste.
// Eine Zahl in der Prosa ist die billigste Art, ein Dokument still falsch werden zu lassen:
// Wer sie liest, prüft nicht nach, und wer einen Bereich ergänzt, liest die Prosa nicht.
//
// Grenze: Das Gate prüft die ZAHL, nicht die Aufzählung dahinter — die nennt ausdrücklich
// nur einen Teil („darunter"). Ein neuer Bereich muss also die Zahl nachziehen; ob er auch
// beschrieben gehört, bleibt eine Frage an den Menschen.
func TestFachkonzeptNenntDieZahlDerGeprueftenBereiche(t *testing.T) {
	zahlwoerter := map[string]int{
		"zehn": 10, "elf": 11, "zwölf": 12, "dreizehn": 13, "vierzehn": 14, "fünfzehn": 15,
		"sechzehn": 16, "siebzehn": 17, "achtzehn": 18, "neunzehn": 19, "zwanzig": 20,
		"einundzwanzig": 21, "zweiundzwanzig": 22, "dreiundzwanzig": 23, "vierundzwanzig": 24,
		"fünfundzwanzig": 25,
	}

	roh, err := os.ReadFile("../docs/FACHKONZEPT.md")
	if err != nil {
		t.Fatalf("FACHKONZEPT.md lesen: %v", err)
	}
	treffer := regexp.MustCompile(`Geprüft werden ([a-zäöüß]+) Bereiche`).FindSubmatch(roh)
	if treffer == nil {
		t.Fatal("FACHKONZEPT.md nennt die Zahl der geprüften Bereiche nicht mehr in der Form " +
			"„Geprüft werden <Zahlwort> Bereiche" + "“ — Satz oder Gate nachziehen, sonst prüft hier nichts mehr")
	}
	genannt, bekannt := zahlwoerter[string(treffer[1])]
	if !bekannt {
		t.Fatalf("Zahlwort %q ist dem Gate unbekannt — in zahlwoerter ergänzen", treffer[1])
	}

	// Pruefe liefert je Bereich genau einen Befund, unabhängig von der Lage.
	gebaut := len(Pruefe(Lage{}))
	if genannt != gebaut {
		t.Errorf("FACHKONZEPT.md nennt %d Bereiche (%q), Pruefe liefert %d — "+
			"die Zahl im Dokument nachziehen", genannt, treffer[1], gebaut)
	}
}
