package system

import (
	"strings"
	"testing"

	"bibliothek/internal/bereitschaft"
	"bibliothek/repository"
)

// Der Alarm-Formatter als reine Funktion: Kritisches kommt mit allen vier Angaben
// in die Mail, Warnungen nur als Zahl, und ohne Kritisches gibt es keine Mail —
// ein Alarm, der bei Warnungen feuert, wird stummgeschaltet.
func TestFormatiereAlarmMail(t *testing.T) {
	t.Run("ohne Kritisches keine Mail", func(t *testing.T) {
		betreff, _, n := formatiereAlarmMail([]bereitschaft.Befund{
			{Bereich: "Demo-Daten", Stufe: bereitschaft.StufeWarnung, Befund: "2000 Demo-Schüler"},
			{Bereich: "Anmeldung", Stufe: bereitschaft.StufeOK},
		}, fehlerImLogAntwort{Stufe: bereitschaft.StufeOK})
		if n != 0 || betreff != "" {
			t.Fatalf("kritische = %d, Betreff %q, erwartet 0 und keine Mail", n, betreff)
		}
	})

	t.Run("Kritisches traegt Befund, Folge, Abhilfe und die Warnungs-Fussnote", func(t *testing.T) {
		betreff, text, n := formatiereAlarmMail([]bereitschaft.Befund{
			{Bereich: "Auslagerung der Backups", Stufe: bereitschaft.StufeKritisch,
				Befund: "Kein Ziel außer Haus eingerichtet.", Folge: "Ein Plattenausfall kostet alles.",
				Abhilfe: "S3_ENDPOINT setzen."},
			{Bereich: "Demo-Daten", Stufe: bereitschaft.StufeWarnung, Befund: "2000 Demo-Schüler"},
		}, fehlerImLogAntwort{Stufe: bereitschaft.StufeOK})
		if n != 1 {
			t.Fatalf("kritische = %d, erwartet 1", n)
		}
		if !strings.Contains(betreff, "1 kritische(r) Befund(e)") {
			t.Errorf("Betreff nennt die Zahl nicht: %q", betreff)
		}
		for _, muss := range []string{
			"Auslagerung der Backups", "Kein Ziel außer Haus", "Plattenausfall", "S3_ENDPOINT",
			"1 Warnung(en)", "Betriebsbereitschaft",
		} {
			if !strings.Contains(text, muss) {
				t.Errorf("Mail-Text ohne %q:\n%s", muss, text)
			}
		}
	})
}

