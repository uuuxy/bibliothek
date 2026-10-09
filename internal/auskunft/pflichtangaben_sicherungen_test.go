package auskunft

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// Die Auskunft nennt die Fristen der Sicherungen vor einem Update und von Hand als feste
// Zahlen; gelten tun sie in zwei Shell-Skripten. Ändert sich eine davon, wird dieser Test rot,
// und die Auskunft an die betroffene Person zieht mit. Die Nachtsicherung liest die Auskunft
// seit dem 29.09.2026 aus dem Job selbst (jobs.BehalteNaechte, jobs.BehalteWochen); bis dahin
// stand sie hier als dritte Zeile, per Muster aus jobs/backup.go gelesen.
//
// Blindheit: nur die zwei Zahlen an je genau einer Fundstelle. Eine weitere Art von Sicherung
// oder eine Zahl, die ihre Bedeutung wechselt, sieht er nicht.
func TestDsgvoSicherungen_FolgenDenQuellen(t *testing.T) {
	faelle := []struct {
		pfad       string
		muster     string
		inAuskunft int
	}{
		{"../../update.sh", `(?m)^BACKUP_RETENTION_DAYS=(\d+)$`, sicherungVorUpdateTage},
		{"../../scripts/backup.sh", `(?m)^RETENTION_ENC_TAGE=(\d+)$`, sicherungVonHandTage},
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
			t.Errorf("%s hält %d, die Auskunft nennt %d — dsgvoSicherungen in pflichtangaben.go nachziehen",
				f.pfad, quelle, f.inAuskunft)
		}
	}
}
