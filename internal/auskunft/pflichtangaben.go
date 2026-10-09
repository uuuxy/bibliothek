package auskunft

import (
	"fmt"

	"bibliothek/jobs"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"
)

// dsgvoVerarbeitungsangaben formuliert die Pflichtangaben nach Art. 15 Abs. 1 DSGVO —
// aus DENSELBEN Fristen, mit denen das System arbeitet (Einstellungen „Datenschutz &
// Sitzung"). Bis 22.08.2026 stand hier pauschal lit. e ohne Fristen, während SECURITY.md,
// VVT- und Art.-13-Entwurf längst zwei Tätigkeiten mit zwei Grundlagen und 90/730 Tagen
// beschrieben — eine Betroffenenauskunft mit falscher Pflichtangabe (Prüfung 22.08., B).
// dsgvoVerarbeitungsangaben baut die Pflichtangaben aus den EINGESTELLTEN Fristen — die
// Karenzzeit vor der Anonymisierung (abgaenger_karenz_tage) ebenso wie die Lesehistorie.
// Bis 02.09.2026 stand hier ein festes „Altfälle nach 360 Tagen", während der Job längst
// mit der Karenz rechnete: eine Pflichtangabe an die betroffene Person, die nicht stimmte.
//
// DRITTER Fall derselben Klasse, gefunden am 10.09.2026: „Protokolle 24 Monate" stand
// starr im Text, während die Aufbewahrung der beiden Protokolle seit dem 16.08.2026 eine
// Einstellung ist (audit_aufbewahrung_monate, Untergrenze 6). Wer sie auf 6 stellte, gab
// der betroffenen Person eine falsche Frist. Jetzt liest die Auskunft dieselbe Quelle wie
// der Löschjob (jobs/cron_audit_retention.go) und der Rückstands-Wächter
// (repository/loeschrueckstand.go): AufbewahrungMonateOderStandard.
func dsgvoVerarbeitungsangaben(lesehistorieTage, lernmittelTage, karenzTage, auditMonate int) DsgvoVerarbeitungsangaben {
	karenz := "sofort nach dem letzten Vorgang"
	if karenzTage > 0 {
		karenz = fmt.Sprintf("nach einer Karenzzeit von %d Tagen ab dem letzten Vorgang (Abgang, letzte Rückgabe oder Schadensregulierung)", karenzTage)
	}
	frist := func(tage int) string {
		if tage <= 0 {
			return "bis zur Löschung des Schülerdatensatzes (Befristung in dieser Installation abgeschaltet)"
		}
		return repository.TageMitZahl(tage) + " nach Rückgabe"
	}
	return DsgvoVerarbeitungsangaben{
		Zwecke: []string{
			"Verwaltung der Lernmittelausleihe im Rahmen der Lernmittelfreiheit (Nachweis von Ausleihe und Rücklauf, Mahnung, Schadensersatz)",
			"Betrieb der Schülerbücherei (Ausleihe, Vormerkung, Rückgabeerinnerung)",
			"Abwicklung von Schadens- und Verlustfällen",
		},
		Rechtsgrundlage: "Lernmittelausleihe: Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 HSchG (Erhebung und Verarbeitung personenbezogener Daten durch Schulen) und § 153 HSchG (Lernmittelfreiheit) sowie SchDSV — öffentlich-rechtliches Nutzungsverhältnis. " +
			"Schülerbücherei: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO, § 3 SchDSV (freiwillige Nutzung; bei Minderjährigen durch die Erziehungsberechtigten), sofern die Schule sie nicht als schulische Aufgabe nach Art. 6 Abs. 1 lit. e führt — maßgeblich ist das Verzeichnis von Verarbeitungstätigkeiten der Schule.",
		Empfaenger: "Keine Übermittlung an Dritte; Verarbeitung durch das Bibliothekspersonal der Schule. Klassenleitungen erhalten die Liste überfälliger Medien ihrer Klasse. Helfer an der Theke sehen nur Name, Klasse und Sperrstatus.",
		Speicherdauer: "Ausleihvorgänge bleiben der Person zugeordnet: Schülerbücherei " + frist(lesehistorieTage) + ", Lernmittel " + frist(lernmittelTage) + "; danach automatisch getrennt. " +
			"Bearbeitende Person einer Ausleihe nach 14 Tagen entfernt. Schülerdatensatz nach dem Abgang: solange eine Ausleihe offen oder ein Schadensfall unbezahlt ist, bleibt er erhalten; danach wird er " + karenz + " anonymisiert und ab dem 30. Januar des Folgejahres endgültig gelöscht. Papierkorb nach 180 Tagen. Protokolle " + fmt.Sprintf("%d", auditMonate) + " Monate. " + dsgvoSicherungen,
		Herkunft:          "Stammdaten aus der Lehrer- und Schülerdatenbank (LUSD) der Schule (Export/Import), Übernahme aus dem bisherigen Bibliotheksprogramm (Name, Klasse, Ausweisnummer, Geburtsdatum) bzw. manuelle Erfassung durch das Bibliotheksteam",
		Betroffenenrechte: dsgvoBetroffenenrechte,
	}
}

