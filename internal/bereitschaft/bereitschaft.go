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

// Stufen eines Befundes. Es sind drei, weil eine feinere Skala niemand liest.
const (
	StufeOK       = "ok"       // eingerichtet und in Betrieb
	StufeWarnung  = "warnung"  // läuft, aber nicht wie gedacht
	StufeKritisch = "kritisch" // vor dem Echtbetrieb zu klären
)

// Mehrere Prüfungen nennen diese Abhilfe wörtlich gleich; sie steht einmal, damit die
// Selbstprüfung nicht in verschiedenen Tonfällen spricht.
const abhilfeDBNeuLaden = "Datenbankverbindung prüfen und die Seite neu laden."

// befundNichtImEchtbetrieb formuliert den Befund für einen Bereich, den es außerhalb des
// echten Betriebs mit Absicht nicht gibt: kein Mangel, sondern die Umgebung.
func befundNichtImEchtbetrieb(was, umgebung string) string {
	return was + " — in " + umgebung + " ist das richtig so."
}

// Befund ist eine Zeile der Selbstprüfung. Neben dem Befund stehen die Folge und die
// Abhilfe, weil eine Meldung ohne „warum zählt das" und „was ist zu tun" niemanden zum
// Handeln bringt.
type Befund struct {
	Bereich string `json:"bereich"`
	Stufe   string `json:"stufe"`
	Befund  string `json:"befund"`
	Folge   string `json:"folge"`
	Abhilfe string `json:"abhilfe"`
}

// Lage ist der Zustand, über den die Prüfungen urteilen; die Tür trägt ihn zusammen. Ein
// Zeiger oder eine Liste mit nil heißt „nicht erhoben" und wird gewarnt; die leere Liste
// heißt „geprüft, nichts gefunden".
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
	// SchluesselProbe sagt je verschlüsselter Spalte, ob sich ein Wert mit dem laufenden
	// APP_ENCRYPTION_KEY entschlüsseln ließ. nil: nicht lesbar.
	SchluesselProbe *repository.SchluesselProbe

	// Anmeldung
	IMAPHost string
	// SelbstanmeldeDomain ist SELBSTANMELDUNG_DOMAIN. Leer heißt, das Kollegium muss von Hand
	// angelegt werden (auth/selbstanmeldung.go).
	SelbstanmeldeDomain string

	// Aus den Einstellungen in der Datenbank, nicht aus der .env (api/system/mail_settings.go)
	OeffentlicheAdresse string
	SMTPHost            string
	// Die beiden Ausnahmeschalter des Mailversands (mailservice/versand.go), je „steht auf
	// true".
	SMTPUnsicheresTLSErlaubt bool
	SMTPKlartextErlaubt      bool

	// Was scripts/seed_demo.sql angelegt hat (Schüler DEMO-S-, Exemplare DEMO-B-).
	DemoSchueler  int
	DemoExemplare int

	// BeispielLieferanten nennt die Lieferanten mit einer Kennung der früheren Startdaten
	// (Migration 107): Beispiel-Adresse oder Beispiel-Kundennummer. Eine Bestellung an so
	// einen Eintrag geht wirklich hinaus, an niemanden. nil: nicht lesbar.
	BeispielLieferanten []string

	// RechteLive hält die Rechte aus role_permissions als Rolle → Recht → erlaubt. nil heißt
	// „nicht lesbar"; die leere Map meldete jede Zeile der Vorgabe als fehlend.
	RechteLive map[string]map[string]bool

	// Aktive Admin-Konten („Name (Mail)"). nil: nicht lesbar.
	AdminKonten []string

	// AlarmEmpfaenger ist die Einstellung alarm_empfaenger, durch Komma getrennt. Leer gehen
	// die Kritisch-Alarme an alle aktiven Admin-Konten.
	AlarmEmpfaenger string

	// Die Klassennamen verbinden Schüler, Zuordnung der Klassenleitung und Bücherlisten nur
	// als gleicher Text; reißt er, meldet nichts einen Fehler, und die Mahnliste läuft leer.
	KlassenOhneLehrkraft  []string // Klassen bis Jahrgang 10 mit aktiven Schülern, ohne Zuordnungs-Zeile
	VerwaisteZuordnungen  []string // Zuordnungs-Zeilen ohne aktive Schüler
	VerwaisteBuecherliste []string // class_books-Klassen ohne aktive Schüler

	// Das nächtliche Backup überspringt sich ohne Schlüssel und steht still, wenn der
	// Zeitplan stirbt. Die tägliche Alarm-Mail speist sich aus den Befunden, deshalb steht
	// hier, was auch das Abzeichen im Rahmen der Anwendung misst (api/system/backup_status.go).
	// LetztesBackup nil: noch nie gelaufen. Jetzt wird übergeben, damit sich die
	// Altersrechnung testen lässt.
	BackupSchluesselGesetzt bool
	BackupSchluesselSchwach bool
	LetztesBackup           *time.Time
	Jetzt                   time.Time

	// FerientabelleBis ist das letzte Jahr, für das Sommerferien bekannt sind
	// (pkg/lmfplan/sommerferien.go und die Einstellung „Sommerferien"). Danach belegt der
	// LMF-Planer den letzten Tag des Büchertauschs nicht mehr vor.
	FerientabelleBis int

	// UebrigeFerienBis ist das Jahr, in dem die letzten bekannten Herbst-, Weihnachts- und
	// Osterferien enden (pkg/lmfplan/uebrige_ferien.go). Danach rückt eine Leihfrist nur
	// noch über Wochenende, Feiertag und Sommerferien; eine Einstellung dafür gibt es nicht.
	UebrigeFerienBis int

	// RestoreProbe ist das Ergebnis der wöchentlichen Probe, ob sich das jüngste Backup
	// einspielen lässt (jobs/restore_probe.go). nil: noch nie gelaufen oder nicht lesbar.
	RestoreProbe *jobs.RestoreProbeErgebnis

	// LoeschRueckstand nennt je Löschroutine die Datensätze, die seit mindestens einem Tag
	// gelöscht sein müssten. Ein stiller Fehlschlag fiele sonst erst auf, wenn jemand fragt,
	// warum Daten von vor zwei Jahren noch da sind.
	LoeschRueckstand []repository.LoeschRueckstand

	// EhemaligeMitOffenenVorgaengen zählt Ehemalige, deren offenes Buch oder unbezahlte
	// Forderung seit mehr als einem Jahr besteht. Der offene Vorgang schützt Name und
	// Anschrift vor Anonymisierung und Löschung, und keine Routine meldet es.
	EhemaligeMitOffenenVorgaengen *int

	// BescheidFehlend nennt die Pflichtangaben, ohne die die Tür einen Schadensersatz-Bescheid
	// abweist, aus derselben Funktion (BescheidAngaben.FehlendeAngaben).
	BescheidFehlend []string

	// NachbuchMeldungenOffen zählt die Nachbuch-Meldungen der Theke, die seit mehr als 14
	// Tagen niemand quittiert hat: Abweichungen zwischen Offline-Scan und Wirklichkeit.
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

// Pruefe wendet alle Regeln auf eine Lage an und liefert je Bereich einen Befund.
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
