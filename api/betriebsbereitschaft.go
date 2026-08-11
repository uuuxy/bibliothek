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

import "strconv"

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
	}
	return befunde
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
