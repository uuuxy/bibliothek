package api

// betriebsbereitschaft_alarm.go — der Wächter, der sich meldet, statt gelesen werden
// zu müssen.
//
// Die Selbstprüfung (System → Einstellungen → Betriebsbereitschaft) beantwortet, was
// eingerichtet, aber nicht in Betrieb ist — aber nur für den, der hinschaut. Genau so
// lief das Backup wochenlang nicht (BACKUP_ENCRYPTION_KEY kam nicht im Container an):
// Der Wächter existierte, gehört hat ihn niemand. Dieser Alarm schickt KRITISCHE
// Befunde täglich per Mail an alle aktiven Admins — täglich, solange sie bestehen:
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

	"bibliothek/repository"
)

// BereitschaftsAlarm sammelt die Lage, prüft sie und mailt kritische Befunde.
// Läuft NUR im echten Betrieb: Die lokale .env zeigt auf den echten Schul-SMTP —
// ein Entwicklungsrechner darf niemals Alarm-Mails auslösen.
func (s *Server) BereitschaftsAlarm(ctx context.Context) {
	settingsRepo := repository.NewSystemSettingsRepository(s.DB.Pool)
	mailRepo := repository.NewMailSettingsRepository(s.DB.Pool)
	zustandRepo := repository.NewBetriebszustandRepository(s.DB.Pool)

	lage := s.sammleLage(ctx, settingsRepo, mailRepo, zustandRepo)
	if !istEchterBetrieb(lage.AppEnv) {
		return
	}

	befunde := Pruefe(lage)
	betreff, textkoerper, kritische := formatiereAlarmMail(befunde)
	if kritische == 0 {
		return
	}

	empfaenger, err := zustandRepo.AktiveAdminMails(ctx)
	if err != nil || len(empfaenger) == 0 {
		log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e), aber keine Admin-Adressen ermittelbar (%v)", kritische, err)
		return
	}

	for _, an := range empfaenger {
		if err := SendEmail(MailRequest{To: an, Subject: betreff, Body: textkoerper}); err != nil {
			// Kaputter Mailversand ist selbst ein Befund der Seite — hier bleibt nur
			// das Log; eine Mail über den kaputten Mailversand gibt es nicht.
			log.Printf("Bereitschafts-Alarm an %s fehlgeschlagen: %v", an, err)
		}
	}
	log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e) an %d Admin(s) gemeldet", kritische, len(empfaenger))
}

// formatiereAlarmMail baut Betreff und Text aus den Befunden. Reine Funktion —
// prüfbar ohne Mailserver, Datenbank oder Umgebung.
func formatiereAlarmMail(befunde []Befund) (betreff, textkoerper string, kritische int) {
	var kritischeBefunde []Befund
	warnungen := 0
	for _, b := range befunde {
		switch b.Stufe {
		case StufeKritisch:
			kritischeBefunde = append(kritischeBefunde, b)
		case StufeWarnung:
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
		sb.WriteString("‼ " + b.Bereich + "\n")
		sb.WriteString("   Befund:  " + b.Befund + "\n")
		if b.Folge != "" {
			sb.WriteString("   Folge:   " + b.Folge + "\n")
		}
		if b.Abhilfe != "" {
			sb.WriteString("   Abhilfe: " + b.Abhilfe + "\n")
		}
		sb.WriteString("\n")
	}
	if warnungen > 0 {
		fmt.Fprintf(&sb, "Daneben stehen %d Warnung(en) — Details unter System → Einstellungen → Betriebsbereitschaft.\n", warnungen)
	}
	return betreff, sb.String(), len(kritischeBefunde)
}
