package system

// Der Wächter, der sich meldet, statt gelesen werden zu müssen.
//
// Die Selbstprüfung (System → Einstellungen → Betriebsbereitschaft) beantwortet, was
// eingerichtet, aber nicht in Betrieb ist, doch nur dem, der hinschaut; ein Backup, das nicht
// läuft, bliebe so unbemerkt. Dieser Alarm schickt kritische Befunde täglich per Mail an alle
// aktiven Admins, täglich, solange sie bestehen:
// Kritisch soll drücken. Ein Neustart wiederholt die letzte Mail nicht
// (internal/bereitschaft/alarm.go).
//
// Warnungen lösen eine Mail nur mit dem Schalter alarm_warnungen aus (Einstellungen →
// Erreichbarkeit & Alarme): Ohne ihn wäre die Demo-Daten-Warnung Dauerrauschen, und ein Alarm,
// der dauernd feuert, wird stummgeschaltet; ihre Zahl steht dann als Fußnote in der Mail.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"bibliothek/internal/bereitschaft"
	"bibliothek/mailservice"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// alarmGemeldetSchluessel ist der Eintrag in system_einstellungen, der festhält, was die letzte
// Alarm-Mail nannte (bereitschaft.GemeldeterAlarm).
const alarmGemeldetSchluessel = "bereitschafts_alarm_gemeldet"

// BereitschaftsAlarm sammelt die Lage, prüft sie und mailt kritische Befunde und die Fehler im
// Log, wenn ein neuer Punkt dabei ist oder die letzte Mail einen Tag zurückliegt. Läuft nur im
// echten Betrieb: Die lokale .env zeigt auf den echten Schul-SMTP — ein Entwicklungsrechner darf
// niemals Alarm-Mails auslösen.
func (t *Tueren) BereitschaftsAlarm(ctx context.Context) {
	einstellungsAbfragen := repository.NewEinstellungsAbfragen(t.DB.Pool)
	mailAbfragen := repository.NewMailEinstellungsAbfragen(t.DB.Pool)
	zustandAbfragen := repository.NewBetriebszustandAbfragen(t.DB.Pool)

	lage := t.sammleLage(ctx, einstellungsAbfragen, mailAbfragen, zustandAbfragen)
	if !bereitschaft.IstEchterBetrieb(lage.Umgebung) {
		return
	}

	befunde := bereitschaft.Pruefe(lage)
	gruppen := t.fehlergruppen(lage.Jetzt)
	fehlerGruppen, warnGruppen := trenneWarnungen(gruppen)
	fehler := baueFehlerImLog(fehlerGruppen, lage.Jetzt, hoechstzahlMeldungenInDerMail)
	warnungen := alarmWarnungen{an: lage.AlarmWarnungen}
	if warnungen.an {
		warnungen.imLog = baueWarnungenImLog(warnGruppen, lage.Jetzt, hoechstzahlMeldungenInDerMail)
	}
	betreff, textkoerper, kritische := formatiereAlarmMail(befunde, fehler, warnungen)
	if betreff == "" {
		return
	}
	punkte := bereitschaft.AlarmPunkte(befunde, gruppen, lage.AlarmWarnungen)
	zuletzt := ladeGemeldetenAlarm(ctx, zustandAbfragen)
	if !bereitschaft.IstMailFaellig(zuletzt, punkte, lage.Jetzt) {
		slog.Info("Bereitschafts-Alarm: nichts Neues seit der letzten Mail", "befunde", kritische,
			"fehlermeldungen", fehler.Anzahl, "letzte_mail", zuletzt.Zeit)
		return
	}

	admins, err := zustandAbfragen.LadeAktiveAdmins(ctx)
	if err != nil {
		slog.Error("Bereitschafts-Alarm: zu melden, aber Admin-Konten nicht lesbar",
			"befunde", kritische, "fehlermeldungen", fehler.Anzahl, "fehler", err)
		return
	}
	empfaenger, beschreibung := waehleAlarmEmpfaenger(lage.AlarmEmpfaenger, admins)
	if len(empfaenger) == 0 {
		slog.Error("Bereitschafts-Alarm: zu melden, aber keine Empfänger (weder konfiguriert noch aktive Admins)",
			"befunde", kritische, "fehlermeldungen", fehler.Anzahl)
		return
	}

	// Jede Mail nennt alle Empfänger: Wer sie liest, sieht sofort, wohin die Alarme gehen,
	// und ein unbekannter Name in dieser Zeile ist selbst ein Befund (ein nie autorisiertes
	// Admin-Konto fiele so auf).
	textkoerper += "\nDiese Mail ging an " + beschreibung + "\n"

	zugestellt := 0
	for _, an := range empfaenger {
		if err := t.Post().Sende(ctx, mailservice.Nachricht{An: an, Betreff: betreff, Text: textkoerper}); err != nil {
			// Kaputter Mailversand ist selbst ein Befund der Seite — hier bleibt nur
			// das Log; eine Mail über den kaputten Mailversand gibt es nicht.
			slog.Error("Bereitschafts-Alarm: Versand fehlgeschlagen", "empfaenger", an, "fehler", err)
			continue
		}
		zugestellt++
	}
	// Gemeldet ist der Alarm erst, wenn ihn jemand bekam; sonst geht er beim nächsten Lauf noch
	// einmal hinaus.
	if zugestellt == 0 {
		return
	}
	schreibeGemeldetenAlarm(ctx, zustandAbfragen, bereitschaft.GemeldeterAlarm{Zeit: lage.Jetzt, Punkte: punkte})
	slog.Info("Bereitschafts-Alarm: gemeldet", "befunde", kritische, "fehlermeldungen", fehler.Anzahl,
		"anzahl_empfaenger", zugestellt)
}

