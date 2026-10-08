package api

// Listen-Gate: Jede Liste in der Antwort einer GET-Route nennt, was sie begrenzt.
//
// Die Bugklasse heißt „Unbegrenzte Listen" (docs/sweeps.md): Eine Route, die eine ganze
// Tabelle ausliefert, fällt auf kleinen Datenmengen nicht auf und kippt im Betrieb. GET
// /api/audit lieferte nach einem Lasttest 72 MB in einer Antwort; der Server brauchte dafür
// 0,6 Sekunden, der Browser scheiterte am Aufbau der Seite. Ein Test je Route fängt das nur
// dort, wo jemand daran gedacht hat.
//
// Dieses Gate hängt am Merkmal. Es ruft jede GET-Route über den echten Router auf, mit dem
// Apparat des PII-Antwort-Gates und dessen Vollständigkeit über die Matrix, und sucht in
// jeder JSON-Antwort die Listen. Jede gefundene Liste braucht einen Eintrag in listenGrenze
// mit der Antwort auf die Frage: Was begrenzt sie — eine Obergrenze in der Abfrage, ein
// Elternteil, eine feste Menge? Der Bestand vom Tag des Baus steht ohne Antwort in
// listenUnbefragt und wird nur kürzer.
//
// Kein Urteil: Ob die genannte Grenze greift, prüft je Liste ihr eigener Test, gegen mehr
// Zeilen als die Grenze. Sieht nicht: Listen in PDF-, CSV- und Bildantworten; Routen, die das
// PII-Gate mit Begründung auslässt oder die in der gesäten Welt mit einem Fehlerstatus
// antworten; eine Liste, die leer als null oder gar nicht in der Antwort steht; Listen in den
// Zeilen einer Liste und tiefer als zwei Ebenen unter der Wurzel; POST-Routen.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/sse"
)

// listenGrenze: je Liste (Route und Ort in der Antwort, „$" ist die Wurzel) die Antwort.
var listenGrenze = map[string]string{
	"GET /api/audit $":            "LIMIT 1000, die jüngsten zuerst (auditLogMaxZeilen, api/audit_handler.go)",
	"GET /api/admin/auditlog $":   "LIMIT 1000, die jüngsten zuerst (api/audit_logs_handler.go)",
	"GET /api/schueler/deleted $": "LIMIT 1000, die jüngsten Löschungen zuerst (papierkorbLimit, api/student_deleted.go)",
	"GET /api/books $.data": "Kappung bei 50000 Titeln mit Warnung im Log (listBooksSicherheitsLimit, " +
		"inventur/datenbank_buecher_leser.go)",
	"GET /api/signaturen $": "keine Obergrenze in der Abfrage; eine Zeile je Regaladresse (GROUP BY über die " +
		"Titel), die Liste wächst mit den Regalen, nicht mit den Büchern (api/signaturen_handler.go)",
}

