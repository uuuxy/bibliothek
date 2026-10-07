package littera

import (
	"os"
	"strings"
	"testing"

	"bibliothek/internal/uebernahme"
)

// Littera führt die Auflage als Freitext am Titel. In der Sicherung von 2010 tragen 3.182 von
// 10.732 Titeln eine Angabe in 1.033 verschiedenen Werten („1. Aufl.", „2.", „Sonderausg."),
// keiner länger als 50 Zeichen; 23 haben ein Leerzeichen am Rand, 4 zwei Leerzeichen in Folge.
func TestLeseTitel_Auflage(t *testing.T) {
	titel, err := LeseTitel(strings.NewReader("Buchungsnummer,Haupttitel,Auflage\n" +
		`1,"Faust","2. Aufl. "` + "\n" +
		`2,"Die Welle",""` + "\n" +
		`3,"Zeile ohne die Spalte"` + "\n"))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if len(titel) != 3 {
		t.Fatalf("drei Titel erwartet, waren %d", len(titel))
	}
	for i, soll := range []string{"2. Aufl.", "", ""} {
		if titel[i].Auflage != soll {
			t.Errorf("Titel %s: Auflage %q, erwartet %q", titel[i].ID, titel[i].Auflage, soll)
		}
	}
}

// Die Auflage kommt in der Form der Titeltexte an und passt in die Spalte. Der lange Wert ist
// einer aus Littera, um seinen abgeschnittenen Rest ergänzt: Dort endet er nach 50 Zeichen.
func TestFelder_Auflage(t *testing.T) {
	faelle := []struct {
		name, roh, soll string
		gekuerzt        bool
	}{
		{"wie in Littera", "19. völlig neubearb. Aufl.", "19. völlig neubearb. Aufl.", false},
		{"zwei Leerzeichen in Folge", "13. völlig überarb.  Aufl.", "13. völlig überarb. Aufl.", false},
		{"länger als die Spalte", "3., vollst. überarb. und erg. Aufl. / hrsg. und bearb. von der Redaktion",
			"3., vollst. überarb. und erg. Aufl. / hrsg. und be", true},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			pfad := t.TempDir() + "/protokoll.md"
			prot, err := uebernahme.NeuesProtokoll(pfad, "Littera-ID")
			if err != nil {
				t.Fatalf("NeuesProtokoll: %v", err)
			}
			l := &bestandslauf{s: &Schreiber{prot: prot}, ab: &Altbestand{}}

			got := l.felder(Titel{ID: "T1", Haupttitel: "Ein Buch", Auflage: f.roh}).auflage
			if got == nil || *got != f.soll {
				t.Fatalf("auflage = %v, erwartet %q", zeige(got), f.soll)
			}
			if n := len([]rune(*got)); n > uebernahme.MaxAuflage {
				t.Errorf("%d Zeichen, die Spalte fasst %d", n, uebernahme.MaxAuflage)
			}

			prot.Schliessen()
			log, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
			if err != nil {
				t.Fatalf("Protokoll lesen: %v", err)
			}
			gekuerzt := strings.Contains(string(log), "auflage") && strings.Contains(string(log), "gekürzt")
			if gekuerzt != f.gekuerzt {
				t.Errorf("Kürzung im Protokoll: %v, erwartet %v:\n%s", gekuerzt, f.gekuerzt, log)
			}
		})
	}
}

// Ohne Angabe bleibt die Spalte leer: nil wird NULL, kein leerer Text.
func TestFelder_OhneAuflage(t *testing.T) {
	prot, err := uebernahme.NeuesProtokoll(t.TempDir()+"/protokoll.md", "Littera-ID")
	if err != nil {
		t.Fatalf("NeuesProtokoll: %v", err)
	}
	t.Cleanup(prot.Schliessen)
	l := &bestandslauf{s: &Schreiber{prot: prot}, ab: &Altbestand{}}

	if got := l.felder(Titel{ID: "T1", Haupttitel: "Ein Buch"}).auflage; got != nil {
		t.Errorf("auflage = %q, erwartet nil", *got)
	}
}

func zeige(s *string) string {
	if s == nil {
		return "nil"
	}
	return `"` + *s + `"`
}
