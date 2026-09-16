package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// „Überfällig" heißt überall dasselbe — und eine Dauerleihe wird es nie.
//
// Entschieden am 16.09.2026: „kollegen haben keine frist bzw werden einfach nie gesperrt!"
// Eine Ausleihe an jemanden, der kein Schüler ist, ist eine Dauerleihe
// (`ausleihen.ist_handapparat`, gesetzt in erzeugeAusleihe und im Geräte-Pfad).
//
// Die Regel gab es schon: Die Sperr-Automatik zählt seit jeher nur Ausleihen mit
// `ist_handapparat = false`. Sie war nur nicht zu Ende geführt — die Leserliste, die
// Übersicht und die Ersatzforderung rechneten ihr eigenes „überfällig" aus dem blossen
// Datum. Folge: Ein Kollege stand nach einem Jahr mit roter Zahl in der Leserdatei und
// als Mahnfall in der Statistik, während die Theke ihn anstandslos bediente — zwei
// Wahrheiten über dieselbe Ausleihe.
//
// Dieser Detektor hält die Regel, weil die nächste Abfrage sonst wieder danebenfällt: Wer
// `rueckgabe_frist < CURRENT_TIMESTAMP` schreibt, muss entweder Dauerleihen ausschliessen
// oder ohnehin nur Schüler sehen (die Sicht `schueler` im JOIN/FROM — so macht es der
// Mahnlauf, und das ist die andere richtige Antwort).
//
// Gemessen wird am SQL-Text, nicht an einer Dateiliste: Eine Liste wäre nach der ersten
// neuen Abfrage überholt, ohne rot zu werden.
var ueberfaelligPruefung = regexp.MustCompile(`rueckgabe_frist\s*<\s*(CURRENT_TIMESTAMP|now\(\))`)

// verzeichnisse: relativ zu repository/ — die drei Orte, an denen SQL steht.
var verzeichnisse = []string{".", "../api", "../internal"}

func TestUeberfaelligOhneDauerleihen(t *testing.T) {
	var verstoesse []string
	gepruefte := 0

	for _, dir := range verzeichnisse {
		// Der Fehler wird geprüft: Ein nicht lesbares Verzeichnis liesse den Detektor still
		// über die halbe Anwendung hinweggehen und trotzdem grün melden.
		if gehErr := filepath.Walk(dir, func(pfad string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
				return nil
			}
			roh, leseErr := os.ReadFile(filepath.Clean(pfad))
			if leseErr != nil {
				return nil
			}
			for _, block := range sqlBloecke(string(roh)) {
				if !ueberfaelligPruefung.MatchString(block) {
					continue
				}
				gepruefte++
				// Zwei erlaubte Antworten: Dauerleihen ausschliessen ODER nur Schüler sehen.
				nurSchueler := strings.Contains(block, "JOIN schueler") || strings.Contains(block, "FROM schueler")
				if strings.Contains(block, "ist_handapparat") || nurSchueler {
					continue
				}
				verstoesse = append(verstoesse, pfad)
			}
			return nil
		}); gehErr != nil {
			t.Fatalf("%s nicht lesbar: %v", dir, gehErr)
		}
	}

	// Ohne diese Zusicherung wäre eine umbenannte Spalte ein still grüner Test.
	if gepruefte == 0 {
		t.Fatal("keine einzige Überfälligkeits-Abfrage gefunden — der Detektor misst nichts mehr")
	}

	if len(verstoesse) > 0 {
		t.Errorf("Diese Abfragen entscheiden „überfällig\" allein am Datum:\n  %s\n\n"+
			"Eine Dauerleihe (ist_handapparat) wird nicht überfällig — ein Kollege hat keine Frist.\n"+
			"Entweder `AND ist_handapparat = false` ergänzen oder über die Sicht `schueler` gehen,\n"+
			"die ohnehin nur Schüler zeigt (so macht es der Mahnlauf).",
			strings.Join(verstoesse, "\n  "))
	}
}

// sqlBloecke liefert die Inhalte aller Backtick-Zeichenketten — dort steht das SQL.
// Ein Block ist die Einheit, in der eine Bedingung steht; eine zeilenweise Prüfung
// könnte den Ausschluss zwei Zeilen weiter nicht sehen.
func sqlBloecke(quelle string) []string {
	var out []string
	rest := quelle
	for {
		start := strings.Index(rest, "`")
		if start < 0 {
			return out
		}
		rest = rest[start+1:]
		ende := strings.Index(rest, "`")
		if ende < 0 {
			return out
		}
		out = append(out, rest[:ende])
		rest = rest[ende+1:]
	}
}
