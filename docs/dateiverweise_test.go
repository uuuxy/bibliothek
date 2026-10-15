package docs

import (
	"go/scanner"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Dateiverweise: Wo ein Dokument, ein Kommentar, die Oberfläche oder ein Skript eine Datei beim
// Pfad nennt, gibt es die Datei. Zieht eine Datei um oder fällt sie weg, zeigt der Verweis
// sonst weiter auf den alten Ort, und wer ihm folgt, findet nichts.
//
// Ein Verweis ist ein Pfad, der mit einem Ordner der Wurzel beginnt und auf eine Dateiendung
// endet (api/router.go, docs/OFFEN.md); die Oberfläche und die Skripte nennen Pfade auch ab
// frontend/. In den Dokumenten zählen dazu die Ziele der Links. Gemessen wird an den Dateien,
// die zum Repository gehören (git ls-files), nicht am Arbeitsplatz: Dort liegen gebaute
// Dateien, die es im Prüflauf nicht gibt.
//
// Blindheit: ein Dateiname ohne Ordner (router.go), ein Muster (api/bestell*.go), ein Ordner
// ohne Datei, ein Pfad hinter einem Commit (git show 424f186a:docs/x.md meint die Geschichte),
// eine Datei, die .gitignore ausnimmt, im Go-Code alles außer den Kommentaren, Migrationen und
// schema.sql (sie beschreiben den Stand ihres Tages), und ob die Datei noch enthält, was der
// Verweis meint.
//
// Bei Rot: den Pfad nachziehen. Nennt die Stelle mit Absicht eine Datei, die es nicht gibt,
// steht sie mit Grund in dateiverweiseOhneDatei.
func TestDateiverweise_ZeigenAufDateienDieEsGibt(t *testing.T) {
	baum := newDateiBaum(projektDateien(t))
	if len(baum.dateien) < 1500 {
		t.Fatalf("nur %d Dateien im Repository gelesen — die Prüfung mäße gegen einen leeren Baum", len(baum.dateien))
	}

	var fehlend []dateiverweis
	for _, quelle := range dateiverweisQuellen {
		gelesen := 0
		for _, datei := range slices.Sorted(maps.Keys(baum.dateien)) {
			if !quelle.gehoert(datei) {
				continue
			}
			roh, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(datei)))
			if err != nil {
				t.Fatalf("%s: %v", datei, err)
			}
			stuecke := []textstueck{{1, string(roh)}}
			if quelle.nurKommentare {
				stuecke = goKommentare(roh)
			}
			for _, stueck := range stuecke {
				verweise := pfadVerweise(stueck, baum, quelle.ab)
				if quelle.links {
					verweise = append(verweise, linkVerweise(stueck, path.Dir(datei))...)
				}
				gelesen += len(verweise)
				for _, v := range verweise {
					v.quelle = datei
					if !baum.gibtEines(v.ziele) {
						fehlend = append(fehlend, v)
					}
				}
			}
		}
		t.Logf("%s: %d Verweise gelesen", quelle.name, gelesen)
		// Nicht-leer-Garantie je Quelle: Ohne gelesene Verweise prüft der Test nichts.
		if gelesen < quelle.mindestens {
			t.Errorf("%s: nur %d Verweise gelesen, erwartet mindestens %d — der Sammler greift nicht mehr",
				quelle.name, gelesen, quelle.mindestens)
		}
	}

	ausgenommen := vonGitAusgenommen(t, fehlend)
	gedeckt := map[string]bool{}
	for _, v := range fehlend {
		schluessel := v.quelle + ": " + v.text
		if _, ok := dateiverweiseOhneDatei[schluessel]; ok {
			gedeckt[schluessel] = true
			continue
		}
		if slices.ContainsFunc(v.ziele, func(ziel string) bool { return ausgenommen[ziel] }) {
			continue
		}
		t.Errorf("%s:%d nennt %s — die Datei gibt es nicht. Den Pfad nachziehen; nennt die Stelle mit "+
			"Absicht eine Datei, die es nicht gibt, mit Grund in dateiverweiseOhneDatei eintragen (%q).",
			v.quelle, v.zeile, v.text, schluessel)
	}
	for schluessel := range dateiverweiseOhneDatei {
		if !gedeckt[schluessel] {
			t.Errorf("dateiverweiseOhneDatei führt %q, die Stelle gibt es nicht mehr oder die Datei ist da — austragen.", schluessel)
		}
	}
}

