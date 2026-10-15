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
func BackupStatus(schluesselGesetzt, schluesselSchwach bool, letzte *time.Time, jetzt time.Time) string {
	if !schluesselGesetzt || letzte == nil {
		return "critical"
	}
	alter := jetzt.Sub(*letzte)
	switch {
	case alter > BackupKritischAlter:
		return "critical"
	case alter > BackupWarnAlter:
		return "warning"
	case schluesselSchwach:
		return "warning"
	default:
		return "ok"
	}
}