// listenUnbefragt: der Bestand ohne Antwort. Eine befragte Liste wandert mit ihrer Antwort
// nach listenGrenze; eine neue Liste gehört nicht hierher.
var listenUnbefragt = []string{
	"GET /api/abgaenger $.abgaenger",
	"GET /api/action/buchbarcodes $.barcodes",
	"GET /api/action/nachbuch-meldungen $",
	"GET /api/admin/permissions $",
	"GET /api/admin/system/betriebsbereitschaft $.befunde",
	"GET /api/anliegen/eigene $",
	"GET /api/anliegen/offen $",
	"GET /api/audit/tresen-auskunft $.ereignisse",
	"GET /api/audit/tresen-auskunft $.exemplare",
	"GET /api/auth/me $.permissions",
	"GET /api/benutzer $",
	"GET /api/bescheide $",
	"GET /api/bescheide/ausstehend $",
	"GET /api/bestand/abgangsbuch $.abschnitte",
	"GET /api/bestand/zugangsbuch $.abschnitte",
	"GET /api/bestellhistorie $",
	"GET /api/bestellhistorie/uebersicht $.nach_mittel",
	"GET /api/bestellungen $",
	"GET /api/bestellungen/zulauf $",
	"GET /api/books/{id} $.schlagworte",
	"GET /api/buecher/titel/suche $.books",
	"GET /api/buecher/titel/{id}/auflagen $.auflagen",
	"GET /api/buecher/titel/{id}/ausleiher $",
	"GET /api/buecher/titel/{id}/exemplare $",
	"GET /api/buecher/titel/{id}/historie $",
	"GET /api/buecher/titel/{id}/schlagworte $.schlagworte",
	"GET /api/class-books $.data",
	"GET /api/dashboard/summary $.overdue_buckets",
	"GET /api/einstellungen $.sommerferien_programm",
	"GET /api/exemplare/etiketten-offen $",
	"GET /api/exemplare/standorte $",
	"GET /api/faecher $",
	"GET /api/geraete $.data",
	"GET /api/inventur/abgeschlossen $",
	"GET /api/inventur/sessions $",
	"GET /api/jahrgaenge $",
	"GET /api/klassen $",
	"GET /api/klassen-mapping $",
	"GET /api/lieferanten $",
	"GET /api/lmf-plan/{art} $.ausgelassen",
	"GET /api/lmf-plan/{art} $.ausgelassen_regel",
	"GET /api/lmf-plan/{art} $.eingangsjahrgaenge",
	"GET /api/lmf-plan/{art} $.klassen",
	"GET /api/lmf-plan/{art} $.plan.freie_tage",
	"GET /api/lmf-plan/{art} $.zeilen",
	"GET /api/lmf-termine $.eingangsjahrgaenge",
	"GET /api/lmf-termine $.termine",
	"GET /api/mahnwesen $.klassen",
	"GET /api/mail-templates $",
	"GET /api/monitor/slides $.beliebt",
	"GET /api/monitor/slides $.neu_eingetroffen",
	"GET /api/portal/klassensaetze $.data",
	"GET /api/portal/lernmittel $.faecher",
	"GET /api/portal/lernmittel $.titel",
	"GET /api/public/opac/suche $",
	"GET /api/reservierungen/klassensatz $",
	"GET /api/reservierungen/klassensatz/eigene $",
	"GET /api/reservierungen/klassensatz/katalog $",
	"GET /api/reservierungen/klassensatz/katalog/filter $",
	"GET /api/reservierungen/klassensatz/offen $",
	"GET /api/schlagworte $",
	"GET /api/schlagworte/pflege $.zeilen",
	"GET /api/schueler $",
	"GET /api/schueler/{id} $.entliehene_buecher",
	"GET /api/schueler/{id}/bescheid-vorschlag $.ausleihen",
	"GET /api/schueler/{id}/bescheid-vorschlag $.fehlende_angaben",
	"GET /api/schueler/{id}/bescheid-vorschlag $.positionen",
	"GET /api/schueler/{id}/bescheide $",
	"GET /api/schueler/{id}/dsgvo-auskunft $.ausleihhistorie",
	"GET /api/schueler/{id}/dsgvo-auskunft $.fruehere_zugangskonten",
	"GET /api/schueler/{id}/dsgvo-auskunft $.nachbuch_meldungen",
	"GET /api/schueler/{id}/dsgvo-auskunft $.protokolleintraege",
	"GET /api/schueler/{id}/dsgvo-auskunft $.schadensersatz_bescheide",
	"GET /api/schueler/{id}/dsgvo-auskunft $.schadensfaelle",
	"GET /api/schueler/{id}/dsgvo-auskunft $.verarbeitungsangaben.zwecke",
	"GET /api/schueler/{id}/dsgvo-auskunft $.verwaltungsprotokolle",
	"GET /api/schueler/{id}/dsgvo-auskunft $.vormerkungen",
	"GET /api/schueler/{id}/schadensfaelle $.data",
	"GET /api/schueler/{id}/zusammenfuehren-kandidaten $",
	"GET /api/search $.books",
	"GET /api/search $.students",
	"GET /api/signaturen/buecher $.buecher",
	"GET /api/statistiken $.monats_trend",
	"GET /api/statistiken $.popular_titles",
	"GET /api/statistiken $.shelf_warmers",
	"GET /api/systematics $",
	"GET /api/vormerkungen $",
}

// listenUnbefragtHoechstens hält die Länge von listenUnbefragt fest: Wüchse die Liste, hätte
// jemand eine neue Liste ohne Antwort eingetragen.
const listenUnbefragtHoechstens = 87

// listenPfade nennt die Listen einer JSON-Antwort: die Wurzel selbst und Felder bis zwei
// Ebenen darunter. In die Zeilen einer Liste steigt sie nicht hinab.
func listenPfade(rumpf []byte) []string {
	var wert any
	if err := json.Unmarshal(rumpf, &wert); err != nil {
		return nil
	}
	var aus []string
	sammleListen(wert, "$", 0, &aus)
	sort.Strings(aus)
	return aus
}

func sammleListen(wert any, pfad string, tiefe int, aus *[]string) {
	switch w := wert.(type) {
	case []any:
		*aus = append(*aus, pfad)
	case map[string]any:
		if tiefe >= 2 {
			return
		}
		for feld, inhalt := range w {
			sammleListen(inhalt, pfad+"."+feld, tiefe+1, aus)
		}
	}
}

