package jobs

import (
	"context"
	"log/slog"
	"os"
	"time"

	"bibliothek/repository"
)

// Aufbewahrungsdauer ist das höchste Alter einer Nachtsicherung im laufenden Betrieb: die Nächte
// und die Wochen der Aufbewahrung hintereinander (backup_aufbewahrung.go).
const Aufbewahrungsdauer = (AufbewahrteNaechte + 7*AufbewahrteWochen) * 24 * time.Hour

// backupVerzeichnis ist BACKUP_DIR, ohne Angabe ./backups.
func backupVerzeichnis() string {
	if verzeichnis := os.Getenv("BACKUP_DIR"); verzeichnis != "" {
		return verzeichnis
	}
	return "./backups"
}

// fuehreNachtsicherungAus ist der Lauf um zeitSicherung. Ist die eigene Sicherung abgeschaltet,
// entsteht keine neue, und es laufen nur die alten Stände aus.
func (z *Zeitplan) fuehreNachtsicherungAus(ctx context.Context, backup *BackupLauf) {
	if z.sicherungAbgeschaltet(ctx) {
		slog.Info("Sicherung abgeschaltet (Einstellungen → Sicherung): keine Nachtsicherung, ältere Stände laufen aus")
		raeumeAbgelaufeneSicherungenAuf(ctx, time.Now())
		return
	}
	slog.Info("Scheduler Backup: starting scheduled daily database backup...")
	backup.FuehreAus(ctx)
}

// sicherungAbgeschaltet liest den Schalter der eigenen Sicherung (Vorgabe aus). Ist die
// Einstellung nicht lesbar, wird gesichert: Eine Sicherung zu viel schadet nicht, eine fehlende,
// die eingeschaltet war, schon.
func (z *Zeitplan) sicherungAbgeschaltet(ctx context.Context) bool {
	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		slog.Error("Sicherung: Einstellung nicht lesbar, es wird gesichert", "fehler", err)
		return false
	}
	return !einst.SicherungAn
}

// raeumeAbgelaufeneSicherungenAuf löscht die Sicherungen, die älter sind als die
// Aufbewahrungsdauer, im Verzeichnis und außer Haus. Ohne neue Sicherung löscht die Aufbewahrung
// nichts mehr; die alten Stände sollen aber nicht länger liegen als im Betrieb, denn die
// Datenschutz-Auskunft nennt etwa drei Monate (internal/auskunft).
func raeumeAbgelaufeneSicherungenAuf(ctx context.Context, jetzt time.Time) {
	waehle := func(dateien []BackupDatei) []BackupDatei { return abgelaufen(dateien, jetzt) }
	loescheBackups(backupVerzeichnis(), waehle)
	if ziel, ok := leseS3Ziel(); ok {
		loescheBackupsInS3(ctx, ziel, waehle)
	}
}

// abgelaufen liefert die Sicherungen, die älter sind als die Aufbewahrungsdauer.
func abgelaufen(dateien []BackupDatei, jetzt time.Time) []BackupDatei {
	grenze := jetzt.Add(-Aufbewahrungsdauer)
	var weg []BackupDatei
	for _, d := range dateien {
		if d.SicherungsZeit().Before(grenze) {
			weg = append(weg, d)
		}
	}
	return weg
}
