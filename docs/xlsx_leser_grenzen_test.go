package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Jede XLSX-Lesestelle geht durch xlsxgrenze.MitMappe — excelize.OpenReader steht nur
// noch dort.
//
// Das war bis zum 17.09.2026 eine Frage des Speichers (eine Entpack-Bombe aus einer
// winzigen Datei, Sicherheits-Audit 07.09.2026). Seit GO-2026-6452 ist es zusätzlich eine
// Frage der Sicherheit, und zwar eine überraschende: excelize schützt den gewöhnlichen
// Weg gegen negative Zeichenketten-Indizes, den AUSLAGERUNGS-Weg aber nicht. Erreichbar
// wird der nur, wenn die XML-Grenze KLEINER ist als die Gesamtgrenze — und genau das
// verhindert Optionen(), indem es beide gleich setzt
// (pkg/xlsxgrenze/negativer_sharedstring_test.go misst es). Seit dem 08.10.2026 trägt
// MitMappe zusätzlich die Schranke gegen Abstürze der Bibliothek und die Abweisung von
// OLE-Containern (OFFEN.md 5.10); ein direkter OpenReader-Aufruf ginge an beidem vorbei.
// Deshalb: **Eine vierte Lesestelle ist eine Frage.**
var xlsxLeserBestand = map[string]string{
	"api/littera_import.go":    "Littera-Katalogübernahme (POST /api/import/littera, manage_inventory)",
	"internal/lusd/quelle.go":  "LUSD-Schülerabgleich (POST /api/lusd/preview und /import, import_students)",
	"inventur/excel_import.go": "Listenimport des Bestands",
}

// einzigerOpenReader ist die eine Datei, die excelize.OpenReader rufen darf; dass die
// Zeile dort die Optionen mit den Grenzen trägt, prüft dieser Test gleich mit.
const einzigerOpenReader = "pkg/xlsxgrenze/xlsxgrenze.go"

var (
	musterOpenReader = regexp.MustCompile(`excelize\.OpenReader\(`)
	musterMitMappe   = regexp.MustCompile(`xlsxgrenze\.MitMappe\(`)
)

func TestXlsxLeser_JederDurchMitMappe(t *testing.T) {
	leser := map[string]int{}
	direkt := map[string]bool{}
	wurzel := ".."
	err := filepath.Walk(wurzel, func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		roh, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		quelle := musterZeilenKomm.ReplaceAllString(string(roh), "")
		rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(pfad, wurzel)), "/")
		if musterMitMappe.MatchString(quelle) {
			leser[rel]++
		}
		if musterOpenReader.MatchString(quelle) && rel != einzigerOpenReader {
			direkt[rel] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	if len(leser) == 0 {
		t.Fatal("keine einzige XLSX-Lesestelle gefunden — der Detektor misst nichts; " +
			"heißt der Weg noch xlsxgrenze.MitMappe?")
	}
	tuer, err := os.ReadFile(filepath.Join(wurzel, einzigerOpenReader))
	if err != nil || !musterOpenReader.Match(tuer) {
		t.Fatalf("%s ruft excelize.OpenReader nicht mehr — die eine Tür ist umgezogen; "+
			"diesen Test mitziehen (Fehler: %v)", einzigerOpenReader, err)
	}
	for _, zeile := range strings.Split(string(tuer), "\n") {
		if musterOpenReader.MatchString(zeile) && !strings.Contains(zeile, "Optionen()") {
			t.Errorf("%s öffnet ohne Optionen(): %q — ohne die Grenzen steht der "+
				"Auslagerungs-Weg der Zeichenkettentabelle offen (GO-2026-6452).",
				einzigerOpenReader, strings.TrimSpace(zeile))
		}
	}

	var neu, vorbei, veraltet []string
	for pfad := range leser {
		if _, bekannt := xlsxLeserBestand[pfad]; !bekannt {
			neu = append(neu, pfad)
		}
	}
	for pfad := range direkt {
		vorbei = append(vorbei, pfad)
	}
	for pfad, zweck := range xlsxLeserBestand {
		if leser[pfad] == 0 {
			veraltet = append(veraltet, pfad)
		}
		if zweck == "" {
			t.Errorf("%s steht ohne Zweck im Bestand", pfad)
		}
	}
	sort.Strings(neu)
	sort.Strings(vorbei)
	sort.Strings(veraltet)

	for _, p := range vorbei {
		t.Errorf("%s ruft excelize.OpenReader direkt und geht damit an Entpackgrenze, "+
			"OLE-Abweisung und Absturz-Schranke vorbei — xlsxgrenze.MitMappe nehmen.", p)
	}
	for _, p := range neu {
		t.Errorf("%s ist eine neue XLSX-Lesestelle. Bitte mit ihrem Zweck in "+
			"xlsxLeserBestand eintragen.", p)
	}
	for _, p := range veraltet {
		t.Errorf("%s liest keine XLSX-Datei mehr — bitte aus xlsxLeserBestand austragen, "+
			"damit der Bestand weiter etwas aussagt.", p)
	}
}