func TestListenPfade_ErkenntDieFormen(t *testing.T) {
	faelle := []struct {
		rumpf string
		soll  []string
	}{
		{`[{"id":1}]`, []string{"$"}},
		{`[]`, []string{"$"}},
		{`{"data":[],"gesamt":0}`, []string{"$.data"}},
		{`{"plan":{"freie_tage":[]},"zeilen":[{"klassen":["5a"]}]}`, []string{"$.plan.freie_tage", "$.zeilen"}},
		{`{"a":{"b":{"c":[]}}}`, nil},
		{`{"name":"x","anzahl":3}`, nil},
		{`null`, nil},
		{`kein json`, nil},
	}
	for _, f := range faelle {
		if ist := listenPfade([]byte(f.rumpf)); !reflect.DeepEqual(ist, f.soll) {
			t.Errorf("listenPfade(%s) = %v, erwartet %v", f.rumpf, ist, f.soll)
		}
	}
}

func TestListenNennenIhreGrenze(t *testing.T) {
	pool := pgTestPool(t)
	t.Setenv("RATE_LIMIT", "100000")
	// Ein definierter Anfang: Was ein früherer Test liegen ließ, änderte sonst, welche Felder
	// einer Antwort gefüllt sind.
	resetBestandsdaten(t, pool)
	if err := (&db.Database{Pool: pool}).InitPermissions(context.Background()); err != nil {
		t.Fatalf("InitPermissions: %v", err)
	}
	authenticator, err := auth.NewAuthenticator(
		"listen-gate-testgeheimnis-mind-32-bytes-lang!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	srv := NewServer(&db.Database{Pool: pool}, authenticator, sse.NewBroker(), false)
	srv.Uhr = saisonUhr
	router := srv.Routes()

	welt := baueKanarienWelt(t, pool, authenticator)
	aufrufe := bauePIIAufrufe(welt)
	matrix := leseMatrix(t)

	routen := make([]string, 0, len(aufrufe))
	for route := range aufrufe {
		routen = append(routen, route)
	}
	sort.Strings(routen)

	gefunden := map[string]bool{}
	for _, route := range routen {
		permission, mitSitzung := rechtFuerZeile(matrix[route].Recht)
		setzeGenauEinRecht(t, pool, permission)
		req := httptest.NewRequest(http.MethodGet, aufrufe[route].URL, nil)
		if mitSitzung {
			req.AddCookie(&http.Cookie{Name: "session_token", Value: welt.sessionToken})
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code >= 400 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			continue
		}
		for _, pfad := range listenPfade(rec.Body.Bytes()) {
			gefunden[route+" "+pfad] = true
		}
	}
	// Nicht-leer-Garantie: Findet der Lauf kaum Listen, erreicht er die Routen nicht mehr.
	if len(gefunden) < 40 {
		t.Fatalf("nur %d Listen gefunden — die Aufrufe erreichen die Routen nicht mehr, dieses Gate wäre still grün", len(gefunden))
	}

	unbefragt := map[string]bool{}
	for _, liste := range listenUnbefragt {
		unbefragt[liste] = true
	}
	var neu, weg, doppelt []string
	for liste := range gefunden {
		if strings.TrimSpace(listenGrenze[liste]) == "" && !unbefragt[liste] {
			neu = append(neu, liste)
		}
	}
	for liste := range listenGrenze {
		if !gefunden[liste] {
			weg = append(weg, liste)
		}
		if unbefragt[liste] {
			doppelt = append(doppelt, liste)
		}
	}
	for _, liste := range listenUnbefragt {
		if !gefunden[liste] {
			weg = append(weg, liste)
		}
	}
	sort.Strings(neu)
	sort.Strings(weg)
	sort.Strings(doppelt)
	if len(neu) > 0 {
		t.Errorf("Liste ohne Antwort:\n  %s\n\n"+
			"Was begrenzt sie? Eine Tabelle, die mit der Zeit wächst, braucht eine Obergrenze in der "+
			"Abfrage, einen Index auf der Sortierspalte und die Ansage der Kappung in der Oberfläche; "+
			"der Test dazu läuft gegen mehr Zeilen als die Grenze. Die Antwort gehört nach "+
			"listenGrenze. War die Liste schon da und erscheint erst jetzt, weil die gesäte Welt "+
			"sie füllt, gilt dieselbe Frage.", strings.Join(neu, "\n  "))
	}
	if len(weg) > 0 {
		t.Errorf("Eintrag ohne Liste in der Antwort:\n  %s\n\nEintrag austragen oder den Ort berichtigen.",
			strings.Join(weg, "\n  "))
	}
	if len(doppelt) > 0 {
		t.Errorf("Steht in listenGrenze und in listenUnbefragt:\n  %s\n\nAus listenUnbefragt austragen.",
			strings.Join(doppelt, "\n  "))
	}
	if n := len(listenUnbefragt); n > listenUnbefragtHoechstens {
		t.Errorf("listenUnbefragt ist auf %d gewachsen (festgehalten: %d). Eine neue Liste gehört mit "+
			"ihrer Antwort nach listenGrenze.", n, listenUnbefragtHoechstens)
	} else if n < listenUnbefragtHoechstens {
		t.Errorf("listenUnbefragt ist auf %d geschrumpft — listenUnbefragtHoechstens auf %d setzen.", n, n)
	}
}
