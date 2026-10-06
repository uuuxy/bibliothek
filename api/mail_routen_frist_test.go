package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Gate: Ruft die Oberfläche eine Route, hinter der der Server in der Anfrage Mail verschickt,
// trägt der Aufruf eine eigene Frist (timeoutMs).
//
// Die Oberfläche gibt einer ändernden Anfrage 10 s (apiFetch.js). Der Versand hat eigene
// Fristen (mailservice/versand.go: Verbindung und Sitzung), und der Sammelversand öffnet je
// Klasse eine Sitzung. Gibt die Oberfläche zuerst auf, arbeitet der Server weiter, niemand
// sieht seine Antwort, und der zweite Versuch trifft eine Arbeit, die schon geschehen ist:
// Mahnlisten gehen doppelt hinaus, eine Bestellung meldet „war bereits erfasst", obwohl ihre
// Mail nie ankam.
//
// Gemessen wird am Merkmal: jede Funktion in api/, die SendEmail oder den Versand aus
// mailservice erreicht, und jede Route, deren Anmeldung eine solche Funktion nennt. Blind für
// einen Versand über ein anderes Paket, für Routen außerhalb von mux.Handle in routes*.go
// und router.go, für eine Adresse, die die Oberfläche aus Teilen zusammensetzt, und dafür, ob
// die genannte Frist lang genug ist (das hält TestMailFristen_DeckenDenVersandAb für die zwei
// Konstanten).
func TestMailRouten_AufruferTragenEigeneFrist(t *testing.T) {
	routen := mailRouten(t, funktionenDieMailVerschicken(t))
	if len(routen) < 5 {
		t.Fatalf("nur %d Routen mit Mailversand gefunden (%v): Der Detektor sieht den Versand nicht mehr", len(routen), routen)
	}
	quellen := oberflaechenQuellen(t)
	if len(quellen) < 200 {
		t.Fatalf("nur %d Dateien der Oberfläche gelesen: Der Sammler läuft ins Leere", len(quellen))
	}
	for _, route := range routen {
		aufrufe := aufrufeDerRoute(quellen, route)
		if len(aufrufe) == 0 {
			t.Errorf("%s %s verschickt Mail, in der Oberfläche steht kein Aufruf mit dieser Adresse: "+
				"Entweder setzt sie die Adresse aus Teilen zusammen (dann sieht dieses Gate sie nicht), "+
				"oder die Route hat keinen Aufrufer.", route.methode, route.pfad)
		}
		for _, a := range aufrufe {
			if !strings.Contains(a.text, "timeoutMs") {
				t.Errorf("%s: Der Aufruf von %s %s trägt keine eigene Frist. Der Server verschickt dort "+
					"in der Anfrage Mail; mit der Vorgabe von 10 s gibt die Oberfläche vor ihm auf. "+
					"timeoutMs: FRIST_MAILVERSAND_MS oder FRIST_SAMMELVERSAND_MS (apiFetch.js) ergänzen.",
					a.datei, route.methode, route.pfad)
			}
		}
	}
}

// TestMailFristen_DeckenDenVersandAb hält die zwei Fristen der Oberfläche an die des Servers:
// Ein einzelner Versand dauert höchstens Verbindungs- plus Sitzungsfrist, der Sammelversand
// hat die Frist der lang laufenden Vorgänge.
func TestMailFristen_DeckenDenVersandAb(t *testing.T) {
	einzel := zahlAus(t, "../frontend/src/lib/apiFetch.js", `FRIST_MAILVERSAND_MS = (\d+)`)
	sammel := zahlAus(t, "../frontend/src/lib/apiFetch.js", `FRIST_SAMMELVERSAND_MS = (\d+)`)
	verbindung := zahlAus(t, "../mailservice/versand.go", `smtpVerbindungsTimeout = (\d+) \* time\.Second`)
	sitzung := zahlAus(t, "../mailservice/versand.go", `smtpSitzungsFrist = (\d+) \* time\.Second`)

	if hoechstens := (verbindung + sitzung) * 1000; einzel <= hoechstens {
		t.Errorf("FRIST_MAILVERSAND_MS ist %d ms, ein Versand darf %d ms dauern: Die Oberfläche gäbe "+
			"vor dem Server auf", einzel, hoechstens)
	}
	if lang := int(LangLaufendeFrist / time.Millisecond); sammel != lang {
		t.Errorf("FRIST_SAMMELVERSAND_MS ist %d ms, die Frist der lang laufenden Vorgänge %d ms", sammel, lang)
	}
}

