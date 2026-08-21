package api

// betriebsbereitschaft.go — beantwortet EINE Frage: Was ist eingerichtet, aber nicht in
// Betrieb?
//
// Diese Anlage hat eine wiederkehrende Fehlerart, und sie ist immer dieselbe: Eine
// Funktion ist fertig programmiert, getestet und verdrahtet — und tut nichts, weil eine
// Einstellung fehlt. Kein Fehler, kein Statuscode, kein Log-Eintrag, der jemandem
// auffiele. Dreimal gefunden, jedes Mal von Hand:
//
//   * uploadBackupToS3 (jobs/backup.go) überspringt sich mit "skipping offsite upload",
//     weil vier Variablen leer sind. Verschlüsselte Backups und Host-Dumps liegen
//     seitdem beide auf derselben Platte.
//   * BACKUP_ENCRYPTION_KEY stand in der .env, kam aber nicht im Container an — der
//     nächtliche Job übersprang sich still. Dafür gibt es seither api/backup_status.go;
//     diese Datei ist dessen Verallgemeinerung.
//   * Der Bestelllink fiel aus, weil `oeffentliche_adresse` nie gesetzt war. Die Mails
//     gingen raus, nur ohne den Link, um dessentwillen es sie gibt.
//
// Die Prüfungen sind bewusst als REINE Funktion über eine Lage gebaut: Sie lassen sich
// vollständig testen, ohne Umgebungsvariablen zu verbiegen oder eine Datenbank zu
// brauchen. Das Zusammentragen der Lage steht daneben in betriebsbereitschaft_handler.go.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"bibliothek/db"
	"bibliothek/jobs"
)

// Stufen eines Befundes. Bewusst nur drei — eine feinere Skala liest niemand.
const (
	StufeOK       = "ok"       // eingerichtet und in Betrieb
	StufeWarnung  = "warnung"  // läuft, aber nicht wie gedacht
	StufeKritisch = "kritisch" // vor dem Echtbetrieb zwingend zu klären
)

// Befund ist eine Zeile der Selbstprüfung.
//
// Vier Felder, weil drei nicht reichen: „Was ist" (Befund) beantwortet nicht, „warum das
// zählt" (Folge) — und ohne „was zu tun ist" (Abhilfe) landet die Meldung auf einem
// Zettel statt in der .env.
type Befund struct {
	Bereich string `json:"bereich"`
	Stufe   string `json:"stufe"`
	Befund  string `json:"befund"`
	Folge   string `json:"folge"`
	Abhilfe string `json:"abhilfe"`
}

