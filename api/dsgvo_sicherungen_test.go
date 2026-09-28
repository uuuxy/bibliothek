package api

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// Die Auskunft nennt die Fristen der Sicherungen als feste Zahlen; gelten tun die Zahlen in
// drei Quellen außerhalb dieses Pakets. Ändert sich eine davon (OFFEN.md 5.30: 12 wöchentliche
// Stände), wird dieser Test rot, und die Auskunft an die betroffene Person zieht mit.
//
// Blindheit: nur die drei Zahlen an je genau einer Fundstelle. Eine vierte Art von Sicherung
// oder eine Zahl, die ihre Bedeutung wechselt (Stände statt Nächte), sieht er nicht.
func TestDsgvoSicherungen_FolgenDenQuellen(t *testing.T) {
	faelle := []struct {
		pfad       string
		muster     string
		inAuskunft int
	}{
		{"../jobs/backup.go", `rotateBackups\(backupDir, (\d+)\)`, sicherungNaechte},
		{"../update.sh", `(?m)^BACKUP_RETENTION_DAYS=(\d+)$`, sicherungVorUpdateTage},
		{"../scripts/backup.sh", `(?m)^RETENTION_ENC_TAGE=(\d+)$`, sicherungVonHandTage},
	}
	for _, f := range faelle {
		b, err := os.ReadFile(f.pfad)
		if err != nil {
			t.Fatalf("%s lesen: %v", f.pfad, err)
		}
		treffer := regexp.MustCompile(f.muster).FindAllSubmatch(b, -1)
		if len(treffer) != 1 {
			t.Fatalf("%s: %d Treffer für %s, erwartet genau einen — die Quelle hat sich verändert, Test nachziehen",
				f.pfad, len(treffer), f.muster)
		}
		quelle, err := strconv.Atoi(string(treffer[0][1]))
		if err != nil {
			t.Fatalf("%s: %q ist keine Zahl: %v", f.pfad, treffer[0][1], err)
		}
		if quelle != f.inAuskunft {
			t.Errorf("%s hält %d, die Auskunft nennt %d — dsgvoSicherungen in dsgvo_auskunft.go nachziehen",
				f.pfad, quelle, f.inAuskunft)
		}
	}
}