// dateiverweiseOhneDatei: „Quelle: Verweis" und warum die Stelle eine Datei nennt, die es nicht
// gibt.
var dateiverweiseOhneDatei = map[string]string{
	"docs/DEPLOYMENT.md: migrations/NNN_beschreibung.sql": "Muster für den Namen einer neuen Migration",
}

// dateiverweisQuelle beschreibt, welche Dateien gelesen werden und ab welchen Ordnern ein Pfad
// dort gemeint sein kann ("" ist die Wurzel des Repositorys).
type dateiverweisQuelle struct {
	name          string
	gehoert       func(datei string) bool
	nurKommentare bool
	links         bool
	ab            []string
	mindestens    int
}

var dateiverweisQuellen = []dateiverweisQuelle{
	{
		name: "Dokumente",
		gehoert: func(d string) bool {
			return strings.HasSuffix(d, ".md") && (strings.HasPrefix(d, "docs/") || !strings.Contains(d, "/"))
		},
		links: true, ab: []string{"", "frontend/"}, mindestens: 1000,
	},
	{
		name: "Kommentare im Go-Code",
		// docs/docs.go erzeugt swag aus den Kommentaren der Türen.
		gehoert:       func(d string) bool { return strings.HasSuffix(d, ".go") && d != "docs/docs.go" },
		nurKommentare: true, ab: []string{"", "frontend/"}, mindestens: 400,
	},
	{
		name: "Oberfläche",
		gehoert: func(d string) bool {
			return strings.HasPrefix(d, "frontend/") && hatEndung(d, "js", "mjs", "svelte", "css", "html")
		},
		ab: []string{"", "frontend/", "frontend/src/"}, mindestens: 200,
	},
	{
		name: "Skripte, Workflows und Bau",
		gehoert: func(d string) bool {
			return !strings.HasPrefix(d, "frontend/") &&
				(hatEndung(d, "sh", "py", "yml", "yaml") || path.Base(d) == "Dockerfile")
		},
		ab: []string{"", "frontend/"}, mindestens: 40,
	},
}

func hatEndung(datei string, endungen ...string) bool {
	return slices.Contains(endungen, strings.TrimPrefix(path.Ext(datei), "."))
}

// projektDateien nennt die Dateien des Repositorys: was git führt und was neu dazugekommen ist,
// ohne das, was .gitignore ausnimmt, und ohne das, was im Arbeitsbaum gelöscht ist.
func projektDateien(t *testing.T) []string {
	t.Helper()
	geloescht := gitListe(t, "ls-files", "-z", "--deleted")
	var dateien []string
	for _, datei := range gitListe(t, "ls-files", "-z", "--cached", "--others", "--exclude-standard") {
		if !slices.Contains(geloescht, datei) {
			dateien = append(dateien, datei)
		}
	}
	return dateien
}

func gitListe(t *testing.T, argumente ...string) []string {
	t.Helper()
	befehl := exec.Command("git", argumente...)
	befehl.Dir = ".."
	befehl.Env = umgebungOhneGit()
	ausgabe, err := befehl.Output()
	if err != nil {
		t.Fatalf("git %v: %v — die Prüfung misst an den Dateien des Repositorys und braucht git", argumente, err)
	}
	return strings.FieldsFunc(string(ausgabe), func(r rune) bool { return r == 0 })
}