// Die Fehler im Log haben einen eigenen Teil der Mail, jede Meldung unter dem Befund, die Ursache
// darunter, und der Betreff nennt sie. Schon sie allein lösen die Mail aus; ohne Fehler und ohne
// kritischen Befund geht keine hinaus, auch nicht, wenn der Server nicht zählt.
func TestFormatiereAlarmMail_FehlerImLog(t *testing.T) {
	fehler := fehlerImLogAntwort{
		Stufe: bereitschaft.StufeKritisch, Anzahl: 3,
		Befund: "3 Fehlermeldungen in den letzten 24 Stunden, zuletzt heute um 11:47.",
		Meldungen: []fehlerZeile{
			{Text: "Anfrage mit Serverfehler beendet", Zusatz: "2 × · zuletzt 11:47"},
			{Text: "Backup: S3 upload failed", Zusatz: "1 × · 02:31 · 403 AccessDenied"},
		},
		Folge: "Eine Fehlermeldung heißt: Etwas ist schiefgegangen.", Abhilfe: "Befehl im Betriebshandbuch.",
	}
	kritisch := bereitschaft.Befund{Bereich: "Auslagerung der Backups", Stufe: bereitschaft.StufeKritisch,
		Befund: "Kein Ziel außer Haus.", Folge: "Ein Plattenausfall kostet alles.", Abhilfe: "S3_ENDPOINT setzen."}

	betreff, text, n := formatiereAlarmMail(nil, fehler)
	if n != 0 || betreff != "Bibliothek: 3 Fehlermeldungen im Log" {
		t.Errorf("nur Fehler: kritische = %d, Betreff %q, erwartet 0 und den Betreff der Fehler", n, betreff)
	}
	want := "‼ Fehler im Log\n" +
		"   Befund:  3 Fehlermeldungen in den letzten 24 Stunden, zuletzt heute um 11:47.\n" +
		"            Anfrage mit Serverfehler beendet\n" +
		"              2 × · zuletzt 11:47\n" +
		"            Backup: S3 upload failed\n" +
		"              1 × · 02:31 · 403 AccessDenied\n" +
		"   Folge:   Eine Fehlermeldung heißt: Etwas ist schiefgegangen.\n" +
		"   Abhilfe: Befehl im Betriebshandbuch.\n"
	if !strings.Contains(text, want) {
		t.Errorf("Mail-Text ohne den Teil\n%s\nist:\n%s", want, text)
	}
	if !strings.HasPrefix(text, "Die tägliche Selbstprüfung meldet Zustände, die zu klären sind.\n") {
		t.Errorf("erster Satz der Mail: %q", strings.SplitN(text, "\n", 2)[0])
	}

	betreff, text, n = formatiereAlarmMail([]bereitschaft.Befund{kritisch}, fehler)
	if want := "Bibliothek: 1 kritische(r) Befund(e) der Betriebsbereitschaft, 3 Fehlermeldungen im Log"; n != 1 || betreff != want {
		t.Errorf("beides: kritische = %d, Betreff %q, erwartet 1 und %q", n, betreff, want)
	}
	if i, j := strings.Index(text, "‼ Auslagerung der Backups"), strings.Index(text, "‼ Fehler im Log"); i < 0 || j < i {
		t.Errorf("beides: Befund an %d, Fehler an %d, erwartet erst den Befund, dann die Fehler:\n%s", i, j, text)
	}

	for _, stufe := range []string{bereitschaft.StufeOK, bereitschaft.StufeWarnung} {
		if betreff, _, _ := formatiereAlarmMail(nil, fehlerImLogAntwort{Stufe: stufe}); betreff != "" {
			t.Errorf("Fehler im Log mit Stufe %q: Betreff %q, erwartet keine Mail", stufe, betreff)
		}
	}
}

// Die Wahl der Empfänger: konfiguriert schlägt Verteiler, Müll ohne @ fällt raus, und ganz
// ohne Konfiguration greift der Admin-Rückfall; ein Alarm, der niemanden erreicht, ist keiner.
func TestWaehleAlarmEmpfaenger(t *testing.T) {
	admins := []repository.AdminKonto{
		{Name: "Nina Berger", Email: "nberger@philipp-reis-schule.de"},
		{Name: "Jonas Keller", Email: "jkeller@philipp-reis-schule.de"},
	}

	t.Run("konfigurierte Adressen gewinnen", func(t *testing.T) {
		an, beschreibung := waehleAlarmEmpfaenger(" nberger@philipp-reis-schule.de , it@schule.de ", admins)
		if len(an) != 2 || an[0] != "nberger@philipp-reis-schule.de" || an[1] != "it@schule.de" {
			t.Fatalf("Empfänger: %v", an)
		}
		if !strings.Contains(beschreibung, "konfigurierten") {
			t.Errorf("Beschreibung nennt den Modus nicht: %q", beschreibung)
		}
	})

	t.Run("leer -> alle aktiven Admins", func(t *testing.T) {
		an, beschreibung := waehleAlarmEmpfaenger("", admins)
		if len(an) != 2 || an[1] != "jkeller@philipp-reis-schule.de" {
			t.Fatalf("Rückfall-Empfänger: %v", an)
		}
		if !strings.Contains(beschreibung, "alle aktiven Admin-Konten") ||
			!strings.Contains(beschreibung, "Jonas Keller") {
			t.Errorf("Beschreibung unvollständig: %q", beschreibung)
		}
	})

	t.Run("nur Müll ohne At -> Rückfall statt stummem Verteiler", func(t *testing.T) {
		an, _ := waehleAlarmEmpfaenger("kaputt, nochkaputter", admins)
		if len(an) != 2 || an[0] != "nberger@philipp-reis-schule.de" {
			t.Fatalf("Rückfall griff nicht: %v", an)
		}
	})
}
