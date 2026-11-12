package bereitschaft

import (
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"
)

// litteraBefundDauer ist die Zeit, in der die Selbstprüfung das Ergebnis der letzten
// Littera-Übernahme als Befund nennt und die Alarm-Mail es meldet. Danach steht es als erledigt
// da: Eine Übernahme, die die Schule mit einigen fehlenden Datensätzen hingenommen hat, soll
// nicht jeden Tag für immer kritisch melden. Das Protokoll am Server bleibt.
const litteraBefundDauer = 7 * 24 * time.Hour

// pruefeLitteraUebernahme beurteilt das Ergebnis, das der Befehl littera-altbestand nach dem
// schreibenden Lauf ablegt (littera.Ergebnis). Der Befehl läuft am Server und nicht im Programm;
// ohne diesen Befund erführe die Alarm-Mail nichts von einer abgebrochenen oder unvollständigen
// Übernahme.
func pruefeLitteraUebernahme(l Lage) Befund {
	b := Befund{Bereich: "Littera-Übernahme"}
	e := l.LitteraUebernahme
	if e == nil {
		b.Stufe = StufeOK
		b.Befund = "Keine Littera-Übernahme verzeichnet."
		return b
	}
	am := e.Zeitpunkt.In(schulzeit.Zone()).Format("02.01.2006 15:04")
	protokoll := fmt.Sprintf("Das Protokoll %s liegt im Verzeichnis, in dem der Befehl lief.", e.Protokoll)
	if l.Jetzt.Sub(e.Zeitpunkt) > litteraBefundDauer {
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Letzte Übernahme am %s: %s.", am, litteraZusammenfassung(e.Abgebrochen, e.Vollstaendig, e.Fehler, e.Warnungen))
		return b
	}
	switch {
	case e.Abgebrochen:
		b.Stufe = StufeKritisch
		b.Befund = fmt.Sprintf("Die Übernahme am %s ist abgebrochen (%d Fehler, %d Warnungen im Protokoll).", am, e.Fehler, e.Warnungen)
		b.Folge = "Der Bestand aus Littera ist nur zum Teil angekommen."
		b.Abhilfe = protokoll + " Die Ursache steht in der Ausgabe des Befehls und im Protokoll; danach die Übernahme neu starten."
	case !e.Vollstaendig:
		b.Stufe = StufeKritisch
		b.Befund = fmt.Sprintf("Die Übernahme am %s ist unvollständig (%d Fehler, %d Warnungen im Protokoll).", am, e.Fehler, e.Warnungen)
		b.Folge = "Nicht jeder Datensatz aus Littera ist angekommen."
		b.Abhilfe = protokoll + " Es nennt jeden fehlenden Datensatz mit seiner Kennung aus Littera."
	case e.Warnungen > 0:
		b.Stufe = StufeWarnung
		b.Befund = fmt.Sprintf("Die Übernahme am %s ist vollständig, mit %d Warnungen im Protokoll.", am, e.Warnungen)
		b.Folge = "Jeder Datensatz ist angekommen; die Warnungen nennen, was dabei angepasst wurde, etwa eine ungültige ISBN."
		b.Abhilfe = protokoll
	default:
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Die Übernahme am %s ist vollständig, ohne Warnung.", am)
	}
	return b
}

// litteraZusammenfassung nennt das Ergebnis in wenigen Worten, für die Zeit nach dem Befund.
func litteraZusammenfassung(abgebrochen, vollstaendig bool, fehler, warnungen int) string {
	switch {
	case abgebrochen:
		return fmt.Sprintf("abgebrochen, %d Fehler und %d Warnungen", fehler, warnungen)
	case !vollstaendig:
		return fmt.Sprintf("unvollständig, %d Fehler und %d Warnungen", fehler, warnungen)
	}
	return fmt.Sprintf("vollständig, %d Warnungen", warnungen)
}