// vonGitAusgenommen fragt für die fehlenden Ziele, welche .gitignore ausnimmt: eine gebaute
// oder am Arbeitsplatz abgelegte Datei, die das Repository mit Absicht nicht führt. Einen Pfad
// hinter einer Verknüpfung nimmt git nicht an; gefragt wird dann nach der Verknüpfung selbst.
func vonGitAusgenommen(t *testing.T, fehlend []dateiverweis) map[string]bool {
	t.Helper()
	zieleJeFrage := map[string][]string{}
	for _, v := range fehlend {
		for _, ziel := range v.ziele {
			frage := bisZurVerknuepfung(ziel, istVerknuepfung)
			zieleJeFrage[frage] = append(zieleJeFrage[frage], ziel)
		}
	}
	ausgenommen := map[string]bool{}
	if len(zieleJeFrage) == 0 {
		return ausgenommen
	}
	befehl := exec.Command("git", "check-ignore", "-z", "--stdin")
	befehl.Dir = ".."
	befehl.Env = umgebungOhneGit()
	befehl.Stdin = strings.NewReader(strings.Join(slices.Sorted(maps.Keys(zieleJeFrage)), "\x00"))
	ausgabe, err := befehl.Output()
	// Ende 1 heißt: keines der Ziele ist ausgenommen.
	if exit, ok := err.(*exec.ExitError); err != nil && (!ok || exit.ExitCode() != 1) {
		t.Fatalf("git check-ignore: %v", err)
	}
	for _, frage := range strings.FieldsFunc(string(ausgabe), func(r rune) bool { return r == 0 }) {
		for _, ziel := range zieleJeFrage[frage] {
			ausgenommen[ziel] = true
		}
	}
	return ausgenommen
}

// bisZurVerknuepfung kürzt einen Pfad auf seinen ersten Ordner, der eine Verknüpfung ist.
func bisZurVerknuepfung(pfad string, verknuepft func(string) bool) string {
	teile := strings.Split(pfad, "/")
	for i := 1; i < len(teile); i++ {
		if anfang := strings.Join(teile[:i], "/"); verknuepft(anfang) {
			return anfang
		}
	}
	return pfad
}

