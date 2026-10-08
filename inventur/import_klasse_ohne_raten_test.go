package inventur

import (
	"context"
	"testing"
)

// Der Listenimport nimmt den Jahrgang nur aus der Spalte „klasse" und rät ihn nicht aus einer
// Zahl im Titel: „Die 13½ Leben des Käpt'n Blaubär" bekäme sonst Jahrgang 13. Ein geratener
// Jahrgang ist in der Datenbank von einem gepflegten nicht zu unterscheiden. Die Spalte ergibt
// die Spanne N bis N.
func TestVerarbeiteImportZeile_JahrgangNurAusDerSpalte(t *testing.T) {
	faelle := []struct {
		name      string
		titel     string
		mitSpalte bool
		klasse    string
		want      int
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
			if buch.JahrgangVon != f.want || buch.JahrgangBis != f.want {
				t.Errorf("Titel %q, Spalte %q: Jahrgang %d bis %d, erwartet %d bis %d",
					f.titel, f.klasse, buch.JahrgangVon, buch.JahrgangBis, f.want, f.want)
			}
		})
	}
}
