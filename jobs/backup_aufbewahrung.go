package jobs

import (
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Die Aufbewahrung der Nachtsicherung: die jüngsten 14, dazu 12 wöchentliche Stände. Ein
// Fehler, der unbemerkt Daten verändert (ein falscher Import, ein Löschlauf, ein Skript zur
// Reparatur) und erst nach den sechs Wochen der Sommerferien auffällt, steckte bei 14 Nächten
// allein in jeder vorhandenen Sicherung. Die Folge: Gelöschte Personen stehen bis zu etwa drei
// Monate in den Sicherungen; die Auskunft nennt das (internal/auskunft, dsgvoSicherungen).
//
// Das gilt für die Nachtsicherung im Backup-Verzeichnis des Containers und für ihre Kopie außer
// Haus (backup_auslagerung.go). Die Sicherung vor einem Update (update.sh, vordeploy_…) und die
// von Hand (scripts/backup.sh, bibliothek_backup_…) liegen in ./backups auf dem Host und haben
// eigene Fristen.
const (
	// AufbewahrteNaechte: So viele jüngste Nachtsicherungen bleiben.
	AufbewahrteNaechte = 14
	// AufbewahrteWochen: Von den älteren bleibt je Kalenderwoche die jüngste, für so viele Wochen.
	AufbewahrteWochen = 12
)

// sicherungsStempel ist das Zeitformat im Namen der Nachtsicherung (UTC), geschrieben von
// BackupLauf.FuehreAus und gelesen von sicherungsZeit.
const sicherungsStempel = "2006-01-02T150405Z"

// zuLoeschen liefert die Sicherungen, die die Aufbewahrung nicht hält. dateien ist nach Namen
// sortiert, also zeitlich (leseBackups). Die naechte jüngsten bleiben; von den älteren je
// Kalenderwoche (ISO, Schulzeitzone) die jüngste, für die wochen jüngsten Wochen, in denen es
// eine gibt.
func zuLoeschen(dateien []BackupDatei, naechte, wochen int) []BackupDatei {
	if len(dateien) <= naechte {
		return nil
	}
	aelter := dateien[:len(dateien)-naechte]
	bleibt := make(map[string]bool, wochen)
	gesehen := make(map[[2]int]bool, wochen)
	for i := len(aelter) - 1; i >= 0 && len(gesehen) < wochen; i-- {
		jahr, woche := sicherungsZeit(aelter[i]).In(schulzeit.Zone()).ISOWeek()
		if !gesehen[[2]int{jahr, woche}] {
			gesehen[[2]int{jahr, woche}] = true
			bleibt[aelter[i].Name] = true
		}
	}
	var weg []BackupDatei
	for _, d := range aelter {
		if !bleibt[d.Name] {
			weg = append(weg, d)
		}
	}
	return weg
}

// sicherungsZeit ist der Zeitpunkt aus dem Namen, nicht die Änderungszeit, die ein Kopieren
// oder Zurückspielen verschiebt. Passt der Name nicht zum Stempel, gilt die Änderungszeit.
func sicherungsZeit(d BackupDatei) time.Time {
	stempel := strings.TrimSuffix(strings.TrimPrefix(d.Name, backupPraefix), backupEndung)
	if t, err := time.Parse(sicherungsStempel, stempel); err == nil {
		return t
	}
	return d.GeaendertAm
}
