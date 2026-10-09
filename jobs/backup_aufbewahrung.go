package jobs

import (
	"strings"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Die Aufbewahrung der Nachtsicherung, entschieden am 28.09.2026: die jüngsten 14, dazu 12
// wöchentliche Stände. Ein Fehler, der still Daten verändert — ein falscher Import, ein
// Löschlauf, ein Reparaturskript — und erst nach den sechs Wochen der Sommerferien auffällt,
// steckte mit 14 Nächten allein in jeder vorhandenen Sicherung. Folge: Gelöschte Personen stehen
// bis zu etwa drei Monate in den Sicherungen; die Auskunft nennt es (internal/auskunft, dsgvoSicherungen).
//
// Nur die Nachtsicherung im Backup-Verzeichnis des Containers. Die Sicherung vor einem Update
// (update.sh, vordeploy_…) und die von Hand (scripts/backup.sh, bibliothek_backup_…) liegen in
// ./backups auf dem Host und haben eigene Fristen; die Kopie außer Haus hat noch keine Regel
// (docs/OFFEN.md 7.3).
const (
	// BehalteNaechte: so viele jüngste Nachtsicherungen bleiben.
	BehalteNaechte = 14
	// BehalteWochen: Von den älteren bleibt je Kalenderwoche die jüngste, für so viele Wochen.
	BehalteWochen = 12
)

// sicherungsStempel ist das Zeitformat im Namen der Nachtsicherung (UTC), geschrieben von
// RunDatabaseBackup und gelesen von sicherungsZeit.
const sicherungsStempel = "2006-01-02T150405Z"

// zuLoeschen liefert die Sicherungen, die die Aufbewahrung nicht hält. dateien ist nach Namen
// sortiert, also zeitlich (listeBackups). Die naechte jüngsten bleiben; von den älteren je
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

// sicherungsZeit ist der Zeitpunkt aus dem Namen — nicht die Änderungszeit, die ein Kopieren
// oder Zurückspielen verschiebt. Passt der Name nicht zum Stempel, gilt die Änderungszeit.
func sicherungsZeit(d BackupDatei) time.Time {
	stempel := strings.TrimSuffix(strings.TrimPrefix(d.Name, backupPraefix), backupEndung)
	if t, err := time.Parse(sicherungsStempel, stempel); err == nil {
		return t
	}
	return d.GeaendertAm
}
