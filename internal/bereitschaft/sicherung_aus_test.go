package bereitschaft

import (
	"strings"
	"testing"
)

// Hat die Schule die eigene Sicherung abgeschaltet, melden die drei Bereiche der Sicherung „in
// Ordnung, abgeschaltet", auch wenn nichts davon eingerichtet ist: Sonst ginge jeden Tag eine
// Alarm-Mail über eine Entscheidung hinaus, und der Hinweis stünde über jeder Seite.
func TestBetriebsbereitschaft_SicherungAusIstKeinBefund(t *testing.T) {
	l := lageEingerichtet()
	l.SicherungAus = true
	l.BackupSchluesselGesetzt = false
	l.LetztesBackup = nil
	l.S3Bucket = ""
	l.RestoreProbe = nil

	befunde := Pruefe(l)
	for _, bereich := range []string{"Nächtliches Backup", "Auslagerung der Backups", "Restore-Probe"} {
		b := befundZu(t, befunde, bereich)
		if b.Stufe != StufeOK || !strings.Contains(b.Befund, "abgeschaltet") {
			t.Errorf("%s: Stufe %q, Befund %q — erwartet ok mit „abgeschaltet“", bereich, b.Stufe, b.Befund)
		}
	}
}
