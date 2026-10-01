package repository

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Die Frist des Papierkorbs steht zweimal: als Konstante der Löschläufe
// (StandardAnonymisierungSoftDeleteTage) und als Zahl in den Sätzen, mit denen die Oberfläche
// sie ankündigt — im Löschdialog der Akte, in der Liste des Papierkorbs und in der
// Beschreibung des Rechts zum endgültigen Löschen. Ändert sich die Konstante, nennen die Sätze
// sonst weiter die alte Zahl.
//
// Gelesen werden die Dateien als Text. Nennt eine von ihnen keine Frist mehr, ist der Test rot
// und nicht still grün.
//
// Blind für eine weitere Stelle der Oberfläche, die die Frist nennt.
func TestPapierkorbFrist_OberflaecheNenntDieZahlDerLoeschlaeufe(t *testing.T) {
	soll := fmt.Sprintf("%d Tage", StandardAnonymisierungSoftDeleteTage)
	frist := regexp.MustCompile(`\d+ Tage`)
	for _, stelle := range []struct {
		datei string
		// nurZeilenMit grenzt in einer Datei mit vielen Texten auf die Zeilen zum Papierkorb ein.
		nurZeilenMit string
	}{
		{"../frontend/src/lib/StudentProfileDeleteModal.svelte", ""},
		{"../frontend/src/lib/components/students/DeletedStudentList.svelte", "Papierkorb"},
		{"../frontend/src/lib/permissionMetadata.js", "Papierkorb"},
	} {
		quelle, err := os.ReadFile(stelle.datei)
		if err != nil {
			t.Fatalf("%s lesen: %v", stelle.datei, err)
		}
		var genannt []string
		for _, zeile := range strings.Split(string(quelle), "\n") {
			if strings.Contains(zeile, stelle.nurZeilenMit) {
				genannt = append(genannt, frist.FindAllString(zeile, -1)...)
			}
		}
		if len(genannt) == 0 {
			t.Fatalf("%s nennt keine Frist in Tagen mehr — Form geändert? Dann diesen Test "+
				"nachziehen, sonst prüft er nichts.", stelle.datei)
		}
		for _, g := range genannt {
			if g != soll {
				t.Errorf("%s nennt „%s“, die Löschläufe rechnen mit „%s“", stelle.datei, g, soll)
			}
		}
	}
}