// ladeGemeldetenAlarm liest, was die letzte Alarm-Mail nannte. Fehlt der Eintrag oder ist er nicht
// lesbar, gilt nichts als gemeldet: lieber eine Mail zu viel als ein verschluckter Alarm.
func ladeGemeldetenAlarm(ctx context.Context, zustand *repository.BetriebszustandAbfragen) bereitschaft.GemeldeterAlarm {
	var alarm bereitschaft.GemeldeterAlarm
	roh, err := zustand.LadeEinstellungswert(ctx, alarmGemeldetSchluessel)
	if err == nil {
		err = json.Unmarshal([]byte(roh), &alarm)
	}
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("Bereitschafts-Alarm: letzte Mail nicht lesbar, die Mail geht hinaus", "fehler", err)
		}
		return bereitschaft.GemeldeterAlarm{}
	}
	return alarm
}

// schreibeGemeldetenAlarm hält fest, was die Mail nannte; scheitert das, kommt sie beim nächsten
// Lauf noch einmal.
func schreibeGemeldetenAlarm(ctx context.Context, zustand *repository.BetriebszustandAbfragen, alarm bereitschaft.GemeldeterAlarm) {
	wert, err := json.Marshal(alarm)
	if err == nil {
		err = zustand.SchreibeEinstellungswert(ctx, alarmGemeldetSchluessel, string(wert))
	}
	if err != nil {
		slog.Warn("Bereitschafts-Alarm: Mail nicht als gemeldet festgehalten, sie kommt beim nächsten Lauf noch einmal",
			"fehler", err)
	}
}

// alarmWarnungen ist, was die Mail mit dem Schalter alarm_warnungen zusätzlich nennt. Der
// Nullwert heißt: Schalter aus.
type alarmWarnungen struct {
	an    bool
	imLog fehlerImLogAntwort
}

// formatiereAlarmMail baut Betreff und Text aus den Befunden, den Fehlern im Log und, mit dem
// Schalter, den Warnungen. Reine Funktion — prüfbar ohne Mailserver, Datenbank oder Umgebung.
// Ohne etwas zu melden bleibt der Betreff leer, und es geht keine Mail hinaus.
func formatiereAlarmMail(befunde []bereitschaft.Befund, fehler fehlerImLogAntwort, warnungen alarmWarnungen) (betreff, textkoerper string, kritische int) {
	kritischeBefunde, warnBefunde := teileBefunde(befunde)
	// Gezählt wird nur, was die Mail ausschreibt: Warnungen nur mit dem Schalter.
	fehlerImLog, gemeldeteWarnBefunde, warnungenImLog := 0, 0, 0
	if fehler.Stufe == bereitschaft.StufeKritisch {
		fehlerImLog = fehler.Anzahl
	}
	if warnungen.an {
		gemeldeteWarnBefunde, warnungenImLog = len(warnBefunde), warnungen.imLog.Anzahl
	}
	if len(kritischeBefunde)+fehlerImLog+gemeldeteWarnBefunde+warnungenImLog == 0 {
		return "", "", 0
	}
	betreff = alarmBetreff(len(kritischeBefunde), fehlerImLog, gemeldeteWarnBefunde, warnungenImLog)

	var sb strings.Builder
	sb.WriteString("Die tägliche Selbstprüfung meldet Zustände, die zu klären sind.\n" +
		"Diese Mail kommt täglich, solange sie bestehen.\n\n")
	for _, b := range kritischeBefunde {
		schreibeAlarmTeil(&sb, zeichenKritisch, b, nil)
	}
	if fehlerImLog > 0 {
		schreibeAlarmTeil(&sb, zeichenKritisch, bereitschaft.Befund{
			Bereich: "Fehler im Log", Befund: fehler.Befund, Folge: fehler.Folge, Abhilfe: fehler.Abhilfe,
		}, fehler.Meldungen)
	}
	schreibeWarnungsteil(&sb, warnBefunde, warnungen)
	return betreff, sb.String(), len(kritischeBefunde)
}