// Lage ist der Zustand, über den geurteilt wird — eingesammelt vom Handler.
type Lage struct {
	AppEnv string

	// Auslagerung der Backups
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string

	// Geheimnisse
	EnforceProdSecrets bool
	JWTSecret          string
	AppEncryptionKey   string

	// Anmeldung
	ImapHost string

	// Aus den Einstellungen (Datenbank, nicht .env — siehe api/mail_settings.go)
	OeffentlicheAdresse string
	SmtpHost            string

	// Bestand
	DemoSchueler int

	// Live-Rechte (role_permissions) als Rolle→Recht→erlaubt. nil heisst: nicht
	// lesbar — bewusst unterschieden von der leeren Map, denn „leer" würde jede
	// Vorgabe-Zeile als fehlend melden und die Auskunft in Lärm ertränken.
	RechteLive map[string]map[string]bool

	// Aktive Admin-Konten ("Name (mail)"). nil: nicht lesbar.
	AdminKonten []string

	// Konfigurierte Alarm-Empfaenger (Einstellung alarm_empfaenger, kommasepariert).
	// Leer = die Kritisch-Alarme gehen an alle aktiven Admin-Konten.
	AlarmEmpfaenger string

	// Klassen-Drift (Befund F3): Die Klassennamen verbinden Schüler,
	// Klassenlehrer-Zuordnung und Bücherlisten nur als übereinstimmender Text —
	// reißt der Text, meldet nichts einen Fehler, Mahnlisten laufen still leer.
	// nil heißt jeweils: nicht erhoben (Warnung), leere Liste: alles verbunden.
	KlassenOhneLehrkraft  []string // aktive Schüler-Klassen ohne Zuordnungs-Zeile
	VerwaisteZuordnungen  []string // Zuordnungs-Zeilen ohne aktive Schüler
	VerwaisteBuecherliste []string // class_books-Klassen ohne aktive Schüler

	// Nächtliches Backup (Ausfallmatrix 20.08.2026, B3): Der Job überspringt sich
	// still bei fehlendem Key und steht still, wenn der Cron stirbt. Das sah bisher
	// nur das Dashboard-Badge (backup_status.go) — die tägliche Kritisch-Alarm-Mail
	// speist sich aus DIESEN Befunden und blieb stumm. nil LetztesBackup: noch nie
	// gelaufen. Jetzt ist injiziert, damit die Altersrechnung testbar bleibt.
	BackupKeySet  bool
	BackupKeyWeak bool
	LetztesBackup *time.Time
	Jetzt         time.Time

	// Restore-Probe (Schema-Erweiterung 21.08.2026): Ob das jüngste Backup
	// WIEDERHERSTELLBAR ist, beweist wöchentlich jobs.RunRestoreProbe — ein Backup,
	// das sich nicht einspielen lässt, ist exakt so viel wert wie keins. nil: noch
	// nie gelaufen (oder Ergebnis nicht lesbar).
	RestoreProbe *jobs.RestoreProbeErgebnis
}

// IstBekanntesDefaultGeheimnis meldet, ob ein Wert eines der mitgelieferten
// Beispiel-Geheimnisse ist.
//
// Die Liste stand bis zum 11.08.2026 fest verdrahtet in main.go. Sie hier ein zweites Mal
// aufzuschreiben hätte genau die Fehlerart erzeugt, gegen die diese Datei antritt: zwei
// Listen, die auseinanderlaufen, und eine Selbstprüfung, die „alles gut" meldet, während
// der Server aus demselben Grund den Start verweigert.
func IstBekanntesDefaultGeheimnis(wert string) bool {
	switch wert {
	case "super-secret-default-key-at-least-32-bytes", // Default aus docker-compose.yml (JWT)
		"super-secure-aes-key-32-chars-ok", // Default aus docker-compose.yml (AES)
		"supergeheim_lokal":
		return true
	}
	return false
}

// istEchterBetrieb: local/development/test sind Spielwiesen — dort sind mock-Anmeldung
// und Beispiel-Geheimnisse richtig und dürfen nicht als Mangel gemeldet werden. Ein
// Wächter, der auf dem Entwicklungsrechner dauernd rot ist, wird abgeschaltet statt
// gelesen.
func istEchterBetrieb(appEnv string) bool {
	switch appEnv {
	case "local", "development", "test":
		return false
	}
	return true
}

// Pruefe wendet alle Regeln auf eine Lage an. Reine Funktion, keine Seiteneffekte.
func Pruefe(l Lage) []Befund {
	echt := istEchterBetrieb(l.AppEnv)

	befunde := []Befund{
		pruefeAuslagerung(l),
		pruefeGeheimnisse(l, echt),
		pruefeAnmeldung(l, echt),
		pruefeBestelllink(l),
		pruefeMailversand(l),
		pruefeDemodaten(l),
		pruefeRechteVorgabe(l),
		pruefeAdminKonten(l),
		pruefeKlassenDrift(l),
		pruefeBackupAlter(l, echt),
		pruefeRestoreProbe(l, echt),
	}
	return befunde
}