// Die Sicherungen nennt die Auskunft jeder Leserart gleich. Die Aufbewahrung der
// Nachtsicherung liest sie aus dem Job (jobs.BehalteNaechte, jobs.BehalteWochen); die zwei
// anderen Fristen stehen in Shell-Skripten (update.sh, scripts/backup.sh), und
// pflichtangaben_sicherungen_test.go hält sie deckungsgleich. Bis zum 28.09.2026 stand hier nur
// „Verschlüsselte Backups 14 Tage" — die Sicherung vor einem Update und die von Hand fehlten.
// Die wöchentlichen Stände gibt es seit dem 29.09.2026.
const (
	sicherungVorUpdateTage = 30 // update.sh, BACKUP_RETENTION_DAYS
	sicherungVonHandTage   = 7  // scripts/backup.sh, RETENTION_ENC_TAGE
)

var dsgvoSicherungen = fmt.Sprintf("Verschlüsselte Sicherungen: die der letzten %d Nächte, "+
	"dazu von den älteren je Kalenderwoche eine, für %d Wochen (zusammen etwa drei Monate); "+
	"eine Sicherung vor einem Update wird beim ersten Update nach %d Tagen gelöscht, "+
	"eine von Hand angelegte beim ersten weiteren Lauf nach %d Tagen.",
	jobs.BehalteNaechte, jobs.BehalteWochen, sicherungVorUpdateTage, sicherungVonHandTage)

// dsgvoBetroffenenrechte gilt für jede Leserart gleich.
const dsgvoBetroffenenrechte = "Recht auf Berichtigung (Art. 16), Löschung (Art. 17), Einschränkung (Art. 18) und Widerspruch (Art. 21) sowie Widerruf einer Einwilligung; Beschwerderecht beim Hessischen Beauftragten für Datenschutz und Informationsfreiheit (HBDI)"

// DsgvoFristWerte sind die eingestellten Fristen, die die Pflichtangaben nennen.
type DsgvoFristWerte struct {
	LesehistorieTage, LernmittelTage, KarenzTage, AuditMonate, AnliegenTage int
}

// DsgvoPflichtangaben wählt die Pflichtangaben nach der Art des Lesers: Für einen Schüler
// gelten Lernmittelfreiheit und Schülerbücherei, für jede andere Art (Kollegium und
// Sonderkonten, Migration 153) das Beschäftigungsverhältnis (pflichtangaben_kollegium.go).
func DsgvoPflichtangaben(art string, f DsgvoFristWerte) DsgvoVerarbeitungsangaben {
	if leserart.IstSchueler(art) {
		return dsgvoVerarbeitungsangaben(f.LesehistorieTage, f.LernmittelTage, f.KarenzTage, f.AuditMonate)
	}
	return dsgvoVerarbeitungsangabenKollegium(f)
}
