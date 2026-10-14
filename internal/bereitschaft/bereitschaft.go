// Package bereitschaft beantwortet eine Frage: Was ist eingerichtet, aber nicht in Betrieb?
//
// Eine Funktion kann fertig gebaut und verdrahtet sein und trotzdem nichts tun, weil eine
// Einstellung fehlt: Die Auslagerung der Sicherung überspringt sich ohne Zugangsdaten, die
// nächtliche Sicherung ohne Schlüssel, und die Bestellmail geht ohne öffentliche Adresse ohne
// ihren Link hinaus. Kein Fehler und kein Statuscode zeigt das an; die Selbstprüfung nennt es.
//
// Die Prüfungen sind eine reine Funktion über eine Lage: Sie lassen sich ganz testen, ohne
// Umgebungsvariablen zu verbiegen oder eine Datenbank zu brauchen. Die Lage trägt die Tür
// zusammen (api/system/betriebsbereitschaft_handler.go).
package bereitschaft

import (
	"time"

	"bibliothek/jobs"
	"bibliothek/repository"
)

// Stufen eines Befundes. Bewusst nur drei — eine feinere Skala liest niemand.
const (
	StufeOK       = "ok"       // eingerichtet und in Betrieb
	StufeWarnung  = "warnung"  // läuft, aber nicht wie gedacht
	StufeKritisch = "kritisch" // vor dem Echtbetrieb zwingend zu klären
)

// Zwei Formulierungen sagen mehrere Prüfungen wörtlich gleich. Sie stehen hier an
// einer Stelle, damit die Selbstprüfung nicht in vier Tonfällen spricht (go:S1192).
const abhilfeDBNeuLaden = "Datenbankverbindung prüfen und die Seite neu laden."

// befundNichtImEchtbetrieb formuliert den Befund für einen Bereich, den es außerhalb
// des echten Betriebs bewusst nicht gibt — kein Mangel, sondern die Umgebung.
func befundNichtImEchtbetrieb(was, umgebung string) string {
	return was + " — in " + umgebung + " ist das richtig so."
}

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
	Umgebung string

	// Auslagerung der Backups
	S3Endpunkt          string
	S3Zugangsschluessel string
	S3Geheimschluessel  string
	S3Bucket            string

	// Geheimnisse
	GeheimnisSchutzScharf bool
	JWTGeheimnis          string
	Schluessel            string
	// SchluesselProbe: Ließ sich je verschlüsselter Spalte ein Wert mit dem laufenden
	// APP_ENCRYPTION_KEY entschlüsseln? nil: nicht lesbar.
	SchluesselProbe *repository.SchluesselProbe

	// Anmeldung
	IMAPHost string
	// SelbstanmeldeDomain: SELBSTANMELDUNG_DOMAIN — leer heißt, Kollegium muss von Hand
	// angelegt werden (auth/selbstanmeldung.go).
	SelbstanmeldeDomain string

	// Aus den Einstellungen (Datenbank, nicht .env — siehe api/system/mail_settings.go)
	OeffentlicheAdresse string
	SMTPHost            string
	// Die beiden Ausnahmeschalter des Mailversands (mailservice/versand.go), jeweils
	// „steht auf true". Bis zum 21.09.2026 reichte Compose sie nicht durch; seitdem wirken
	// sie im Container, und nur der Klartext-Schalter schrieb je eine Logzeile.
	SMTPUnsicheresTLSErlaubt bool
	SMTPKlartextErlaubt      bool

	// Bestand: was scripts/seed_demo.sql angelegt hat (Schüler DEMO-S-, Exemplare DEMO-B-).
	DemoSchueler  int
	DemoExemplare int

	// Lieferanten mit einer Kennung der alten Startdaten (db/seed.go 30.05.–07.09.2026,
	// Migration 107): Beispiel-Adresse oder Beispiel-Kundennummer. Eine Bestellung an so
	// einen Eintrag geht wirklich raus — an niemanden. nil: nicht lesbar.
	BeispielLieferanten []string

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
	KlassenOhneLehrkraft  []string // Klassen bis Jahrgang 10 mit aktiven Schülern, ohne Zuordnungs-Zeile
	VerwaisteZuordnungen  []string // Zuordnungs-Zeilen ohne aktive Schüler
	VerwaisteBuecherliste []string // class_books-Klassen ohne aktive Schüler

	// Nächtliches Backup (Ausfallmatrix 20.08.2026, B3): Der Job überspringt sich
	// still bei fehlendem Key und steht still, wenn der Cron stirbt. Das sah bisher
	// nur das Dashboard-Badge (api/system/backup_status.go) — die tägliche Kritisch-Alarm-Mail
	// speist sich aus DIESEN Befunden und blieb stumm. nil LetztesBackup: noch nie
	// gelaufen. Jetzt ist injiziert, damit die Altersrechnung testbar bleibt.
	BackupSchluesselGesetzt bool
	BackupSchluesselSchwach bool
	LetztesBackup           *time.Time
	Jetzt                   time.Time

	// Ferientabelle (06.09.2026): Der LMF-Planer belegt den letzten Tag des
	// Büchertauschs aus den Sommerferien Hessen vor (pkg/lmfplan/sommerferien.go, KMK-
	// Beschluss in Sechsjahresblöcken, plus die Jahre aus der Einstellung
	// „Sommerferien"). Läuft die Tabelle aus, steht der Planer ohne Vorgabe da — still.
	// Das letzte bekannte Jahr, damit die Warnung hier steht, wo sie jemand liest, und
	// die Abhilfe eine Einstellung ist, nicht ein Programm-Update.
	FerientabelleBis int

	// Ferien für Leihfristen (24.09.2026): Herbst-, Weihnachts- und Osterferien Hessen
	// stehen als Tabelle im Programm (pkg/lmfplan/uebrige_ferien.go); eine Leihfrist, die in
	// sie fällt, rückt auf den nächsten Schultag. Das Jahr, in dem die letzten dieser
	// Ferien enden — danach rücken die Fristen nur noch über Wochenende, Feiertag und
	// Sommerferien. Anders als bei den Sommerferien gibt es keine Einstellung dafür.
	UebrigeFerienBis int

	// Restore-Probe (Schema-Erweiterung 21.08.2026): Ob das jüngste Backup
	// WIEDERHERSTELLBAR ist, beweist wöchentlich jobs.RunRestoreProbe — ein Backup,
	// das sich nicht einspielen lässt, ist exakt so viel wert wie keins. nil: noch
	// nie gelaufen (oder Ergebnis nicht lesbar).
	RestoreProbe *jobs.RestoreProbeErgebnis

	// DSGVO-Löschroutinen: je eine Zeile pro Routine mit der Anzahl überfälliger, nicht
	// gelöschter Datensätze. nil heißt „nicht erhoben" (Warnung) — bewusst unterschieden
	// von der leeren Liste, die „geprüft, nichts überfällig" bedeutet.
	//
	// Bis zum 23.08.2026 stand hier ein einzelner Zähler für die Anonymisierung: EINE
	// Routine. Die endgültige Abgänger-Löschung, die Befristung der Lesehistorie, die der
	// Anliegen und die Aufbewahrung des Protokolls liefen jede Nacht unbeobachtet — und keine von ihnen
	// hatte zu diesem Zeitpunkt jemals auch nur eine Zeile gelöscht (das Projekt war 86
	// Tage alt, die kürzeste Frist 90 Tage). Der erste echte Löschvorgang stand noch
	// bevor; ein stiller Fehlschlag wäre erst aufgefallen, wenn jemand fragt, warum
	// Daten von vor zwei Jahren noch da sind.
	LoeschRueckstand []repository.LoeschRueckstand

	// Ehemalige mit offenen Vorgängen seit mehr als einem Jahr (Register 05.09.2026,
	// Entscheidung 1): Ein offenes Buch oder eine unbezahlte Forderung schützt den
	// Datensatz vor Anonymisierung und Löschung — richtig, solange jemand mahnt oder
	// abrechnet. Schließt niemand den Vorgang, bleibt Name und Anschrift auf Dauer, und
	// keine Routine meldet es. nil: nicht erhoben.
	EhemaligeMitOffenenVorgaengen *int

	// Schadensersatz-Bescheid (11.09.2026): die Pflichtangaben, die noch fehlen — aus
	// derselben Funktion, mit der die Bescheid-Tür abweist (BescheidAngaben.FehlendeAngaben).
	// Ohne diese Zeile fiel die Lücke erst auf, wenn ein Bescheid gebraucht wurde.
	// nil: nicht erhoben; leere Liste: alles da.
	BescheidFehlend []string

	// Nachbuch-Meldungen (Migration 117), die seit mehr als 14 Tagen niemand quittiert
	// hat: Abweichungen zwischen Offline-Scan und Wirklichkeit, die jemand ansehen muss.
	// nil: nicht erhoben.
	NachbuchMeldungenOffen *int
}

