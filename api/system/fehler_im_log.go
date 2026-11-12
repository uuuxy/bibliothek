package system

// Die Seite „Fehler im Log" und ihr Teil in der täglichen Mail.
//
// Die Betriebsbereitschaft nennt Zustände: eingerichtet, aber nicht in Betrieb. Das Log nennt
// Ereignisse: was im Betrieb schiefging. Beides hat deshalb eine eigene Seite unter den
// Einstellungen und einen eigenen Teil in der Mail des Wächters.

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/internal/bereitschaft"
	"bibliothek/pkg/logger"
	"bibliothek/pkg/schulzeit"
)

// hoechstzahlMeldungenInDerMail ist die Zahl der Meldungen, die die Mail einzeln nennt, je mit
// ihrer letzten Logzeile; sie begrenzt nur bei einer Flut verschiedener Meldungen die Größe der
// Mail. Alle Fehler zeigt die Seite, alle Zeilen der Befehl im Betriebshandbuch.
const hoechstzahlMeldungenInDerMail = 30

// befehlImHandbuch führt zu jeder einzelnen Zeile: Seite und Mail nennen je Meldung nur die letzte.
const befehlImHandbuch = "Jede Zeile mit Uhrzeit zeigt der Befehl im Betriebshandbuch (docs/DEPLOYMENT.md), Abschnitt „Fehler im Log finden“."

// fehlerImLogAntwort ist die Lage im Log: ein Satz dazu, die Meldungen, Folge und Abhilfe.
type fehlerImLogAntwort struct {
	// Stufe ist ok, warnung (nicht erhoben) oder kritisch, wie bei der Betriebsbereitschaft.
	Stufe string `json:"stufe"`
	// Anzahl zählt die Zeilen aller Meldungen im Zeitraum.
	Anzahl    int           `json:"anzahl"`
	Befund    string        `json:"befund"`
	Meldungen []fehlerZeile `json:"meldungen"`
	Folge     string        `json:"folge,omitempty"`
	Abhilfe   string        `json:"abhilfe,omitempty"`
}

// fehlerZeile ist eine Meldung und darunter, was zu ihr gehört: Anzahl, Zeit und Ursache. So
// zeigt sie die Seite (M3, Liste: Beschriftung und Zusatztext) und so steht sie in der Mail.
type fehlerZeile struct {
	Text   string `json:"text"`
	Zusatz string `json:"zusatz,omitempty"`
	// Zeitraum und Logzeile stehen nur in der Mail: Mit erster und letzter Zeit und der ganzen
	// letzten Zeile lässt sich ein Fehler beheben, ohne das Log am Server zu lesen.
	Zeitraum string `json:"-"`
	Logzeile string `json:"-"`
}

// FehlerImLog nennt die Zeilen der Stufe ERROR im Log der letzten 24 Stunden, je Meldung. Die
// Warnungen nennt nur die Mail, mit dem Schalter alarm_warnungen.
// GET /api/admin/system/fehler-im-log
func (t *Tueren) FehlerImLog() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		jetzt := schulzeit.Jetzt()
		fehler, _ := trenneWarnungen(t.fehlergruppen(jetzt))
		tuer.AntworteJSON(w, http.StatusOK, baueFehlerImLog(fehler, jetzt, 0))
	}
}

// fehlergruppen liefert die Meldungen des Fehlerstands, Fehler und Warnungen; nil heißt, der
// Server zählt nicht.
func (t *Tueren) fehlergruppen(jetzt time.Time) []logger.Fehlergruppe {
	if t.Fehlerstand == nil {
		return nil
	}
	return t.Fehlerstand.Gruppen(jetzt)
}

// trenneWarnungen teilt die Gruppen in Fehler und Warnungen. nil bleibt nil, weil es heißt, der
// Server zählt nicht; eine leere Liste heißt, er zählt und fand nichts.
func trenneWarnungen(gruppen []logger.Fehlergruppe) (fehler, warnungen []logger.Fehlergruppe) {
	if gruppen == nil {
		return nil, nil
	}
	fehler, warnungen = []logger.Fehlergruppe{}, []logger.Fehlergruppe{}
	for _, g := range gruppen {
		if g.Warnung {
			warnungen = append(warnungen, g)
		} else {
			fehler = append(fehler, g)
		}
	}
	return fehler, warnungen
}