// pruefeRestoreProbe beurteilt das Ergebnis der wöchentlichen Wiederherstellungs-Probe
// (jobs/restore_probe.go). Ein Backup, das sich nicht einspielen lässt, ist so viel
// wert wie keins — und man erfährt es sonst im schlechtesten Moment.
func pruefeRestoreProbe(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Restore-Probe"}
	switch {
	case !echt:
		b.Stufe = StufeOK
		b.Befund = "Keine Restore-Probe — in " + l.AppEnv + " ist das richtig so."
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
		b.Abhilfe = "Ursache im Befundtext beheben (Schlüssel? Datei? psql?), dann per Neustart oder " +
			"nächsten Sonntag erneut proben."
	case l.Jetzt.Sub(l.RestoreProbe.Zeitpunkt) > 9*24*time.Hour:
		b.Stufe = StufeKritisch
		b.Befund = fmt.Sprintf("Die letzte erfolgreiche Probe ist %d Tage her — der Wochenlauf steht still.",
			int(l.Jetzt.Sub(l.RestoreProbe.Zeitpunkt).Hours()/24))
		b.Folge = "Ob die aktuellen Backups wiederherstellbar sind, ist wieder unbewiesen."
		b.Abhilfe = "Container-Logs nach 'Restore-Probe' durchsuchen; läuft der Scheduler?"
	default:
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Backup %s erfolgreich wiederhergestellt (%d Tabellen, Probe vom %s).",
			l.RestoreProbe.BackupDatei, l.RestoreProbe.Tabellen, l.RestoreProbe.Zeitpunkt.Format("02.01.2006"))
	}
	return b
}

// pruefeBackupAlter hebt den Backup-Wächter des Dashboard-Badges in die Befunde —
// und damit in die tägliche Kritisch-Alarm-Mail. Die Schwellen und ihre Anwendung
// (computeBackupStatus, backup_status.go) bleiben die EINE Quelle; hier wird nur
// übersetzt, damit Badge und Alarm nie auseinanderlaufen.
func pruefeBackupAlter(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Nächtliches Backup"}
	if !echt {
		b.Stufe = StufeOK
		b.Befund = "Kein Backup-Betrieb — in " + l.AppEnv + " ist das richtig so."
		return b
	}
	switch computeBackupStatus(l.BackupKeySet, l.BackupKeyWeak, l.LetztesBackup, l.Jetzt) {
	case "critical":
		b.Stufe = StufeKritisch
		switch {
		case !l.BackupKeySet:
			b.Befund = "BACKUP_ENCRYPTION_KEY fehlt — der nächtliche Job überspringt sich still."
			b.Folge = "Es entsteht KEIN Backup. Ein Datenbankschaden kostet den gesamten Bestand."
			b.Abhilfe = "BACKUP_ENCRYPTION_KEY in der .env setzen (≥32 Zeichen) und den Schlüssel getrennt verwahren."
		case l.LetztesBackup == nil:
			b.Befund = "Noch nie ein Backup geschrieben."
			b.Folge = "Ein Datenbankschaden kostet den gesamten Bestand."
			b.Abhilfe = "Backup-Job und BACKUP_DIR prüfen; der Job läuft täglich 02:30."
		default:
			b.Befund = fmt.Sprintf("Letztes Backup vor %d Stunden — mehr als %d Stunden alt.",
				int(l.Jetzt.Sub(*l.LetztesBackup).Hours()), int(backupCriticalAge.Hours()))
			b.Folge = "Der Datenverlust-Puffer ist aufgebraucht; der Job steht offenbar still."
			b.Abhilfe = "Container-Logs des Backup-Jobs prüfen (läuft täglich 02:30)."
		}
	case "warning":
		b.Stufe = StufeWarnung
		if l.LetztesBackup != nil && l.Jetzt.Sub(*l.LetztesBackup) > backupWarnAge {
			b.Befund = fmt.Sprintf("Letztes Backup vor %d Stunden — mindestens ein Lauf wurde verpasst.",
				int(l.Jetzt.Sub(*l.LetztesBackup).Hours()))
			b.Folge = "Bleibt es dabei, ist der Datenverlust-Puffer in Kürze aufgebraucht."
			b.Abhilfe = "Container-Logs des Backup-Jobs prüfen (läuft täglich 02:30)."
		} else {
			b.Befund = "BACKUP_ENCRYPTION_KEY ist zu kurz für die SHA-256-Ableitung."
			b.Folge = "Die Backups sind schwächer verschlüsselt als vorgesehen."
			b.Abhilfe = "Einen Schlüssel mit mindestens 32 Zeichen setzen."
		}
	default:
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Letztes Backup vor %d Stunden.", int(l.Jetzt.Sub(*l.LetztesBackup).Hours()))
	}
	return b
}

