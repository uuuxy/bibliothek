package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Gate gegen eine Kopfzeile, die einen Stand behauptet, den die Datei nicht mehr hat.
//
// Anlass (23.08.2026): Vier Dokumente trugen „Zuletzt aktualisiert: 2026-08-06" bzw.
// „Stand 2026-08-05" im Kopf und beschrieben darunter Vorgänge vom 11., 21. und 23.
// August. Wer die Kopfzeile liest, um zu entscheiden, ob er dem Dokument traut, wurde
// also gezielt in die Irre geführt — und zwar von der einzigen Zeile, die genau dafür
// da ist.
//
// Geprüft wird ohne git, nur aus dem Text: Kein Datum IM Dokument darf jünger sein als
// das Datum, das der Kopf behauptet. Das trifft genau den Fehlerfall (jemand ergänzt
// einen datierten Abschnitt und vergisst den Kopf) und erzeugt keine Fehlalarme durch
// Formatierungs-Commits, die den git-Zeitstempel verschieben, ohne etwas zu sagen.
//
// Dokumente OHNE Kopfzeile prüft das Gate nicht — nicht jedes braucht eine (FACHKONZEPT
// und resilience_and_recovery führen bewusst keine). Wer keine Zusicherung gibt, kann
// auch keine brechen.
//
// Reparatur bei Rot: Das Datum im Kopf auf den jüngsten im Text genannten Stand ziehen —
// oder die Kopfzeile entfernen, wenn sie ohnehin niemand pflegt.
func TestStandAngabenNichtVeraltet(t *testing.T) {
	kopfMuster := regexp.MustCompile(`(?i)(Zuletzt aktualisiert|Stand):?\s*\**\s*(\d{4}-\d{2}-\d{2})`)
	// Beide Schreibweisen, die im Projekt vorkommen: 2026-08-23 und 23.08.2026.
	datumMuster := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})|(\d{2})\.(\d{2})\.(\d{4})`)

	const kopfZeilen = 12

	dateien, err := filepath.Glob("*.md")
	if err != nil {
		t.Fatalf("Dokumente suchen: %v", err)
	}
	unterordner, err := filepath.Glob("*/*.md")
	if err != nil {
		t.Fatalf("Unterordner durchsuchen: %v", err)
	}
	dateien = append(dateien, unterordner...)

	if len(dateien) < 10 {
		t.Fatalf("nur %d Dokumente gefunden — der Sammler greift vermutlich nicht mehr", len(dateien))
	}

	var geprueft int
	for _, datei := range dateien {
		rohdaten, readErr := os.ReadFile(datei)
		if readErr != nil {
			t.Fatalf("%s lesen: %v", datei, readErr)
		}
		inhalt := string(rohdaten)

		// NUR im Kopf suchen, nicht im ganzen Text: In befunde.md steht "Stand: 2026-08-06"
		// als Abschluss eines Abschnitts (Zeile 663), in invarianten.md mitten in einem Satz.
		// Beides sind Aussagen über einen Abschnitt, keine Zusicherung über das Dokument —
		// ohne diese Grenze meldete das Gate zwei Dateien falsch-rot. Ein Detektor, der
		// Richtiges anzeigt, wird abgeschaltet.
		zeilen := strings.SplitN(inhalt, "\n", kopfZeilen+1)
		kopfbereich := strings.Join(zeilen[:min(len(zeilen), kopfZeilen)], "\n")

		kopf := kopfMuster.FindStringSubmatch(kopfbereich)
		if kopf == nil {
			continue // Kein Stand behauptet, also nichts zu brechen.
		}
		geprueft++
		behauptet := kopf[2]

		var daten []string
		for _, treffer := range datumMuster.FindAllStringSubmatch(inhalt, -1) {
			if treffer[1] != "" {
				daten = append(daten, treffer[1]+"-"+treffer[2]+"-"+treffer[3])
				continue
			}
			daten = append(daten, treffer[6]+"-"+treffer[5]+"-"+treffer[4])
		}
		sort.Strings(daten)
		juengstes := daten[len(daten)-1]

		if juengstes > behauptet {
			t.Errorf("%s behauptet im Kopf den Stand %s, beschreibt im Text aber Vorgänge bis %s.\n"+
				"→ Die Kopfzeile auf %s ziehen — oder sie entfernen, wenn sie niemand pflegt.",
				datei, behauptet, juengstes, juengstes)
		}
	}

	if geprueft < 3 {
		t.Errorf("nur %d Dokumente mit Stand-Angabe erkannt — das Kopfmuster greift vermutlich nicht mehr "+
			"(erwartet werden mindestens ARCHITECTURE, DEPLOYMENT, SECURITY)", geprueft)
	}
}