// baueFehlerImLog fasst die Meldungen für Seite und Mail, die jüngste zuerst. Schon eine ist
// kritisch: ERROR heißt, jemand sollte nachsehen (docs/STIL.md, Abschnitt 11). hoechstens
// begrenzt die einzeln genannten Meldungen, 0 heißt alle; die übrigen zählt eine letzte Zeile.
func baueFehlerImLog(gruppen []logger.Fehlergruppe, jetzt time.Time, hoechstens int) fehlerImLogAntwort {
	if gruppen == nil {
		return fehlerImLogAntwort{
			Stufe: bereitschaft.StufeWarnung, Meldungen: []fehlerZeile{},
			Befund:  "Nicht erhoben: Der Server zählt seine Fehlermeldungen nicht.",
			Folge:   "Ein Fehler fällt nur auf, wenn jemand das Log liest.",
			Abhilfe: befehlImHandbuch,
		}
	}
	gruppen = slices.Clone(gruppen)
	slices.SortStableFunc(gruppen, func(a, b logger.Fehlergruppe) int { return b.Zuletzt.Compare(a.Zuletzt) })
	a := fehlerImLogAntwort{Stufe: bereitschaft.StufeOK, Meldungen: []fehlerZeile{}}
	for _, g := range gruppen {
		a.Anzahl += g.Anzahl
	}
	stunden := int(logger.FehlerZeitraum.Hours())
	if a.Anzahl == 0 {
		a.Befund = fmt.Sprintf("Keine Fehlermeldung in den letzten %d Stunden.", stunden)
		return a
	}

	a.Stufe = bereitschaft.StufeKritisch
	zuletzt := "zuletzt "
	if a.Anzahl == 1 {
		zuletzt = ""
	}
	a.Befund = fmt.Sprintf("%s in den letzten %d Stunden, %s%s.",
		fehlermeldungen(a.Anzahl), stunden, zuletzt, zeitpunkt(gruppen[0].Zuletzt, jetzt))
	a.Meldungen = fehlerZeilen(gruppen, jetzt, hoechstens, weitereFehlermeldungen)
	a.Folge = "Eine Fehlermeldung heißt: Etwas ist schiefgegangen, und das Programm konnte sich nicht selbst helfen."
	a.Abhilfe = "Die Ursache steht unter jeder Meldung. " + befehlImHandbuch
	return a
}

// baueWarnungenImLog fasst die Warnungen für die Mail, wie baueFehlerImLog die Fehler. Eine Warnung
// heißt: Das Programm hat sich selbst geholfen (docs/STIL.md, Abschnitt 11). Ohne Warnung und
// wenn der Server nicht zählt, bleibt die Anzahl 0, und die Mail nennt keine.
func baueWarnungenImLog(gruppen []logger.Fehlergruppe, jetzt time.Time, hoechstens int) fehlerImLogAntwort {
	a := fehlerImLogAntwort{Stufe: bereitschaft.StufeOK, Meldungen: []fehlerZeile{}}
	gruppen = slices.Clone(gruppen)
	slices.SortStableFunc(gruppen, func(a, b logger.Fehlergruppe) int { return b.Zuletzt.Compare(a.Zuletzt) })
	for _, g := range gruppen {
		a.Anzahl += g.Anzahl
	}
	if a.Anzahl == 0 {
		return a
	}
	zuletzt := "zuletzt "
	if a.Anzahl == 1 {
		zuletzt = ""
	}
	a.Stufe = bereitschaft.StufeWarnung
	a.Befund = fmt.Sprintf("%s in den letzten %d Stunden, %s%s.",
		warnungenText(a.Anzahl), int(logger.FehlerZeitraum.Hours()), zuletzt, zeitpunkt(gruppen[0].Zuletzt, jetzt))
	a.Meldungen = fehlerZeilen(gruppen, jetzt, hoechstens, weitereWarnungen)
	a.Folge = "Eine Warnung heißt: Das Programm hat sich selbst geholfen. Häuft sich eine, steckt oft ein Fehler dahinter."
	a.Abhilfe = "Die Ursache steht unter jeder Meldung. Jede Zeile mit Uhrzeit zeigt der Befehl im Betriebshandbuch " +
		"(docs/DEPLOYMENT.md), Abschnitt „Fehler im Log finden“, mit \"level\":\"WARN\"."
	return a
}

// weitereFehlermeldungen zählt die Fehler, die die Mail nicht einzeln nennt, und zeigt auf die
// Seite, die sie alle nennt.
func weitereFehlermeldungen(n int) string {
	if n == 1 {
		return "1 weitere Fehlermeldung unter System → Einstellungen → Fehler im Log"
	}
	return fmt.Sprintf("%d weitere Fehlermeldungen unter System → Einstellungen → Fehler im Log", n)
}

// weitereWarnungen zählt die Warnungen, die die Mail nicht einzeln nennt; eine Seite für sie gibt
// es nicht, alle zeigt der Befehl im Betriebshandbuch.
func weitereWarnungen(n int) string {
	if n == 1 {
		return "1 weitere Warnung im Log"
	}
	return fmt.Sprintf("%d weitere Warnungen im Log", n)
}

func warnungenText(n int) string {
	if n == 1 {
		return "1 Warnung"
	}
	return fmt.Sprintf("%d Warnungen", n)
}

