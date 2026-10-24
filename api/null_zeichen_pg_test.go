package api

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/tuertest"
	"bibliothek/sse"
)

// Ein Null-Zeichen kommt an keiner Tür als 500 heraus. PostgreSQL nimmt es in keinem Text an, und
// jede Tür, die es weiterreichte, antwortete 500. Die Abweisung steht deshalb am Eingang
// (nullZeichenMiddleware), und dieser Test hält sie für jede Route: als Admin, je einmal mit dem
// Zeichen in der Abfrage, im Pfad und im Rumpf. Erwartet ist die Abweisung des Eingangs, 400 mit
// meldungNullZeichen. Damit läuft keine Tür; nichts wird geschrieben oder verschickt.
//
// Sieht nicht: einen Rumpf über nullZeichenVorlauf (dort scheitert das Lesen in der Tür,
// TestNullZeichenMiddleware) und hochgeladene Dateien, deren Zeilen der Import prüft.

// nullZeichenAbfrage trägt das Zeichen in den Namen, die Türen für Suche und Filter lesen. Der
// Eingang sieht jeden Namen; die Liste zählt für eine Tür, die ohne den Eingang liefe.
const nullZeichenAbfrage = "q=%00&query=%00&suche=%00&search=%00&barcode=%00&isbn=%00&klasse=%00" +
	"&titel=%00&name=%00&schlagwort=%00&status=%00&filter=%00"

// nullZeichenRumpf ist der Rumpf für jede schreibende Route ohne eigenen Eintrag in
// nullZeichenRuempfe.
const nullZeichenRumpf = `{"text":"\u0000"}`

// nullZeichenRuempfe tragen das Zeichen in einem Feld, das die Tür liest. Ohne den Eingang käme es
// an diesen Türen bis zur Datenbank.
var nullZeichenRuempfe = map[string]string{
	"POST /api/action":                                   `{"query":"A\u0000B"}`,
	"POST /api/buecher/aus-isbn":                         `{"isbn":"978\u00003"}`,
	"PUT /api/buecher/exemplare/{id}/barcode":            `{"barcode":"A\u0000B"}`,
	"PUT /api/buecher/titel/{id}/signatur":               `{"signatur":"A\u0000B"}`,
	"PUT /api/anliegen/{id}/erledigen":                   `{"notiz":"A\u0000B"}`,
	"PUT /api/reservierungen/klassensatz/{id}/erledigen": `{"notiz":"A\u0000B"}`,
}

var nullZeichenPlatzhalter = regexp.MustCompile(`\{[^}]*\}`)

// nullZeichenPfad setzt das Zeichen in jeden Platzhalter und hinter einen Pfad, der mit „/“
// endet und damit alles darunter annimmt.
func nullZeichenPfad(pfad string) (string, bool) {
	ersetzt := nullZeichenPlatzhalter.ReplaceAllString(pfad, "x%00y")
	if strings.HasSuffix(ersetzt, "/") {
		ersetzt += "x%00y"
	}
	return ersetzt, ersetzt != pfad
}

func TestNullZeichen_JedeRouteWeistAmEingangAb(t *testing.T) {
	routen := registrierteRouten(t)
	pool := pgtest.Pool(t)
	// Alle Anfragen kommen von derselben Adresse; die Bremse zählt hier nicht mit.
	t.Setenv("RATE_LIMIT", "100000")
	_, sitzung, anmeldung := tuertest.AdminSitzung(t, pool, "null-zeichen@example.org", "Nora", "Nullzeichen")
	// Routes legt das Verzeichnis uploads an. Die Routendateien sind da schon gelesen.
	t.Chdir(t.TempDir())
	rufe := tuertest.RouterAufruf(NewServer(&db.Verbindung{Pool: pool}, anmeldung, sse.NewBroker(), ServerKonfig{}).Routes(), sitzung)

	muster := make([]string, 0, len(routen))
	for m := range routen {
		muster = append(muster, m)
	}
	sort.Strings(muster)

	gefahren := map[string]int{}
	pruefe := func(t *testing.T, art, methode, url, rumpf string) {
		t.Helper()
		gefahren[art]++
		rec := rufe(t, methode, url, rumpf)
		switch {
		case rec.Code >= http.StatusInternalServerError:
			t.Errorf("%s %s: Status %d — %s", methode, url, rec.Code, rec.Body.String())
		case rec.Code != http.StatusBadRequest || tuertest.Fehlermeldung(t, rec) != meldungNullZeichen:
			t.Errorf("%s %s: Status %d — %s, erwartet die Abweisung des Eingangs", methode, url, rec.Code, rec.Body.String())
		}
	}

	for _, m := range muster {
		methode, pfad, mitMethode := strings.Cut(m, " ")
		if !mitMethode {
			methode, pfad = http.MethodGet, m
		}
		_, url := schreibURL(methode + " " + pfad)
		trenner := "?"
		if strings.Contains(url, "?") {
			trenner = "&"
		}
		t.Run("abfrage/"+m, func(t *testing.T) {
			pruefe(t, "abfrage", methode, url+trenner+nullZeichenAbfrage, "{}")
		})
		if mitNull, ok := nullZeichenPfad(pfad); ok {
			t.Run("pfad/"+m, func(t *testing.T) { pruefe(t, "pfad", methode, mitNull, "{}") })
		}
		if mitMethode && methode != http.MethodGet {
			rumpf, ok := nullZeichenRuempfe[m]
			if !ok {
				rumpf = nullZeichenRumpf
			}
			t.Run("rumpf/"+m, func(t *testing.T) { pruefe(t, "rumpf", methode, url, rumpf) })
		}
	}

	// Liest registrierteRouten weniger, wäre der Test still grün.
	if gefahren["abfrage"] < 200 || gefahren["pfad"] < 50 || gefahren["rumpf"] < 80 {
		t.Errorf("gefahren: %v, erwartet mindestens 200 Abfragen, 50 Pfade und 80 Rümpfe", gefahren)
	}
	for route := range nullZeichenRuempfe {
		if _, ok := routen[route]; !ok {
			t.Errorf("nullZeichenRuempfe nennt %q, aber keine Route heißt so", route)
		}
	}
}