// IstEchterBetrieb sagt, ob die Umgebung der Betrieb ist. In local, development und test
// sind die Anmeldung ohne Mailserver und Beispiel-Geheimnisse richtig und kein Mangel: Ein
// Wächter, der am Entwicklungsrechner dauernd rot ist, wird abgeschaltet statt gelesen.
func IstEchterBetrieb(umgebung string) bool {
	switch umgebung {
	case "local", "development", "test":
		return false
	}
	return true
}

// Pruefe wendet alle Regeln auf eine Lage an. Reine Funktion, keine Seiteneffekte.
func Pruefe(l Lage) []Befund {
	echt := IstEchterBetrieb(l.Umgebung)

	befunde := []Befund{
		pruefeAuslagerung(l),
		pruefeGeheimnisse(l, echt),
		pruefeSchluesselPasstZumBestand(l),
		pruefeAnmeldung(l, echt),
		pruefeSelbstanmeldung(l),
		pruefeBestelllink(l),
		pruefeMailversand(l),
		pruefeDemodaten(l),
		pruefeBeispielLieferanten(l),
		pruefeRechteVorgabe(l),
		pruefeAdminKonten(l),
		pruefeKlassenDrift(l),
		pruefeBackupAlter(l, echt),
		pruefeRestoreProbe(l, echt),
		pruefeLoeschroutinen(l),
		pruefeEhemaligeOffen(l),
		pruefeNachbuchMeldungenOffen(l),
		pruefeFerientabelle(l),
		pruefeUebrigeFerien(l),
		pruefeBescheidAngaben(l),
	}
	return befunde
}

// Gesamt fasst die Befunde zu einer Stufe zusammen: Die schärfste gewinnt. Die Seite und jeder
// andere Leser fragen sie hier, damit die Regel nicht ein zweites Mal in der Oberfläche steht.
func Gesamt(befunde []Befund) string {
	gesamt := StufeOK
	for _, b := range befunde {
		switch b.Stufe {
		case StufeKritisch:
			return StufeKritisch
		case StufeWarnung:
			gesamt = StufeWarnung
		}
	}
	return gesamt
}