// pruefeAdminKonten macht sichtbar, WER Vollzugriff hat und die Alarm-Mails erhält.
//
// Anlass (16.08.2026 abends): Die erste Kritisch-Mail ging an ein Admin-Konto, von
// dem der Betreiber nichts wusste — auf Prod lagen VIER aktive Admins, drei davon
// ohne bekannte Herkunft (die Konten-Anlage war bis dahin unauditiert). Eine Liste
// auf dieser Seite hätte das Wochen früher gezeigt. Bewusst Stufe OK, solange
// überhaupt Admins existieren: Mehrere Admins können völlig richtig sein — die
// Seite URTEILT nicht, sie zeigt. Wer hier einen unbekannten Namen liest, handelt.
func pruefeAdminKonten(l Lage) Befund {
	b := Befund{Bereich: "Admin-Konten"}
	switch {
	case l.AdminKonten == nil:
		b.Stufe = StufeWarnung
		b.Befund = "Die Admin-Konten konnten nicht gelesen werden."
		b.Folge = "Unbekannt, wer Vollzugriff hat und die Alarm-Mails erhält."
		b.Abhilfe = "Datenbankverbindung prüfen und die Seite neu laden."
	case len(l.AdminKonten) == 0:
		b.Stufe = StufeKritisch
		b.Befund = "Kein aktives Admin-Konto vorhanden."
		b.Folge = "Niemand kann das System verwalten, und die Kritisch-Alarme erreichen niemanden."
		b.Abhilfe = "Über die Datenbank bzw. INITIAL_ADMIN_EMAIL ein Admin-Konto herstellen."
	default:
		b.Stufe = StufeOK
		ziel := "Die Kritisch-Alarme gehen an alle diese Konten."
		if strings.TrimSpace(l.AlarmEmpfaenger) != "" {
			ziel = "Die Kritisch-Alarme gehen an die konfigurierten Empfänger: " + l.AlarmEmpfaenger + "."
		}
		b.Befund = fmt.Sprintf("%d Admin-Konto/-Konten mit Vollzugriff: %s. %s",
			len(l.AdminKonten), strings.Join(l.AdminKonten, "; "), ziel)
	}
	return b
}