// fehlerZeilen nennt je Meldung eine Zeile, höchstens hoechstens (0 heißt alle); die Zeilen der
// übrigen zählt eine letzte, deren Text weitere liefert.
func fehlerZeilen(gruppen []logger.Fehlergruppe, jetzt time.Time, hoechstens int, weitere func(n int) string) []fehlerZeile {
	gezeigt := gruppen
	if hoechstens > 0 && len(gruppen) > hoechstens {
		gezeigt = gruppen[:hoechstens]
	}
	zeilen := make([]fehlerZeile, 0, len(gezeigt)+1)
	for _, g := range gezeigt {
		zeilen = append(zeilen, fehlerZeile{
			Text: textDerGruppe(g), Zusatz: zusatzZurMeldung(g, jetzt),
			Zeitraum: zeitraumDerMeldung(g, jetzt), Logzeile: g.Zeile,
		})
	}
	rest := 0
	for _, g := range gruppen[len(gezeigt):] {
		rest += g.Anzahl
	}
	if rest > 0 {
		zeilen = append(zeilen, fehlerZeile{Text: weitere(rest)})
	}
	return zeilen
}

// textDerGruppe ist die Beschriftung einer Gruppe: die Meldung, bei der Zeile zu einer Anfrage mit
// ihrer Route. Die Sammelgruppe zählt die Meldungen ihrer Stufe über der Höchstzahl des
// Fehlerstands; welche es zuletzt war, steht in ihrer Ursache.
func textDerGruppe(g logger.Fehlergruppe) string {
	switch {
	case g.Sammel:
		return "Weitere Meldungen, zusammen gezählt"
	case g.Route != "":
		return g.Meldung + ": " + g.Route
	}
	return g.Meldung
}

// zusatzZurMeldung nennt Anzahl, Zeit und Ursache einer Meldung: „2 × · zuletzt 11:47 · Ursache".
func zusatzZurMeldung(g logger.Fehlergruppe, jetzt time.Time) string {
	zeit := uhrzeit(g.Zuletzt, jetzt)
	if g.Anzahl > 1 {
		zeit = "zuletzt " + zeit
	}
	teile := []string{fmt.Sprintf("%d ×", g.Anzahl), zeit}
	if g.Ursache != "" {
		teile = append(teile, g.Ursache)
	}
	return strings.Join(teile, " · ")
}

// zeitraumDerMeldung nennt für die Mail die erste und die letzte Zeile mit Datum: „erstmals gegen
// 12.11. 08:10, zuletzt 12.11. 09:47:12". Die erste ist auf zehn Minuten genau (Fach des
// Fehlerstands), die letzte auf die Sekunde.
func zeitraumDerMeldung(g logger.Fehlergruppe, jetzt time.Time) string {
	if g.Zuletzt.IsZero() {
		return ""
	}
	zuletzt := g.Zuletzt.In(jetzt.Location()).Format("02.01.2006 15:04:05")
	if g.Anzahl <= 1 || g.Erstmals.IsZero() {
		return "am " + zuletzt
	}
	return "erstmals gegen " + g.Erstmals.In(jetzt.Location()).Format("02.01.2006 15:04") + ", zuletzt " + zuletzt
}

func fehlermeldungen(n int) string {
	if n == 1 {
		return "1 Fehlermeldung"
	}
	return fmt.Sprintf("%d Fehlermeldungen", n)
}

// zeitpunkt nennt eine Zeit für einen Satz: „heute um 11:47", „gestern um 16:10", „am 24.10. um 23:58".
func zeitpunkt(zeit, jetzt time.Time) string {
	zeit = zeit.In(jetzt.Location())
	switch tageZurueck(zeit, jetzt) {
	case 0:
		return "heute um " + zeit.Format("15:04")
	case 1:
		return "gestern um " + zeit.Format("15:04")
	}
	return "am " + zeit.Format("02.01.") + " um " + zeit.Format("15:04")
}

// uhrzeit nennt eine Zeit knapp, für die Zeile unter einer Meldung: „11:47", „gestern 16:10",
// „24.10. 23:58".
func uhrzeit(zeit, jetzt time.Time) string {
	zeit = zeit.In(jetzt.Location())
	switch tageZurueck(zeit, jetzt) {
	case 0:
		return zeit.Format("15:04")
	case 1:
		return "gestern " + zeit.Format("15:04")
	}
	return zeit.Format("02.01. 15:04")
}

// tageZurueck zählt die Kalendertage zwischen zeit und jetzt, im Kalender von jetzt.
func tageZurueck(zeit, jetzt time.Time) int {
	tag := func(t time.Time) time.Time {
		j, m, d := t.Date()
		return time.Date(j, m, d, 0, 0, 0, 0, time.UTC)
	}
	return int(tag(jetzt).Sub(tag(zeit.In(jetzt.Location()))).Hours() / 24)
}