func istVerknuepfung(pfad string) bool {
	info, err := os.Lstat(filepath.Join("..", filepath.FromSlash(pfad)))
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// dateiBaum sind die Dateien des Repositorys mit ihren Ordnern.
type dateiBaum struct {
	dateien map[string]bool
	ordner  map[string]bool
}

func newDateiBaum(dateien []string) dateiBaum {
	baum := dateiBaum{dateien: map[string]bool{}, ordner: map[string]bool{}}
	for _, datei := range dateien {
		baum.dateien[datei] = true
		for ordner := path.Dir(datei); ordner != "."; ordner = path.Dir(ordner) {
			baum.ordner[ordner] = true
		}
	}
	return baum
}

func (b dateiBaum) gibtEines(ziele []string) bool {
	return slices.ContainsFunc(ziele, func(ziel string) bool { return b.dateien[ziel] || b.ordner[ziel] })
}

type textstueck struct {
	zeile int
	text  string
}

type dateiverweis struct {
	quelle string
	zeile  int
	text   string
	ziele  []string
}

// goKommentare liefert die Kommentare einer Go-Datei mit ihrer Zeile. Zeichenketten bleiben
// außen vor: Dort stehen erdachte Pfade der Tests.
func goKommentare(quelle []byte) []textstueck {
	dateisatz := token.NewFileSet()
	var leser scanner.Scanner
	leser.Init(dateisatz.AddFile("", dateisatz.Base(), len(quelle)), quelle, nil, scanner.ScanComments)
	var stuecke []textstueck
	for {
		stelle, zeichen, text := leser.Scan()
		if zeichen == token.EOF {
			return stuecke
		}
		if zeichen == token.COMMENT {
			stuecke = append(stuecke, textstueck{dateisatz.Position(stelle).Line, text})
		}
	}
}

var (
	pfadLauf = regexp.MustCompile(`[A-Za-z0-9_.@~/-]+`)
	// Hinter einem Commit meint der Pfad die Geschichte: git show 424f186a:docs/x.md.
	hinterCommit = regexp.MustCompile(`[0-9a-f]{7,40}:$`)
	linkZiel     = regexp.MustCompile(`\]\(([^)\s]+)\)`)
)

// dateiEndungen sind die Endungen, an denen ein Pfad als Verweis auf eine Datei gilt.
var dateiEndungen = []string{
	"go", "mod", "sum", "work", "sql", "sh", "py", "md", "txt", "csv", "json", "yml", "yaml", "toml", "xml",
	"properties", "example", "js", "mjs", "cjs", "svelte", "css", "html", "svg", "png", "jpg", "webp",
	"pdf", "xlsx",
}

// pfadVerweise liest aus einem Text die Pfade, die auf eine Datei zeigen: Der erste Ordner
// liegt unter einer der Wurzeln der Quelle, der letzte Teil trägt eine Dateiendung.
func pfadVerweise(stueck textstueck, baum dateiBaum, ab []string) []dateiverweis {
	var verweise []dateiverweis
	for _, stelle := range pfadLauf.FindAllStringIndex(stueck.text, -1) {
		pfad := strings.TrimRight(stueck.text[stelle[0]:stelle[1]], ".-/@~")
		erster, _, mehr := strings.Cut(pfad, "/")
		davor := stueck.text[:stelle[0]]
		// Hinter einem Dollarzeichen steht ein Kürzel der Oberfläche für einen Ordner ($lib/…),
		// kein Pfad ab einer Wurzel.
		if !mehr || !hatEndung(pfad, dateiEndungen...) || hinterCommit.MatchString(davor) || strings.HasSuffix(davor, "$") {
			continue
		}
		var ziele []string
		for _, wurzel := range ab {
			if baum.ordner[wurzel+erster] {
				ziele = append(ziele, path.Clean(wurzel+pfad))
			}
		}
		if len(ziele) == 0 {
			continue
		}
		zeile := stueck.zeile + strings.Count(stueck.text[:stelle[0]], "\n")
		verweise = append(verweise, dateiverweis{zeile: zeile, text: pfad, ziele: ziele})
	}
	return verweise
}

// linkVerweise liest die Ziele der Links eines Dokuments, vom Ordner des Dokuments aus. Eine
// Adresse im Netz und ein Sprung innerhalb der Seite sind keine Datei.
func linkVerweise(stueck textstueck, ordner string) []dateiverweis {
	var verweise []dateiverweis
	for _, treffer := range linkZiel.FindAllStringSubmatchIndex(stueck.text, -1) {
		roh := stueck.text[treffer[2]:treffer[3]]
		ziel, _, _ := strings.Cut(roh, "#")
		if ziel == "" || strings.Contains(ziel, ":") {
			continue
		}
		zeile := stueck.zeile + strings.Count(stueck.text[:treffer[0]], "\n")
		verweise = append(verweise, dateiverweis{zeile: zeile, text: "(" + roh + ")", ziele: []string{path.Join(ordner, ziel)}})
	}
	return verweise
}

// Der Sammler erkennt jede Form, in der eine Stelle eine Datei beim Pfad nennt, und hält nichts
// für einen Verweis, was keiner ist.
func TestDateiverweise_Selbstprobe(t *testing.T) {
	baum := newDateiBaum([]string{
		"api/router.go", "api/thema/tuer.go", "docs/OFFEN.md", "docs/datenschutz/nachweis.md", "README.md",
		"frontend/src/lib/stores/meldung.svelte.js", "frontend/scripts/gate.mjs", "scripts/gate.sh",
	})
	text := strings.Join([]string{
		"Die Tür steht in `api/router.go`, die Regel in api/thema/tuer.go.",                  // 1: zwei Verweise
		"Weg ist api/alt.go:42 und api/thema/alt_test.go, dazu docs/FEHLT.md.",               // 2: drei ohne Datei
		"Der Speicher (frontend/src/lib/stores/meldung.svelte.js) und src/lib/fehlt.js.",     // 3: einer da, einer ab frontend/ ohne Datei
		"Kein Verweis: router.go, api/bestell*.go, api/, bibliothek/api/fehlt.go,",           // 4
		"https://example.org/docs/fehlt.md, ../api/fehlt.go, pkg/lmf.SQLBedingung,",          // 5
		"git show 424f186a:docs/geschichte.md, ${wurzel}/api/fehlt.go, andere/fehlt.go,",     // 6
		"import x from '$lib/fehlt.js' und api/v1.2 und internal/service.",                   // 7
		"Links: [Plan](OFFEN.md), [Nachweis](datenschutz/nachweis.md#frist), [oben](#kopf),", // 8
		"[Netz](https://example.org/x.md), [weg](FEHLT.md), [hoch](../README.md).",           // 9
	}, "\n")
	stueck := textstueck{zeile: 10, text: text}

	var ohne []string
	gelesen := 0
	verweise := append(pfadVerweise(stueck, baum, []string{"", "frontend/", "frontend/src/"}), linkVerweise(stueck, "docs")...)
	for _, v := range verweise {
		gelesen++
		if !baum.gibtEines(v.ziele) {
			ohne = append(ohne, strconv.Itoa(v.zeile)+" "+v.text)
		}
	}
	sort.Strings(ohne)
	erwartet := []string{"11 api/alt.go", "11 api/thema/alt_test.go", "11 docs/FEHLT.md", "12 src/lib/fehlt.js", "18 (FEHLT.md)"}
	if !slices.Equal(ohne, erwartet) {
		t.Errorf("ohne Datei gemeldet:\n  %q\nerwartet:\n  %q", ohne, erwartet)
	}
	// Zwei in Zeile 1, drei in Zeile 2, zwei in Zeile 3 und vier Links auf eine Datei.
	if gelesen != 11 {
		t.Errorf("%d Verweise gelesen, erwartet 11", gelesen)
	}

	// Ab der Wurzel allein gilt ein Pfad ab frontend/ nicht als Verweis; ab frontend/src/ gilt
	// auch der Pfad, der dort beginnt.
	if v := pfadVerweise(textstueck{1, "src/lib/fehlt.js"}, baum, []string{""}); len(v) != 0 {
		t.Errorf("ab der Wurzel gelesen: %v", v)
	}
	v := pfadVerweise(textstueck{1, "lib/stores/meldung.svelte.js und scripts/gate.mjs"}, baum, []string{"", "frontend/", "frontend/src/"})
	if len(v) != 2 || !baum.gibtEines(v[0].ziele) || !baum.gibtEines(v[1].ziele) {
		t.Errorf("Pfade der Oberfläche: %v", v)
	}

	// Hinter einer Verknüpfung wird nach der Verknüpfung gefragt, sonst nach dem Pfad selbst.
	verknuepft := func(pfad string) bool { return pfad == "frontend/node_modules" }
	for pfad, frage := range map[string]string{
		"frontend/node_modules/paket/datei.go": "frontend/node_modules",
		"frontend/dist/assets/app.js":          "frontend/dist/assets/app.js",
		"frontend/node_modules":                "frontend/node_modules",
	} {
		if ist := bisZurVerknuepfung(pfad, verknuepft); ist != frage {
			t.Errorf("%s: gefragt wird nach %s, erwartet %s", pfad, ist, frage)
		}
	}

	// Im Go-Code zählen die Kommentare, nicht die Zeichenketten.
	quelle := "package x\n\n// Siehe api/router.go.\nvar a = \"api/erdacht.go\" /* und docs/OFFEN.md */\n"
	kommentare := goKommentare([]byte(quelle))
	if len(kommentare) != 2 || kommentare[0].zeile != 3 || kommentare[1].zeile != 4 ||
		strings.Contains(kommentare[0].text+kommentare[1].text, "erdacht") {
		t.Errorf("Kommentare: %v", kommentare)
	}
}
