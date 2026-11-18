package auskunft

import (
	"fmt"

	"bibliothek/internal/jobs"
	"bibliothek/internal/pkg/leserart"
	"bibliothek/internal/repository"
)

// dsgvoVerarbeitungsangaben formuliert die Pflichtangaben nach Art. 15 Abs. 1 DSGVO für einen
// Schüler. Jede Frist im Text ist ein Wert der Einstellungen, mit dem auch die Löschjobs
// rechnen: Eine feste Zahl im Satz gäbe der betroffenen Person eine falsche Frist, sobald die
// Schule die Einstellung ändert.
func dsgvoVerarbeitungsangaben(lesehistorieTage, lernmittelTage, karenzTage, protokollMonate int) Verarbeitungsangaben {
	karenz := "sofort nach dem letzten Vorgang"
	if karenzTage > 0 {
		karenz = fmt.Sprintf("nach einer Karenzzeit von %d Tagen ab dem letzten Vorgang, also dem Abgang, der letzten Rückgabe oder der Schadensregulierung", karenzTage)
	}
	frist := func(tage int) string {
		if tage <= 0 {
			return "bis zur Löschung des Schülerdatensatzes; die Befristung ist in dieser Installation abgeschaltet"
		}
		return repository.TageMitZahl(tage) + " nach Rückgabe"
	}
	return Verarbeitungsangaben{
		Zwecke: []string{
			"Verwaltung der Lernmittelausleihe im Rahmen der Lernmittelfreiheit: Nachweis von Ausleihe und Rücklauf, Mahnung, Schadensersatz",
			"Betrieb der Schülerbücherei: Ausleihe, Vormerkung, Rückgabeerinnerung",
			"Abwicklung von Schadens- und Verlustfällen",
		},
		Rechtsgrundlage: "Lernmittelausleihe: Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 HSchG zur Verarbeitung personenbezogener Daten durch Schulen und § 153 HSchG zur Lernmittelfreiheit sowie die SchDSV; Grundlage ist ein öffentlich-rechtliches Nutzungsverhältnis. " +
			"Schülerbücherei: Einwilligung nach Art. 6 Abs. 1 lit. a DSGVO und § 3 SchDSV, denn die Nutzung ist freiwillig; bei Minderjährigen willigen die Erziehungsberechtigten ein. Das gilt, sofern die Schule die Schülerbücherei nicht als schulische Aufgabe nach Art. 6 Abs. 1 lit. e führt; maßgeblich ist das Verzeichnis von Verarbeitungstätigkeiten der Schule.",
		Empfaenger: "Keine Übermittlung an Dritte; Verarbeitung durch das Bibliothekspersonal der Schule. Klassenleitungen erhalten die Liste überfälliger Medien ihrer Klasse. Helfer an der Theke sehen nur Name, Klasse und Sperrstatus.",
		Speicherdauer: "Ausleihvorgänge bleiben der Person zugeordnet: Schülerbücherei " + frist(lesehistorieTage) + ", Lernmittel " + frist(lernmittelTage) + "; danach automatisch getrennt. " +
			"Bearbeitende Person einer Ausleihe nach 14 Tagen entfernt. Schülerdatensatz nach dem Abgang: solange eine Ausleihe offen oder ein Schadensfall unbezahlt ist, bleibt er erhalten; danach wird er " + karenz + " anonymisiert und ab dem 30. Januar des Folgejahres endgültig gelöscht. Ein gelöschter Schülerdatensatz liegt 180 Tage im Papierkorb und wird danach endgültig gelöscht, solange keine Ausleihe und keine unbezahlte Forderung offen ist. Protokolle " + fmt.Sprintf("%d", protokollMonate) + " Monate. " + dsgvoSicherungen,
		Herkunft:          "Stammdaten aus der Lehrer- und Schülerdatenbank (LUSD) der Schule per Export und Import, Übernahme aus dem bisherigen Bibliotheksprogramm mit Name, Klasse, Ausweisnummer und Geburtsdatum oder Erfassung von Hand durch das Bibliotheksteam",
		Betroffenenrechte: dsgvoBetroffenenrechte,
	}
}

// Die Sicherungen nennt die Auskunft jeder Leserart gleich. Die Aufbewahrung der
// Nachtsicherung liest sie aus dem Job (jobs.AufbewahrteNaechte, jobs.AufbewahrteWochen); die zwei
// anderen Fristen gelten in Shell-Skripten (update.sh, scripts/backup.sh), und
// pflichtangaben_sicherungen_test.go hält die Zahlen hier gegen sie.
const (
	sicherungVorUpdateTage = 30 // update.sh, BACKUP_RETENTION_DAYS
	sicherungVonHandTage   = 7  // scripts/backup.sh, RETENTION_ENC_TAGE
)

var dsgvoSicherungen = fmt.Sprintf("Verschlüsselte Sicherungen: die der letzten %d Nächte, "+
	"dazu von den älteren je Kalenderwoche eine, für %d Wochen, zusammen etwa drei Monate; "+
	"eine Sicherung vor einem Update wird beim ersten Update nach %d Tagen gelöscht, "+
	"eine von Hand angelegte beim ersten weiteren Lauf nach %d Tagen.",
	jobs.AufbewahrteNaechte, jobs.AufbewahrteWochen, sicherungVorUpdateTage, sicherungVonHandTage)

// dsgvoBetroffenenrechte gilt für jede Leserart gleich.
const dsgvoBetroffenenrechte = "Recht auf Berichtigung (Art. 16), Löschung (Art. 17), Einschränkung (Art. 18) und Widerspruch (Art. 21) sowie Widerruf einer Einwilligung; Beschwerderecht beim Hessischen Beauftragten für Datenschutz und Informationsfreiheit (HBDI)"

// FristWerte sind die eingestellten Fristen, die die Pflichtangaben nennen.
type FristWerte struct {
	LesehistorieTage, LernmittelTage, KarenzTage, ProtokollMonate, AnliegenTage int
}

// Pflichtangaben wählt die Pflichtangaben nach der Art des Lesers: Für einen Schüler
// gelten Lernmittelfreiheit und Schülerbücherei, für jede andere Art (Kollegium und
// Sonderkonten) das Beschäftigungsverhältnis (pflichtangaben_kollegium.go).
func Pflichtangaben(art string, f FristWerte) Verarbeitungsangaben {
	if leserart.IstSchueler(art) {
		return dsgvoVerarbeitungsangaben(f.LesehistorieTage, f.LernmittelTage, f.KarenzTage, f.ProtokollMonate)
	}
	return dsgvoVerarbeitungsangabenKollegium(f)
}
