package api

import (
	"strings"
	"testing"

	"bibliothek/repository"
)

// Der Alarm-Formatter als reine Funktion: Kritisches kommt mit allen vier Angaben
// in die Mail, Warnungen nur als Zahl, und ohne Kritisches gibt es KEINE Mail —
// ein Alarm, der bei Warnungen feuert, wird stummgeschaltet.
func TestFormatiereAlarmMail(t *testing.T) {
	t.Run("ohne Kritisches keine Mail", func(t *testing.T) {
		_, _, n := formatiereAlarmMail([]Befund{
			{Bereich: "Demo-Daten", Stufe: StufeWarnung, Befund: "2000 Demo-Schüler"},
			{Bereich: "Anmeldung", Stufe: StufeOK},
		})
		if n != 0 {
			t.Fatalf("kritische = %d, erwartet 0", n)
		}
	})

	t.Run("Kritisches traegt Befund, Folge, Abhilfe und die Warnungs-Fussnote", func(t *testing.T) {
		betreff, text, n := formatiereAlarmMail([]Befund{
			{Bereich: "Auslagerung der Backups", Stufe: StufeKritisch,
				Befund: "Kein Ziel außer Haus eingerichtet.", Folge: "Ein Plattenausfall kostet alles.",
				Abhilfe: "S3_ENDPOINT setzen."},
			{Bereich: "Demo-Daten", Stufe: StufeWarnung, Befund: "2000 Demo-Schüler"},
		})
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

// Die Empfängerwahl (Betreiber-Wunsch 17.08.2026): konfiguriert schlägt Verteiler,
// Müll ohne @ fällt raus, und ganz ohne Konfiguration greift der Admin-Rückfall —
// ein Alarm, der niemanden erreicht, ist keiner.
func TestWaehleAlarmEmpfaenger(t *testing.T) {
	admins := []repository.AdminKonto{
		{Name: "Peter Flasch", Email: "pflasch@philipp-reis-schule.de"},
		{Name: "Andrea Trumpfheller", Email: "atrumpfheller@philipp-reis-schule.de"},
	}

	t.Run("konfigurierte Adressen gewinnen", func(t *testing.T) {
		an, beschreibung := waehleAlarmEmpfaenger(" pflasch@philipp-reis-schule.de , it@schule.de ", admins)
		if len(an) != 2 || an[0] != "pflasch@philipp-reis-schule.de" || an[1] != "it@schule.de" {
			t.Fatalf("Empfänger: %v", an)
		}
		if !strings.Contains(beschreibung, "konfigurierten") {
			t.Errorf("Beschreibung nennt den Modus nicht: %q", beschreibung)
		}
	})

	t.Run("leer -> alle aktiven Admins", func(t *testing.T) {
		an, beschreibung := waehleAlarmEmpfaenger("", admins)
		if len(an) != 2 || an[1] != "atrumpfheller@philipp-reis-schule.de" {
			t.Fatalf("Rückfall-Empfänger: %v", an)
		}
		if !strings.Contains(beschreibung, "alle aktiven Admin-Konten") ||
			!strings.Contains(beschreibung, "Andrea Trumpfheller") {
			t.Errorf("Beschreibung unvollständig: %q", beschreibung)
		}
	})

	t.Run("nur Müll ohne At -> Rückfall statt stummem Verteiler", func(t *testing.T) {
		an, _ := waehleAlarmEmpfaenger("kaputt, nochkaputter", admins)
		if len(an) != 2 || an[0] != "pflasch@philipp-reis-schule.de" {
			t.Fatalf("Rückfall griff nicht: %v", an)
		}
	})
}