// pruefeRechteVorgabe vergleicht die Live-Rechte mit der Code-Vorgabe (db.RechteVorgabe).
//
// Der Seed schreibt nur FEHLENDE Zeilen (ON CONFLICT DO NOTHING) — ändert sich die
// Vorgabe im Code, erreicht sie eine bestehende Anlage nie von selbst. Genau so sah
// das Kollegium wochenlang nur einen Teil seiner Menüpunkte, und niemand merkte es.
//
// Eine Abweichung ist bewusst nur eine WARNUNG, kein Fehler: role_permissions ist
// über die Oberfläche einstellbar, die Abweichung kann eine Admin-Entscheidung sein.
// Die Prüfung urteilt nicht darüber — sie macht die Abweichung sichtbar, damit ein
// Mensch entscheidet. Repariert wird hier nichts.
func pruefeRechteVorgabe(l Lage) Befund {
	b := Befund{Bereich: "Rechte-Vorgabe"}
	if l.RechteLive == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Die Live-Rechte (role_permissions) konnten nicht gelesen werden."
		b.Folge = "Ob Menü und API der aktuellen Vorgabe folgen, ist unbekannt."
		b.Abhilfe = "Datenbankverbindung prüfen und die Seite neu laden."
		return b
	}

	var abweichungen []string
	vorgabe := map[string]map[string]bool{}
	for _, v := range db.RechteVorgabe {
		if vorgabe[v.Role] == nil {
			vorgabe[v.Role] = map[string]bool{}
		}
		vorgabe[v.Role][v.Permission] = v.Allowed
		zeile, bekannt := l.RechteLive[v.Role]
		liveWert, vorhanden := false, false
		if bekannt {
			liveWert, vorhanden = zeile[v.Permission]
		}
		switch {
		case !vorhanden:
			abweichungen = append(abweichungen,
				fmt.Sprintf("%s/%s fehlt live (Vorgabe: %s)", v.Role, v.Permission, anAus(v.Allowed)))
		// Optionale Paare (db.RechteOptional): Der Haken in der Rechte-Matrix ist
		// dort der vorgesehene Gebrauch, kein Drift — nur die Existenz zählt.
		case db.RechteOptional[v.Role+"/"+v.Permission]:
		case liveWert != v.Allowed:
			abweichungen = append(abweichungen,
				fmt.Sprintf("%s/%s live %s, Vorgabe %s", v.Role, v.Permission, anAus(liveWert), anAus(v.Allowed)))
		}
	}
	// Gegenrichtung: Live-Zeilen, deren Rolle oder Recht die Vorgabe nicht kennt —
	// Reste alter Vokabulare (z. B. eine umbenannte Rolle) oder Tippfehler im Editor.
	for rolle, zeile := range l.RechteLive {
		for recht, erlaubt := range zeile {
			if _, kennt := vorgabe[rolle][recht]; !kennt {
				abweichungen = append(abweichungen,
					fmt.Sprintf("%s/%s live %s, in der Vorgabe unbekannt", rolle, recht, anAus(erlaubt)))
			}
		}
	}

	if len(abweichungen) == 0 {
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Live-Rechte decken sich mit der Code-Vorgabe (%d Zeilen).", len(db.RechteVorgabe))
		return b
	}

	// Map-Reihenfolge ist Zufall — sortieren, damit Anzeige und Tests stabil sind.
	sort.Strings(abweichungen)
	const zeigeMax = 6
	gezeigt := abweichungen
	if len(gezeigt) > zeigeMax {
		gezeigt = append(append([]string{}, gezeigt[:zeigeMax]...),
			fmt.Sprintf("… und %d weitere", len(abweichungen)-zeigeMax))
	}
	b.Stufe = StufeWarnung
	b.Befund = fmt.Sprintf("%d Abweichung(en) von der Code-Vorgabe: %s.",
		len(abweichungen), strings.Join(gezeigt, "; "))
	b.Folge = "Menü und API richten sich nach der Live-Tabelle, nicht nach dem Code. " +
		"Nach einer Code-Änderung bleibt die alte Einstellung bestehen — eine Rolle sieht " +
		"dann mehr oder weniger, als die aktuelle Vorgabe vorsieht, ohne dass es auffällt."
	b.Abhilfe = "Jede Zeile prüfen: bewusste Admin-Entscheidung → so lassen (die Warnung " +
		"dokumentiert sie), Drift nach Code-Änderung → System → Berechtigungen angleichen."
	return b
}

func anAus(erlaubt bool) string {
	if erlaubt {
		return "an"
	}
	return "aus"
}

func pruefeAuslagerung(l Lage) Befund {
	b := Befund{Bereich: "Auslagerung der Backups"}
	if l.S3Endpoint != "" && l.S3AccessKey != "" && l.S3SecretKey != "" && l.S3Bucket != "" {
		b.Stufe = StufeOK
		b.Befund = "Backups werden zusätzlich außer Haus abgelegt (" + l.S3Bucket + ")."
		return b
	}
	b.Stufe = StufeKritisch
	b.Befund = "Kein Ziel außer Haus eingerichtet — der Job überspringt sich still."
	b.Folge = "Alle Backups liegen auf derselben Platte wie die Datenbank. " +
		"Ein Plattenausfall kostet den gesamten Bestand samt aller Sicherungen."
	b.Abhilfe = "S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY und S3_BUCKET in der .env setzen."
	return b
}

