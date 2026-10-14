package bereitschaft

import (
	"fmt"
	"strings"
)

// pruefeLoeschroutinen fragt, ob die nächtlichen Löschroutinen tun, was sie sollen. Gemessen
// wird der Rückstand, nicht ein Log: Ein Job, der still scheitert, schreibt keine Zeile, die
// jemand liest, aber er hinterlässt Zeilen, die seit mehr als einem Tag weg sein müssten.
//
// Die Zahlen kommen aus BetriebszustandAbfragen.ZaehleLoeschRueckstand mit denselben
// SQL-Ausdrücken wie die Jobs (repository/loeschfristen.go), damit der Wächter nicht
// beruhigt, während der Job schläft.
func pruefeLoeschroutinen(l Lage) Befund {
	b := Befund{Bereich: "DSGVO-Löschroutinen"}
	if l.LoeschRueckstand == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Der Zustand der Löschroutinen konnte nicht erhoben werden."
		b.Folge = "Unklar, ob die nächtlichen Löschroutinen laufen."
		b.Abhilfe = "Datenbankverbindung prüfen; Container-Logs nach 'Scheduler GDPR', 'Scheduler Lesehistorie', 'Scheduler Anliegen' und 'Audit-Aufbewahrung' durchsuchen."
		return b
	}

	var ueberfaellig, abgeschaltet []string
	for _, r := range l.LoeschRueckstand {
		switch {
		case r.Aus:
			abgeschaltet = append(abgeschaltet, r.Routine)
		case r.Zeilen > 0:
			ueberfaellig = append(ueberfaellig, fmt.Sprintf("%s: %d (Frist %s)", r.Routine, r.Zeilen, r.Frist))
		}
	}

	if len(ueberfaellig) > 0 {
		b.Stufe = StufeKritisch
		b.Befund = "Überfällig und nicht gelöscht — " + strings.Join(ueberfaellig, "; ") + "."
		b.Folge = "Mindestens eine nächtliche Löschroutine läuft nicht oder scheitert still — Personendaten bleiben über die Frist hinaus stehen (Art. 5 (1) e DSGVO)."
		b.Abhilfe = "Container-Logs der letzten Nacht nach der genannten Routine durchsuchen (der Fehlertext steht dort); die Jobs wiederholen sich jede Nacht und heilen den Rückstand selbst, sobald die Ursache weg ist."
		return b
	}

	if len(abgeschaltet) > 0 {
		b.Stufe = StufeWarnung
		b.Befund = "Frist steht auf 0 = aus: " + strings.Join(abgeschaltet, ", ") + "."
		b.Folge = "Diese Daten werden nicht befristet — das Verzeichnis der Verarbeitungstätigkeiten und der Datenschutzhinweis müssen das so ausweisen."
		b.Abhilfe = "Einstellungen → Datenschutz & Sitzung: Frist in Tagen setzen, oder die Entscheidung im VVT festhalten."
		return b
	}

	b.Stufe = StufeOK
	b.Befund = fmt.Sprintf("%d Löschroutinen geprüft, kein überfälliger Datensatz.", len(l.LoeschRueckstand))
	return b
}

// EhemaligeOffenSeitTagen ist die Schwelle des Wächters: ein Jahr. Kürzer meldete er
// jede Herbst-Mahnung, länger ließe die Akte Jahre stehen.
const EhemaligeOffenSeitTagen = 365

// pruefeEhemaligeOffen meldet Ehemalige, die mit Name und Anschrift stehen bleiben, weil ein
// Vorgang nie geschlossen wurde. Was zu tun ist (Verlust buchen, Forderung stellen oder
// stornieren), entscheidet ein Mensch; danach greifen Karenz und Löschung von selbst.
func pruefeEhemaligeOffen(l Lage) Befund {
	b := Befund{Bereich: "Ehemalige mit offenen Vorgängen"}
	if l.EhemaligeMitOffenenVorgaengen == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Nicht erhoben — die Zählung der Ehemaligen mit offenen Vorgängen ist fehlgeschlagen."
		b.Folge = "Unklar, ob Datensätze Weggegangener über die Löschfrist hinaus stehen bleiben."
		b.Abhilfe = "Datenbankverbindung prüfen; die Zählung läuft bei jedem Aufruf dieser Seite."
		return b
	}
	n := *l.EhemaligeMitOffenenVorgaengen
	if n > 0 {
		b.Stufe = StufeWarnung
		b.Befund = fmt.Sprintf("%d Ehemalige haben seit mehr als einem Jahr ein offenes Buch oder eine unbezahlte Forderung.", n)
		b.Folge = "Solange der Vorgang offen ist, bleiben Name und Anschrift gespeichert — Karenz, Anonymisierung und Löschung greifen nicht (Art. 5 (1) e DSGVO)."
		b.Abhilfe = "Leserdatei → Ehemalige / Archiv → Akte öffnen: das Buch als Verlust melden (schließt die Ausleihe, die Forderung entsteht), dann die Forderung als bezahlt oder storniert buchen. Danach anonymisiert und löscht das System von selbst."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = "Kein Ehemaliger mit einem seit über einem Jahr offenen Vorgang."
	return b
}

// NachbuchOffenSeitTagen ist die Schwelle des Wächters: zwei Wochen. Eine Meldung, die so
// lange niemand quittiert, hat niemand angesehen.
const NachbuchOffenSeitTagen = 14

// pruefeNachbuchMeldungenOffen meldet Abweichungen zwischen Offline-Scan und Wirklichkeit
// (umgebuchte Bücher, abgewiesene Ausleihen, veraltete Scans), die niemand quittiert hat.
// Was daraus folgt, etwa ein Buch suchen oder ein Kind ansprechen, entscheidet ein Mensch.
func pruefeNachbuchMeldungenOffen(l Lage) Befund {
	b := Befund{Bereich: "Nachbuch-Meldungen der Theke"}
	if l.NachbuchMeldungenOffen == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Nicht erhoben — die Zählung der offenen Nachbuch-Meldungen ist fehlgeschlagen."
		b.Folge = "Unklar, ob Abweichungen aus dem Offline-Betrieb unbesehen liegen bleiben."
		b.Abhilfe = "Datenbankverbindung prüfen; die Zählung läuft bei jedem Aufruf dieser Seite."
		return b
	}
	n := *l.NachbuchMeldungenOffen
	if n > 0 {
		b.Stufe = StufeWarnung
		b.Befund = fmt.Sprintf("%d Nachbuch-Meldungen sind seit mehr als zwei Wochen nicht quittiert.", n)
		b.Folge = "Ein umgebuchtes oder abgewiesenes Buch aus dem Offline-Betrieb hat niemand angesehen; die Beteiligten stehen mit Namen darin, ohne Frist."
		b.Abhilfe = "Theke → Meldungen: jede Zeile prüfen und quittieren. Quittierte Meldungen löscht das System nach der Lesehistorie-Frist von selbst."
		return b
	}
	b.Stufe = StufeOK
	b.Befund = "Keine Nachbuch-Meldung liegt länger als zwei Wochen unquittiert."
	return b
}