// teileBefunde trennt die kritischen Befunde von den Warnungen; Befunde in Ordnung fallen weg.
func teileBefunde(befunde []bereitschaft.Befund) (kritisch, warnung []bereitschaft.Befund) {
	for _, b := range befunde {
		switch b.Stufe {
		case bereitschaft.StufeKritisch:
			kritisch = append(kritisch, b)
		case bereitschaft.StufeWarnung:
			warnung = append(warnung, b)
		}
	}
	return kritisch, warnung
}

// alarmBetreff nennt im Betreff, was die Mail ausschreibt, je mit seiner Zahl; eine Zahl 0 fehlt.
func alarmBetreff(kritische, fehlerImLog, warnBefunde, warnungenImLog int) string {
	var teile []string
	if kritische > 0 {
		teile = append(teile, fmt.Sprintf("%d kritische(r) Befund(e) der Betriebsbereitschaft", kritische))
	}
	if fehlerImLog > 0 {
		teile = append(teile, fehlermeldungen(fehlerImLog)+" im Log")
	}
	if warnBefunde > 0 {
		teile = append(teile, fmt.Sprintf("%d Warnung(en) der Betriebsbereitschaft", warnBefunde))
	}
	if warnungenImLog > 0 {
		teile = append(teile, warnungenText(warnungenImLog)+" im Log")
	}
	return "Bibliothek: " + strings.Join(teile, ", ")
}

// schreibeWarnungsteil schreibt die Warnungen: mit dem Schalter ausgeschrieben, die der
// Betriebsbereitschaft und die im Log; ohne ihn nur die Zahl der Warnungen der
// Betriebsbereitschaft als Fußnote.
func schreibeWarnungsteil(sb *strings.Builder, warnBefunde []bereitschaft.Befund, warnungen alarmWarnungen) {
	if !warnungen.an {
		if len(warnBefunde) > 0 {
			fmt.Fprintf(sb, "Daneben stehen %d Warnung(en) — Details unter System → Einstellungen → Betriebsbereitschaft.\n", len(warnBefunde))
		}
		return
	}
	for _, b := range warnBefunde {
		schreibeAlarmTeil(sb, zeichenWarnung, b, nil)
	}
	if l := warnungen.imLog; l.Anzahl > 0 {
		schreibeAlarmTeil(sb, zeichenWarnung, bereitschaft.Befund{
			Bereich: "Warnungen im Log", Befund: l.Befund, Folge: l.Folge, Abhilfe: l.Abhilfe,
		}, l.Meldungen)
	}
}

// Die Zeichen vor dem Bereich: Kritisches und Fehler drücken, Warnungen nicht.
const (
	zeichenKritisch = "‼"
	zeichenWarnung  = "⚠"
)

// schreibeAlarmTeil schreibt einen Teil der Mail: Zeichen, Bereich und Befund, die Meldungen
// darunter, dann Folge und Abhilfe.
func schreibeAlarmTeil(sb *strings.Builder, zeichen string, b bereitschaft.Befund, meldungen []fehlerZeile) {
	fmt.Fprintf(sb, "%s %s\n", zeichen, b.Bereich)
	fmt.Fprintf(sb, "   Befund:  %s\n", b.Befund)
	for _, m := range meldungen {
		fmt.Fprintf(sb, "            %s\n", m.Text)
		if m.Zusatz != "" {
			fmt.Fprintf(sb, "              %s\n", m.Zusatz)
		}
		if m.Zeitraum != "" {
			fmt.Fprintf(sb, "              Zeit: %s\n", m.Zeitraum)
		}
		if m.Logzeile != "" {
			fmt.Fprintf(sb, "              Letzte Zeile im Log: %s\n", m.Logzeile)
		}
	}
	if b.Folge != "" {
		fmt.Fprintf(sb, "   Folge:   %s\n", b.Folge)
	}
	if b.Abhilfe != "" {
		fmt.Fprintf(sb, "   Abhilfe: %s\n", b.Abhilfe)
	}
	sb.WriteString("\n")
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