func pruefeGeheimnisse(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Geheimnisse"}
	defaultsAktiv := IstBekanntesDefaultGeheimnis(l.JWTSecret) ||
		IstBekanntesDefaultGeheimnis(l.AppEncryptionKey)

	switch {
	case !defaultsAktiv && l.EnforceProdSecrets:
		b.Stufe = StufeOK
		b.Befund = "Eigene Geheimnisse gesetzt, die harte Absicherung ist scharf."
	case !defaultsAktiv:
		b.Stufe = StufeWarnung
		b.Befund = "Eigene Geheimnisse gesetzt, ENFORCE_PROD_SECRETS steht aber auf false."
		b.Folge = "Der Server würde auch mit einem Beispiel-Geheimnis starten — " +
			"ein versehentliches Zurückfallen fiele nicht auf."
		b.Abhilfe = "ENFORCE_PROD_SECRETS=true setzen."
	case echt:
		b.Stufe = StufeKritisch
		b.Befund = "Es laufen mitgelieferte Beispiel-Geheimnisse."
		b.Folge = "JWT_SECRET und APP_ENCRYPTION_KEY stehen öffentlich im Repository. " +
			"Wer sie kennt, kann Sitzungen fälschen und verschlüsselte Ablagen lesen."
		b.Abhilfe = "Eigene Werte setzen (JWT_SECRET ≥32 Zeichen, APP_ENCRYPTION_KEY genau 32 Byte), " +
			"danach ENFORCE_PROD_SECRETS=true."
	default:
		b.Stufe = StufeOK
		b.Befund = "Beispiel-Geheimnisse — in " + l.AppEnv + " ist das richtig so."
	}
	return b
}

func pruefeAnmeldung(l Lage, echt bool) Befund {
	b := Befund{Bereich: "Anmeldung"}
	switch {
	case l.ImapHost == "mock" && echt:
		b.Stufe = StufeKritisch
		b.Befund = "IMAP_HOST steht auf \"mock\"."
		b.Folge = "Die Anmeldung akzeptiert JEDES Passwort für jede hinterlegte E-Mail-Adresse."
		b.Abhilfe = "Den echten IMAP-Host der Schule eintragen."
	case l.ImapHost == "mock":
		b.Stufe = StufeOK
		b.Befund = "Mock-Anmeldung — in " + l.AppEnv + " ist das richtig so."
	case l.ImapHost == "":
		b.Stufe = StufeKritisch
		b.Befund = "IMAP_HOST ist leer."
		b.Folge = "Niemand kann sich anmelden."
		b.Abhilfe = "Den IMAP-Host der Schule eintragen."
	default:
		b.Stufe = StufeOK
		b.Befund = "Anmeldung läuft über " + l.ImapHost + "."
	}
	return b
}

