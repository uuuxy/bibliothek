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

	admins, err := zustandRepo.AktiveAdmins(ctx)
	if err != nil {
		log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e), aber Admin-Konten nicht lesbar (%v)", kritische, err)
		return
	}
	empfaenger, beschreibung := waehleAlarmEmpfaenger(lage.AlarmEmpfaenger, admins)
	if len(empfaenger) == 0 {
		log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e), aber keine Empfänger (weder konfiguriert noch aktive Admins)", kritische)
		return
	}

	// Jede Mail nennt ALLE Empfänger: Wer sie liest, sieht sofort, wohin die Alarme
	// gehen — ein unbekannter Name in dieser Zeile ist selbst ein Befund (genau so
	// fiel am 16.08.2026 ein nie autorisiertes Admin-Konto auf).
	textkoerper += "\nDiese Mail ging an " + beschreibung + "\n"

	for _, an := range empfaenger {
		if err := SendEmail(MailRequest{To: an, Subject: betreff, Body: textkoerper}); err != nil {
			// Kaputter Mailversand ist selbst ein Befund der Seite — hier bleibt nur
			// das Log; eine Mail über den kaputten Mailversand gibt es nicht.
			log.Printf("Bereitschafts-Alarm an %s fehlgeschlagen: %v", an, err)
		}
	}
	log.Printf("Bereitschafts-Alarm: %d kritische(r) Befund(e) an %d Empfänger gemeldet", kritische, len(empfaenger))
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

// waehleAlarmEmpfaenger bestimmt, wohin die Kritisch-Alarme gehen (Betreiber-Wunsch
// 17.08.2026): Ist in den Einstellungen alarm_empfaenger gesetzt (kommaseparierte
// Adressen), gehen sie GENAU dorthin — sonst an alle aktiven Admin-Konten. Der
// Rückfall ist Absicht: Ein Alarm, der niemanden erreicht, ist keiner. Einträge ohne
// "@" werden verworfen (Tippfehler sollen den Verteiler nicht stumm leeren, solange
// noch gültige Adressen übrig sind — ganz ohne gültige greift der Rückfall).
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
