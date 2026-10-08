package jobs

import "time"

const (
	// BackupWarnAlter ist das Alter, ab dem ein Lauf fehlt: Die Sicherung läuft täglich um
	// 02:30 UTC, nach 26 Stunden ist mindestens einer ausgefallen.
	BackupWarnAlter = 26 * time.Hour
	// BackupKritischAlter ist das Alter, ab dem der Stand zweier Tage fehlt.
	BackupKritischAlter = 48 * time.Hour
)

// BackupStatus ordnet den Stand der Sicherung ein: "ok", "warning" oder "critical". Das
// Abzeichen der Verwaltung und die Selbstprüfung fragen hier, damit beide dieselben Schwellen
// anlegen. Ein schwacher Schlüssel stuft auf "warning", überschreibt aber kein "critical":
// Eine fehlende Sicherung wiegt schwerer als eine schwach verschlüsselte.
func BackupStatus(keySet, keyWeak bool, last *time.Time, now time.Time) string {
	if !keySet || last == nil {
		return "critical"
	}
	age := now.Sub(*last)
	switch {
	case age > BackupKritischAlter:
		return "critical"
	case age > BackupWarnAlter:
		return "warning"
	case keyWeak:
		return "warning"
	default:
		return "ok"
	}
}
