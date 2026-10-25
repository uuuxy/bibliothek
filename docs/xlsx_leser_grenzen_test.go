package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Jede XLSX-Lesestelle geht durch xlsxgrenze.LeseMappe; excelize.OpenReader steht nur dort.
//
// LeseMappe begrenzt den Speicher gegen eine Entpack-Bombe aus einer winzigen Datei, fängt
// Abstürze der Bibliothek ab und weist OLE-Container ab. Dazu kommt GO-2026-6452: excelize
// schützt den gewöhnlichen Weg gegen negative Zeichenketten-Indizes, den Weg über die
// Auslagerung aber nicht. Erreichbar wird der nur, wenn die XML-Grenze kleiner ist als die
// Gesamtgrenze, und genau das verhindert Optionen(), indem es beide gleich setzt
// (pkg/xlsxgrenze/negativer_sharedstring_test.go misst es). Ein direkter Aufruf von
// OpenReader ginge an allem vorbei; eine weitere Lesestelle ist deshalb eine Frage.
var xlsxLeserBestand = map[string]string{
	"api/system/littera_import.go": "Littera-Katalogübernahme (POST /api/import/littera, manage_inventory)",
	"internal/lusd/quelle.go":      "LUSD-Schülerabgleich (POST /api/lusd/preview und /import, import_students)",
	"api/katalog/listenimport.go":  "Listenimport des Bestands",
}

// einzigerOpenReader ist die eine Datei, die excelize.OpenReader rufen darf; dass die
// Zeile dort die Optionen mit den Grenzen trägt, prüft dieser Test gleich mit.
const einzigerOpenReader = "pkg/xlsxgrenze/xlsxgrenze.go"

var (
	musterOpenReader = regexp.MustCompile(`excelize\.OpenReader\(`)
	musterLeseMappe  = regexp.MustCompile(`xlsxgrenze\.LeseMappe\(`)
)

func TestXLSXLeser_JederDurchLeseMappe(t *testing.T) {
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
		if musterLeseMappe.MatchString(quelle) {
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
			"heißt der Weg noch xlsxgrenze.LeseMappe?")
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
			"OLE-Abweisung und Absturz-Schranke vorbei — xlsxgrenze.LeseMappe nehmen.", p)
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
