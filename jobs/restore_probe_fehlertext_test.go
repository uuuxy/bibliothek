package jobs

import (
	"strings"
	"testing"
)

// Der psql-Fehlertext landet in der DB, auf der Betriebsbereitschafts-Seite und in der
// Alarm-Mail — Datenkontext (CONTEXT: COPY schueler, line 12, column vorname: "…") hat
// dort nichts verloren (Prüfung 22.08.2026).
func TestKuerzeFehlertext_EntferntDatenkontext(t *testing.T) {
	roh := "ERROR:  invalid input syntax for type date: \"31.02.2012\"\n" +
		"CONTEXT:  COPY schueler, line 12, column geburtsdatum: \"31.02.2012\"\n" +
		"DETAIL:  Schüler Mia Muster\n" +
		"HINT:  irgendwas\n" +
		"psql:<stdin>:4711: error: ...\n"
	got := kuerzeFehlertext(roh)
	for _, verboten := range []string{"CONTEXT", "Mia Muster", "DETAIL", "HINT", "COPY schueler"} {
		if strings.Contains(got, verboten) {
			t.Errorf("Datenkontext %q überlebt im Fehlertext: %q", verboten, got)
		}
	}
	if !strings.Contains(got, "invalid input syntax") || !strings.Contains(got, "psql:<stdin>:4711") {
		t.Errorf("die eigentliche Fehlermeldung muss bleiben: %q", got)
	}
	lang := kuerzeFehlertext(strings.Repeat("x", 900))
	if len(lang) > 510 {
		t.Errorf("Kürzung auf 500 Zeichen fehlt: %d", len(lang))
	}
}
