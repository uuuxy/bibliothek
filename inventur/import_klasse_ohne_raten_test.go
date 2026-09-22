package inventur

import (
	"context"
	"testing"
)

// Der Listenimport nimmt die Klasse nur aus der Spalte „klasse". Bis zum 22.09.2026 riet er
// sie aus der ersten Zahl im Titel, wenn die Spalte fehlte, leer oder keine Zahl war
// (inferGradeLevelFromTitle): „Die 13½ Leben des Käpt'n Blaubär" bekam Klasse 13. Eine
// geratene Klasse ist in der Datenbank von einer gepflegten nicht zu unterscheiden, und
// genau diese Unterscheidung braucht die Entscheidung, wie die Klasse in die Spanne
// übergeht (docs/OFFEN.md 5.5).
func TestVerarbeiteImportZeile_KlasseNurAusDerSpalte(t *testing.T) {
	faelle := []struct {
		name      string
		titel     string
		mitSpalte bool
		klasse    string
		want      int16
	}{
		{"Zahl im Titel, keine Spalte", "Die 13½ Leben des Käpt'n Blaubär", false, "", 0},
		{"Schulbuchtitel, keine Spalte", "Mathematik 7", false, "", 0},
		{"Spalte leer", "Mathematik 7", true, "", 0},
		{"Spalte ohne Zahl", "Deutsch 9", true, "abc", 0},
		{"Spalte gewinnt, Titel zählt nicht", "Mathematik 7", true, "8", 8},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			zeile := []string{"9783161484100", f.titel, "Ein Autor"}
			spalten := map[string]int{"isbn": 0, "titel": 1, "autor": 2, "fach": -1, "klasse": -1, "bestand": -1}
			if f.mitSpalte {
				zeile = append(zeile, f.klasse)
				spalten["klasse"] = 3
			}
			buch, err := verarbeiteImportZeile(ImportConfig{
				Ctx:       context.Background(),
				Row:       zeile,
				ColIdx:    spalten,
				Metadaten: offlineMetadatenClient(),
			})
			if err != nil || buch == nil {
				t.Fatalf("Zeile: %v", err)
			}
			if buch.GradeLevel != f.want {
				t.Errorf("Titel %q, Spalte %q: Klasse %d, erwartet %d", f.titel, f.klasse, buch.GradeLevel, f.want)
			}
		})
	}
}