func TestMailRouten_Selbstprobe(t *testing.T) {
	quellen := map[string]string{
		"ohne.js":     "await apiFetch('/api/mail/send-bulk-overdue', {\n\tmethod: 'POST',\n\tbody: JSON.stringify(x)\n});",
		"mit.js":      "await apiFetch('/api/mail/send-bulk-overdue', { method: 'POST', timeoutMs: FRIST_SAMMELVERSAND_MS });",
		"lesen.js":    "const liste = await apiGet('/api/bestellungen');",
		"post.js":     "await apiPost('/api/bestellungen', { a: f(1) }, { timeoutMs: FRIST_MAILVERSAND_MS });",
		"kurz.js":     "await apiPost('/api/bestellungen', { a: f(1) });",
		"platz.js":    "const res = await apiFetch(`/api/anliegen/${id}/erledigen`, {\n\tmethod: 'PUT'\n});",
		"klient.js":   "await apiClient.put(`/api/anliegen/${a.id}/erledigen`, { notiz }, { timeoutMs: 1 });",
		"andere.js":   "await apiPost('/api/bestellungen/suche', { q });",
		"erwaehnt.js": entferneKommentare("// POST /api/bestellungen verschickt die Mail\nconst x = 1;"),
	}
	faelle := []struct {
		route mailRoute
		ohne  []string
	}{
		{mailRoute{"POST", "/api/mail/send-bulk-overdue"}, []string{"ohne.js"}},
		{mailRoute{"POST", "/api/bestellungen"}, []string{"kurz.js"}},
		{mailRoute{"PUT", "/api/anliegen/{id}/erledigen"}, []string{"platz.js"}},
	}
	for _, fall := range faelle {
		var ohne []string
		for _, a := range aufrufeDerRoute(quellen, fall.route) {
			if !strings.Contains(a.text, "timeoutMs") {
				ohne = append(ohne, a.datei)
			}
		}
		if strings.Join(ohne, ",") != strings.Join(fall.ohne, ",") {
			t.Errorf("%s %s: ohne Frist gemeldet %v, erwartet %v", fall.route.methode, fall.route.pfad, ohne, fall.ohne)
		}
	}
	if n := len(aufrufeDerRoute(quellen, mailRoute{"POST", "/api/bestellungen"})); n != 2 {
		t.Errorf("POST /api/bestellungen: %d Aufrufe gefunden, erwartet 2 (nicht das Lesen, nicht /suche, nicht die Erwähnung)", n)
	}
}

type mailRoute struct{ methode, pfad string }

type routenAufruf struct{ datei, text string }

// funktionenDieMailVerschicken nennt die Funktionen des Pakets, die den Versand erreichen:
// unmittelbar oder über eine Funktion, die es tut.
func funktionenDieMailVerschicken(t *testing.T) map[string]bool {
	t.Helper()
	pfade, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("api/ nicht lesbar: %v", err)
	}
	nennt := map[string]map[string]bool{}
	sendet := map[string]bool{}
	dateisatz := token.NewFileSet()
	for _, pfad := range pfade {
		if strings.HasSuffix(pfad, "_test.go") {
			continue
		}
		datei, err := parser.ParseFile(dateisatz, pfad, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", pfad, err)
		}
		for _, erklaerung := range datei.Decls {
			funktion, ok := erklaerung.(*ast.FuncDecl)
			if !ok || funktion.Body == nil {
				continue
			}
			name := funktion.Name.Name
			nennt[name] = map[string]bool{}
			ast.Inspect(funktion.Body, func(knoten ast.Node) bool {
				switch k := knoten.(type) {
				case *ast.SelectorExpr:
					if x, ok := k.X.(*ast.Ident); ok && x.Name == "mailservice" &&
						(k.Sel.Name == "SendTestMail" || k.Sel.Name == "VersendeUeberSMTP") {
						sendet[name] = true
					}
					nennt[name][k.Sel.Name] = true
				case *ast.Ident:
					if k.Name == "SendEmail" {
						sendet[name] = true
					}
					nennt[name][k.Name] = true
				}
				return true
			})
		}
	}
	if len(sendet) < 4 {
		t.Fatalf("nur %d Funktionen mit unmittelbarem Versand gefunden: SendEmail oder mailservice heißen anders", len(sendet))
	}
	for geaendert := true; geaendert; {
		geaendert = false
		for name, genannte := range nennt {
			if sendet[name] {
				continue
			}
			for genannt := range genannte {
				if sendet[genannt] {
					sendet[name], geaendert = true, true
					break
				}
			}
		}
	}
	return sendet
}

var (
	routenAdresse  = regexp.MustCompile(`^mux\.Handle(?:Func)?\("(GET|POST|PUT|PATCH|DELETE) ([^"]+)"`)
	aufrufName     = regexp.MustCompile(`\b([A-Za-z_]\w*)\(`)
	methodeImRumpf = regexp.MustCompile(`method:\s*['"]([A-Za-z]+)['"]`)
	platzhalter    = regexp.MustCompile(`\\\{[^}]+\\\}`)
)

