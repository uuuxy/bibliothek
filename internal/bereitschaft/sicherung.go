package bereitschaft

import (
	"fmt"
	"time"

	"bibliothek/jobs"
	"bibliothek/pkg/schulzeit"
)

func pruefeAuslagerung(l Lage) Befund {
	b := Befund{Bereich: "Auslagerung der Backups"}
	if l.S3Endpunkt != "" && l.S3Zugangsschluessel != "" && l.S3Geheimschluessel != "" && l.S3Bucket != "" {
		b.Stufe = StufeOK
		b.Befund = "Backups werden zusätzlich außer Haus abgelegt (" + l.S3Bucket + ")."
		return b
	}
	b.Stufe = StufeKritisch
	b.Befund = "Kein Ziel außer Haus eingerichtet — der Job überspringt sich still."
	b.Folge = "Alle Backups liegen auf derselben Platte wie die Datenbank. " +
		"Ein Plattenausfall kostet den gesamten Bestand samt aller Sicherungen."
	b.Abhilfe = "S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY und S3_BUCKET in der .env setzen und den Stack " +
		"neu starten (docker compose reicht sie seit 22.08.2026 durch; davor kamen sie nie im Container an)."
	return b
}

// pruefeRestoreProbe beurteilt das Ergebnis der wöchentlichen Probe (jobs/restore_probe.go).
// Ein Backup, das sich nicht einspielen lässt, ist so viel wert wie keins, und ohne die
// Probe erfährt man es im schlechtesten Moment.
func pruefeRestoreProbe(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Restore-Probe"}
	switch {
	case !echt:
		b.Stufe = StufeOK
		b.Befund = befundNichtImEchtbetrieb("Keine Restore-Probe", l.Umgebung)
	case l.RestoreProbe == nil:
		b.Stufe = StufeWarnung
		b.Befund = "Noch kein Probelauf verzeichnet."
		b.Folge = "Ob die nächtlichen Backups wiederherstellbar sind, ist unbewiesen."
		b.Abhilfe = "Die Probe läuft sonntags 03:30 — nach dem ersten Lauf verschwindet diese Meldung. " +
			"Bleibt sie über eine Woche stehen, Container-Logs nach 'Restore-Probe' durchsuchen."
	case !l.RestoreProbe.Erfolg:
		b.Stufe = StufeKritisch
		b.Befund = "Die letzte Wiederherstellungs-Probe ist FEHLGESCHLAGEN: " + l.RestoreProbe.Fehler
		b.Folge = "Im Ernstfall ließe sich die Datenbank aus dem jüngsten Backup nicht wiederherstellen."
		b.Abhilfe = "Ursache im Befundtext beheben (Schlüssel? Datei? psql?). Nach einem Fehlschlag " +
			"probt der Stack beim nächsten Neustart von selbst erneut, sonst am Sonntag 03:30 UTC."
	case l.Jetzt.Sub(l.RestoreProbe.Zeitpunkt) > 9*24*time.Hour:
		b.Stufe = StufeKritisch
		b.Befund = fmt.Sprintf("Die letzte erfolgreiche Probe ist %d Tage her — der Wochenlauf steht still.",
			int(l.Jetzt.Sub(l.RestoreProbe.Zeitpunkt).Hours()/24))
		b.Folge = "Ob die aktuellen Backups wiederherstellbar sind, ist wieder unbewiesen."
		b.Abhilfe = "Container-Logs nach 'Restore-Probe' durchsuchen; läuft der Scheduler?"
	default:
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Backup %s erfolgreich wiederhergestellt (%d Tabellen, Probe vom %s).",
			l.RestoreProbe.BackupDatei, l.RestoreProbe.Tabellen, l.RestoreProbe.Zeitpunkt.In(schulzeit.Zone()).Format("02.01.2006"))
	}
	return b
}

// pruefeBackupAlter bringt den Zustand des Backups in die Befunde und damit in die tägliche
// Alarm-Mail. Die Schwellen stehen nur in jobs.BackupStatus, damit das Abzeichen im Rahmen
// der Anwendung und der Alarm nicht auseinanderlaufen.
func pruefeBackupAlter(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Nächtliches Backup"}
	if !echt {
		b.Stufe = StufeOK
		b.Befund = befundNichtImEchtbetrieb("Kein Backup-Betrieb", l.Umgebung)
		return b
	}
	switch jobs.BackupStatus(l.BackupSchluesselGesetzt, l.BackupSchluesselSchwach, l.LetztesBackup, l.Jetzt) {
	case "critical":
		b.Stufe = StufeKritisch
		switch {
		case !l.BackupSchluesselGesetzt:
			b.Befund = "BACKUP_ENCRYPTION_KEY fehlt — der nächtliche Job überspringt sich still."
			b.Folge = "Es entsteht KEIN Backup. Ein Datenbankschaden kostet den gesamten Bestand."
			b.Abhilfe = "BACKUP_ENCRYPTION_KEY in der .env setzen (≥32 Zeichen) und den Schlüssel getrennt verwahren."
		case l.LetztesBackup == nil:
			b.Befund = "Noch nie ein Backup geschrieben."
			b.Folge = "Ein Datenbankschaden kostet den gesamten Bestand."
			b.Abhilfe = "Backup-Job und BACKUP_DIR prüfen; der Job läuft täglich 02:30."
		default:
			b.Befund = fmt.Sprintf("Letztes Backup vor %d Stunden — mehr als %d Stunden alt.",
				int(l.Jetzt.Sub(*l.LetztesBackup).Hours()), int(jobs.BackupKritischAlter.Hours()))
			b.Folge = "Der Datenverlust-Puffer ist aufgebraucht; der Job steht offenbar still."
			b.Abhilfe = "Container-Logs des Backup-Jobs prüfen (läuft täglich 02:30)."
		}
	case "warning":
		b.Stufe = StufeWarnung
		if l.LetztesBackup != nil && l.Jetzt.Sub(*l.LetztesBackup) > jobs.BackupWarnAlter {
			b.Befund = fmt.Sprintf("Letztes Backup vor %d Stunden — mindestens ein Lauf wurde verpasst.",
				int(l.Jetzt.Sub(*l.LetztesBackup).Hours()))
			b.Folge = "Bleibt es dabei, ist der Datenverlust-Puffer in Kürze aufgebraucht."
			b.Abhilfe = "Container-Logs des Backup-Jobs prüfen (läuft täglich 02:30)."
		} else {
			b.Befund = "BACKUP_ENCRYPTION_KEY ist zu kurz (unter 32 Zeichen)."
			b.Folge = "Auch die scrypt-Ableitung rettet eine kurze Passphrase nicht — eine entwendete Backup-Datei wäre offline durchprobierbar."
			b.Abhilfe = "Einen Schlüssel mit mindestens 32 Zeichen setzen."
		}
	default:
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Letztes Backup vor %d Stunden.", int(l.Jetzt.Sub(*l.LetztesBackup).Hours()))
	}
	return b
}