func pruefeBestelllink(l Lage) Befund {
	b := Befund{Bereich: "Bestell-Bestätigungslink"}
	if l.OeffentlicheAdresse != "" {
		b.Stufe = StufeOK
		b.Befund = "Links verweisen auf " + l.OeffentlicheAdresse + "."
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = "Keine öffentliche Adresse hinterlegt."
	b.Folge = "Bestellmails gehen raus, aber ohne den Bestätigungslink, " +
		"um dessentwillen es sie gibt. Der Lieferant kann nicht bestätigen."
	b.Abhilfe = "Einstellungen → öffentliche Adresse der Anwendung eintragen."
	return b
}

func pruefeMailversand(l Lage) Befund {
	b := Befund{Bereich: "Mailversand (Mahnwesen)"}
	if l.SmtpHost != "" {
		b.Stufe = StufeOK
		b.Befund = "Versand über " + l.SmtpHost + "."
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = "Kein SMTP-Server in den Einstellungen."
	b.Folge = "Mahnungen und Bestellmails können nicht zugestellt werden."
	b.Abhilfe = "Einstellungen → Mail-Konfiguration ausfüllen und mit dem Testversand prüfen."
	return b
}

func pruefeDemodaten(l Lage) Befund {
	b := Befund{Bereich: "Demo-Daten"}
	if l.DemoSchueler == 0 {
		b.Stufe = StufeOK
		b.Befund = "Keine Demo-Datensätze im Bestand."
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = strconv.Itoa(l.DemoSchueler) + " Demo-Schüler im Bestand."
	b.Folge = "Statistik, Mahnwesen und Bestellbedarf mischen echte Zahlen mit Fiktion. " +
		"Demo-Eltern-Adressen enden auf example.invalid, ein Mahnlauf erreicht sie nie."
	b.Abhilfe = "Vor dem Echtstart den DEMO-Block aus scripts/seed_demo.sql (Abschnitt 1) ausführen."
	return b
}

// pruefeKlassenDrift macht die Text-Verbindungen des Klassennamens sichtbar
// (Befund F3, bewertung/datenbank-pruefbericht.md): Schüler, Klassenlehrer-
// Zuordnung und Bücherlisten hängen nur an übereinstimmenden Namen. Der
// Jahreswechsel versetzt die Zuordnung seit 18.08.2026 mit (student_promotion.go),
// aber Umbenennungen von Hand oder LUSD-Umschichtungen können weiter driften —
// und ohne diese Prüfung merkt es niemand, bis eine Mahnliste leer läuft.
// Bewusst nur Warnung: Eine Klasse ohne Lehrkraft-Zuordnung kann gewollt sein
// (die Mahnmail geht dann eben an niemanden — genau das soll man hier sehen).
func pruefeKlassenDrift(l Lage) Befund {
	b := Befund{Bereich: "Klassen-Zuordnung"}
	if l.KlassenOhneLehrkraft == nil || l.VerwaisteZuordnungen == nil || l.VerwaisteBuecherliste == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Die Klassen-Verbindungen konnten nicht geprüft werden (Datenbank nicht erreichbar)."
		b.Folge = "Ob Mahnwesen und Bücherlisten ihre Klassen finden, ist unbekannt."
		b.Abhilfe = "Datenbankverbindung prüfen und die Seite neu laden."
		return b
	}

	var probleme []string
	if len(l.KlassenOhneLehrkraft) > 0 {
		probleme = append(probleme, "Klassen ohne Lehrkraft-Zuordnung: "+strings.Join(l.KlassenOhneLehrkraft, ", "))
	}
	if len(l.VerwaisteZuordnungen) > 0 {
		probleme = append(probleme, "Zuordnungen ohne aktive Schüler: "+strings.Join(l.VerwaisteZuordnungen, ", "))
	}
	if len(l.VerwaisteBuecherliste) > 0 {
		probleme = append(probleme, "Bücherlisten für unbekannte Klassen: "+strings.Join(l.VerwaisteBuecherliste, ", "))
	}
	if len(probleme) == 0 {
		b.Stufe = StufeOK
		b.Befund = "Alle Klassennamen sind zwischen Schülern, Lehrkraft-Zuordnung und Bücherlisten verbunden."
		return b
	}
	b.Stufe = StufeWarnung
	b.Befund = strings.Join(probleme, " — ")
	b.Folge = "Mahnlisten dieser Klassen erreichen keine Lehrkraft bzw. Listen zeigen ins Leere — ohne Fehlermeldung."
	b.Abhilfe = "Unter Mahnwesen → Klassenlehrer die Zuordnung nachziehen oder verwaiste Einträge entfernen."
	return b
}