// mailRouten liest die Anmeldungen der Routen (registrierungsAusdruecke, wie das Rechte-Gate)
// und behält die, die eine sendende Funktion nennen.
func mailRouten(t *testing.T, sendet map[string]bool) []mailRoute {
	t.Helper()
	dateien, err := filepath.Glob("routes_*.go")
	if err != nil {
		t.Fatalf("glob routes_*.go: %v", err)
	}
	var routen []mailRoute
	gelesen := 0
	for _, datei := range append(dateien, "router.go") {
		inhalt, err := os.ReadFile(datei)
		if err != nil {
			t.Fatalf("%s: %v", datei, err)
		}
		for _, anmeldung := range registrierungsAusdruecke(string(inhalt), "mux.") {
			adresse := routenAdresse.FindStringSubmatch(anmeldung)
			if adresse == nil {
				continue
			}
			gelesen++
			for _, name := range aufrufName.FindAllStringSubmatch(anmeldung, -1) {
				if sendet[name[1]] {
					routen = append(routen, mailRoute{adresse[1], adresse[2]})
					break
				}
			}
		}
	}
	if gelesen < 150 {
		t.Fatalf("nur %d Routen gelesen: Das Muster trifft die Anmeldungen nicht mehr", gelesen)
	}
	return routen
}

// oberflaechenQuellen liest die Quelldateien der Oberfläche ohne Tests und ohne Kommentare.
func oberflaechenQuellen(t *testing.T) map[string]string {
	t.Helper()
	quellen := map[string]string{}
	err := filepath.WalkDir("../frontend/src", func(pfad string, eintrag fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if eintrag.IsDir() {
			if eintrag.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		name := eintrag.Name()
		quelltext := strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".svelte")
		if !quelltext || strings.HasSuffix(name, ".test.js") {
			return nil
		}
		roh, err := os.ReadFile(pfad)
		if err != nil {
			return err
		}
		quellen[pfad] = entferneKommentare(string(roh))
		return nil
	})
	if err != nil {
		t.Fatalf("Oberfläche nicht lesbar: %v", err)
	}
	return quellen
}

var kommentare = regexp.MustCompile(`(?s)/\*.*?\*/|<!--.*?-->|(?m:(^|\s)//[^\n]*)`)

func entferneKommentare(quelle string) string { return kommentare.ReplaceAllString(quelle, "") }

// aufrufeDerRoute findet die Aufrufe, deren erstes Argument die Adresse der Route ist und
// deren Methode zu ihr passt, je mit dem Text des ganzen Aufrufs.
func aufrufeDerRoute(quellen map[string]string, route mailRoute) []routenAufruf {
	adresse := platzhalter.ReplaceAllString(regexp.QuoteMeta(route.pfad), `\$\{[^}]+\}`)
	muster := regexp.MustCompile("\\(\\s*['\"`]" + adresse + "(?:['\"`]|\\?)")
	dateien := make([]string, 0, len(quellen))
	for datei := range quellen {
		dateien = append(dateien, datei)
	}
	sort.Strings(dateien)
	var aufrufe []routenAufruf
	for _, datei := range dateien {
		quelle := quellen[datei]
		for _, stelle := range muster.FindAllStringIndex(quelle, -1) {
			text := quelle[stelle[0]:aufrufEnde(quelle, stelle[0])]
			if aufrufMethode(quelle[:stelle[0]], text) == route.methode {
				aufrufe = append(aufrufe, routenAufruf{datei, text})
			}
		}
	}
	return aufrufe
}

// aufrufMethode liest die Methode am Namen des Aufrufers (apiPost, apiClient.put) oder an
// method im Aufruf; ohne beides ist es ein GET.
func aufrufMethode(davor, aufruf string) string {
	name := strings.ToLower(davor[strings.LastIndexAny(davor, " \t\n=(!")+1:])
	for _, methode := range []string{"POST", "PUT", "PATCH", "DELETE", "GET"} {
		if strings.HasSuffix(name, strings.ToLower(methode)) {
			return methode
		}
	}
	if m := methodeImRumpf.FindStringSubmatch(aufruf); m != nil {
		return strings.ToUpper(m[1])
	}
	return "GET"
}

// aufrufEnde liefert die Stelle hinter der Klammer, die die bei auf öffnende schließt, im
// JavaScript der Oberfläche: Klammern in '…', "…" und `…` zählen nicht.
func aufrufEnde(quelle string, auf int) int {
	tiefe := 0
	var zeichenkette byte
	for i := auf; i < len(quelle); i++ {
		c := quelle[i]
		switch {
		case zeichenkette != 0 && c == '\\':
			i++
		case zeichenkette != 0:
			if c == zeichenkette {
				zeichenkette = 0
			}
		case c == '\'' || c == '"' || c == '`':
			zeichenkette = c
		case c == '(':
			tiefe++
		case c == ')':
			if tiefe--; tiefe == 0 {
				return i + 1
			}
		}
	}
	return len(quelle)
}

func zahlAus(t *testing.T, datei, muster string) int {
	t.Helper()
	roh, err := os.ReadFile(datei)
	if err != nil {
		t.Fatalf("%s: %v", datei, err)
	}
	treffer := regexp.MustCompile(muster).FindSubmatch(roh)
	if treffer == nil {
		t.Fatalf("%s: %s nicht gefunden — die Frist heißt anders oder ist anders geschrieben", datei, muster)
	}
	zahl, err := strconv.Atoi(string(treffer[1]))
	if err != nil {
		t.Fatalf("%s: %v", datei, err)
	}
	return zahl
}
