package system

// Der Wächter, der sich meldet, statt gelesen werden zu müssen.
//
// Die Selbstprüfung (System → Einstellungen → Betriebsbereitschaft) beantwortet, was
// eingerichtet, aber nicht in Betrieb ist, doch nur dem, der hinschaut; ein Backup, das nicht
// läuft, bliebe so unbemerkt. Dieser Alarm schickt kritische Befunde täglich per Mail an alle
// aktiven Admins, täglich, solange sie bestehen:
// Kritisch soll drücken.
//
// Warnungen lösen bewusst keine Mail aus (die Demo-Daten-Warnung wäre Dauerrauschen,
// und ein Alarm, der dauernd feuert, wird stummgeschaltet); ihre Zahl steht als
// Fußnote in der Mail.

import (
	"context"
	"fmt"
	"log"
	"strings"

	"bibliothek/internal/bereitschaft"
	"bibliothek/mailservice"
	"bibliothek/repository"
)

// BereitschaftsAlarm sammelt die Lage, prüft sie und mailt kritische Befunde.
// Läuft nur im echten Betrieb: Die lokale .env zeigt auf den echten Schul-SMTP —
// ein Entwicklungsrechner darf niemals Alarm-Mails auslösen.
func (t *Tueren) BereitschaftsAlarm(ctx context.Context) {
	einstellungsAbfragen := repository.NewEinstellungsAbfragen(t.DB.Pool)
	mailAbfragen := repository.NewMailEinstellungsAbfragen(t.DB.Pool)
	zustandAbfragen := repository.NewBetriebszustandAbfragen(t.DB.Pool)

	lage := t.sammleLage(ctx, einstellungsAbfragen, mailAbfragen, zustandAbfragen)
	if !bereitschaft.IstEchterBetrieb(lage.Umgebung) {
		return
	}

	befunde := bereitschaft.Pruefe(lage)
	betreff, textkoerper, kritische := formatiereAlarmMail(befunde)
	if kritische == 0 {
		return
	}

	admins, err := zustandAbfragen.LadeAktiveAdmins(ctx)
	if err != nil {
		log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e), aber Admin-Konten nicht lesbar (%v)", kritische, err)
		return
	}
	empfaenger, beschreibung := waehleAlarmEmpfaenger(lage.AlarmEmpfaenger, admins)
	if len(empfaenger) == 0 {
		log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e), aber keine Empfänger (weder konfiguriert noch aktive Admins)", kritische)
		return
	}

	// Jede Mail nennt alle Empfänger: Wer sie liest, sieht sofort, wohin die Alarme gehen,
	// und ein unbekannter Name in dieser Zeile ist selbst ein Befund (ein nie autorisiertes
	// Admin-Konto fiele so auf).
	textkoerper += "\nDiese Mail ging an " + beschreibung + "\n"

	for _, an := range empfaenger {
		if err := t.Post().Sende(mailservice.Nachricht{An: an, Betreff: betreff, Text: textkoerper}); err != nil {
			// Kaputter Mailversand ist selbst ein Befund der Seite — hier bleibt nur
			// das Log; eine Mail über den kaputten Mailversand gibt es nicht.
			log.Printf("Bereitschafts-Alarm an %s fehlgeschlagen: %v", an, err)
		}
	}
	log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e) an %d Empfänger gemeldet", kritische, len(empfaenger))
}

// formatiereAlarmMail baut Betreff und Text aus den Befunden. Reine Funktion —
// prüfbar ohne Mailserver, Datenbank oder Umgebung.
func formatiereAlarmMail(befunde []bereitschaft.Befund) (betreff, textkoerper string, kritische int) {
	var kritischeBefunde []bereitschaft.Befund
	warnungen := 0
	for _, b := range befunde {
		switch b.Stufe {
		case bereitschaft.StufeKritisch:
			kritischeBefunde = append(kritischeBefunde, b)
		case bereitschaft.StufeWarnung:
			warnungen++
		}
	}
	if len(kritischeBefunde) == 0 {
		return "", "", 0
	}

	betreff = fmt.Sprintf("Bibliothek: %d kritische(r) Befund(e) der Betriebsbereitschaft", len(kritischeBefunde))

	var sb strings.Builder
	sb.WriteString("Die tägliche Selbstprüfung meldet Zustände, die vor dem Echtbetrieb zwingend zu klären sind.\n" +
		"Diese Mail kommt täglich, solange sie bestehen.\n\n")
	for _, b := range kritischeBefunde {
		fmt.Fprintf(&sb, "‼ %s\n", b.Bereich)
		fmt.Fprintf(&sb, "   Befund:  %s\n", b.Befund)
		if b.Folge != "" {
			fmt.Fprintf(&sb, "   Folge:   %s\n", b.Folge)
		}
		if b.Abhilfe != "" {
			fmt.Fprintf(&sb, "   Abhilfe: %s\n", b.Abhilfe)
		}
		sb.WriteString("\n")
	}
	if warnungen > 0 {
		fmt.Fprintf(&sb, "Daneben stehen %d Warnung(en) — Details unter System → Einstellungen → Betriebsbereitschaft.\n", warnungen)
	}
	return betreff, sb.String(), len(kritischeBefunde)
}

// waehleAlarmEmpfaenger bestimmt, wohin die kritischen Alarme gehen: Ist in den
// Einstellungen alarm_empfaenger gesetzt (Adressen mit Komma getrennt), gehen sie genau
// dorthin, sonst an alle aktiven Admin-Konten. Der Rückfall ist Absicht: Ein Alarm, der
// niemanden erreicht, ist keiner. Einträge ohne "@" fallen weg, damit ein Tippfehler den
// Verteiler nicht leert, solange gültige Adressen übrig sind; ohne gültige greift der
// Rückfall.
func waehleAlarmEmpfaenger(konfiguriert string, admins []repository.AdminKonto) (an []string, beschreibung string) {
	for _, teil := range strings.Split(konfiguriert, ",") {
		adresse := strings.TrimSpace(teil)
		if strings.Contains(adresse, "@") {
			an = append(an, adresse)
		}
	}
	if len(an) > 0 {
		return an, "die konfigurierten Alarm-Empfänger: " + strings.Join(an, "; ")
	}

	namen := make([]string, 0, len(admins))
	for _, a := range admins {
		an = append(an, a.Email)
		namen = append(namen, a.Name+" ("+a.Email+")")
	}
	return an, "alle aktiven Admin-Konten: " + strings.Join(namen, "; ")
}
