package api

import (
	"strings"
	"testing"
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
