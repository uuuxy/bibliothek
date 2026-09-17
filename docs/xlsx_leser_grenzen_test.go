package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Jeder XLSX-Leser MUSS xlsxgrenze.Optionen() mitgeben.
//
// Das war bis zum 17.09.2026 eine Frage des Speichers (eine Entpack-Bombe aus einer
// winzigen Datei, Sicherheits-Audit 07.09.2026). Seit GO-2026-6452 ist es zusätzlich eine
// Frage der Sicherheit, und zwar eine überraschende: excelize schützt den gewöhnlichen
// Weg gegen negative Zeichenketten-Indizes, den AUSLAGERUNGS-Weg aber nicht. Erreichbar
// wird der nur, wenn die XML-Grenze KLEINER ist als die Gesamtgrenze — und genau das
// verhindert Optionen(), indem es beide gleich setzt
// (pkg/xlsxgrenze/negativer_sharedstring_test.go misst es).
//
// Ein Leser ohne diese Optionen bekommt die Vorgaben von excelize, und die sind nicht
// gleich: UnzipSizeLimit 16 GB gegen UnzipXMLSizeLimit = StreamChunkSize. Damit lagert er
// aus, und der ungeschützte Weg steht offen. Deshalb: **Ein vierter Leser ist eine Frage.**
var xlsxLeserBestand = map[string]string{
	"api/littera_import.go":     "Littera-Katalogübernahme (POST /api/import/littera, manage_inventory)",
	"api/lusd_parser_quelle.go": "LUSD-Schülerabgleich (POST /api/lusd/preview und /import, import_students)",
	"inventur/excel_import.go":  "Listenimport des Bestands",
}

var musterOpenReader = regexp.MustCompile(`excelize\.OpenReader\(`)

func TestXlsxLeser_JederMitGrenzen(t *testing.T) {
	gefunden := map[string]int{}
	ohneGrenzen := map[string]bool{}
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
		for _, zeile := range strings.Split(quelle, "\n") {
			if !musterOpenReader.MatchString(zeile) {
				continue
			}
			gefunden[rel]++
			if !strings.Contains(zeile, "xlsxgrenze.Optionen()") {
				ohneGrenzen[rel] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Baum lesen: %v", err)
	}
	if len(gefunden) == 0 {
		t.Fatal("kein einziger XLSX-Leser gefunden — der Detektor misst nichts; " +
			"heißt die Bibliothek noch excelize?")
	}

	var neu, ohne, veraltet []string
	for pfad := range gefunden {
		if _, bekannt := xlsxLeserBestand[pfad]; !bekannt {
			neu = append(neu, pfad)
		}
		if ohneGrenzen[pfad] {
			ohne = append(ohne, pfad)
		}
	}
	for pfad, zweck := range xlsxLeserBestand {
		if gefunden[pfad] == 0 {
			veraltet = append(veraltet, pfad)
		}
		if zweck == "" {
			t.Errorf("%s steht ohne Zweck im Bestand", pfad)
		}
	}
	sort.Strings(neu)
	sort.Strings(ohne)
	sort.Strings(veraltet)

	for _, p := range ohne {
		t.Errorf("%s öffnet eine XLSX-Datei OHNE xlsxgrenze.Optionen(). Mit den Vorgaben von "+
			"excelize sind Gesamt- und XML-Grenze verschieden; dann lagert es die "+
			"Zeichenkettentabelle aus, und dort fehlt der Schutz gegen negative Indizes "+
			"(GO-2026-6452) — eine präparierte Datei reißt die Anfrage ab.", p)
	}
	for _, p := range neu {
		t.Errorf("%s ist ein neuer XLSX-Leser. Bitte mit seinem Zweck in xlsxLeserBestand "+
			"eintragen — und sicherstellen, dass er xlsxgrenze.Optionen() mitgibt.", p)
	}
	for _, p := range veraltet {
		t.Errorf("%s liest keine XLSX-Datei mehr — bitte aus xlsxLeserBestand austragen, "+
			"damit der Bestand weiter etwas aussagt.", p)
	}
}
