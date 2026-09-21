package docs

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Gate gegen auseinanderlaufende Postgres-Major-Versionen.
//
// Anlass (31.08.2026): Produktion lief auf Postgres 15, der lokale Stack und E2E aber
// monatelang auf 16 — Constraints, Fehlerbilder und pg_dump-Kompatibilität wurden damit
// gegen eine Version bewiesen, die nirgends in Betrieb war (behoben in 02ac3912, am
// selben Tag Sprung aller Umgebungen auf 18 in 5602e871). Die Klasse dahinter: Die
// Versionsnummer steht an sechs Stellen in fünf Dateien, und nichts zwang sie zur
// Einigkeit — ein Paar (bzw. Sextett), das nur zufällig einig war.
//
// Dieser Test liest alle Fundstellen und verlangt EINE Major-Version. Bewusst keine
// festgeschriebene Zahl: Ein künftiger Sprung auf 19 bleibt ein normaler Commit, der
// alle Dateien gemeinsam zieht — nur ein halber Sprung wird rot.
//
// Reparatur bei Rot: die genannten Dateien auf dieselbe Major-Version bringen. Kommt
// eine neue Fundstelle hinzu (weiteres Compose, neues Skript mit eigenem Pin), gehört
// sie hier in die Liste.
func TestPostgresMajorUeberallGleich(t *testing.T) {
	imagePin := regexp.MustCompile(`image:\s*postgres:(\d+)-alpine`)
	fundstellen := []struct {
		pfad   string
		muster []*regexp.Regexp
	}{
		{"../docker-compose.yml", []*regexp.Regexp{imagePin}},
		{"../docker-compose.local.yml", []*regexp.Regexp{imagePin}},
		{"../.github/workflows/ci.yml", []*regexp.Regexp{
			imagePin,
			regexp.MustCompile(`PG_MAJOR=(\d+)`),
		}},
		{"../Dockerfile", []*regexp.Regexp{
			regexp.MustCompile(`postgresql(\d+)-client`),
		}},
		{"../scripts/sonar_scan.sh", []*regexp.Regexp{
			regexp.MustCompile(`postgres:(\d+)-alpine`),
		}},
	}

	type pin struct {
		pfad  string
		major string
	}
	var pins []pin

	for _, f := range fundstellen {
		inhalt, err := os.ReadFile(f.pfad)
		if err != nil {
			t.Fatalf("%s lesen: %v", f.pfad, err)
		}
		for _, muster := range f.muster {
			treffer := muster.FindAllStringSubmatch(string(inhalt), -1)
			// Liveness je MUSTER, nicht je Datei: In ci.yml stehen zwei Pins — fiele
			// nur einer weg (Zeile umformuliert), hielte der andere die Datei-Zählung
			// grün, und das Gate wäre für diese Fundstelle still abgeschaltet. Genau
			// so in der Gegenprobe gesehen (PG_MAJOR umbenannt → Test blieb grün).
			if len(treffer) == 0 {
				t.Fatalf("in %s greift das Muster %q nicht mehr (Datei umformuliert?) — "+
					"das Gate wäre für diese Fundstelle abgeschaltet. Muster hier nachziehen.",
					f.pfad, muster.String())
			}
			for _, tr := range treffer {
				pins = append(pins, pin{f.pfad, tr[1]})
			}
		}
	}

	referenz := pins[0]
	for _, p := range pins[1:] {
		if p.major != referenz.major {
			t.Errorf("Postgres-Major läuft auseinander: %s sagt %s, %s sagt %s — "+
				"alle Umgebungen (Prod-Compose, lokaler Stack, CI, Dockerfile-Client, "+
				"Sonar-Skript) müssen dieselbe Major-Version fahren.",
				referenz.pfad, referenz.major, p.pfad, p.major)
		}
	}
}

// leseEinePin liest genau einen Versions-Pin aus einer Datei; findet das Muster nichts,
// ist der Detektor tot und der Test scheitert laut (Regel 2 in sweeps.md).
func leseEinePin(t *testing.T, pfad string, muster *regexp.Regexp) string {
	t.Helper()
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	treffer := muster.FindStringSubmatch(string(inhalt))
	if treffer == nil {
		t.Fatalf("in %s greift das Muster %q nicht mehr (Datei umformuliert?) — "+
			"das Gate wäre für diese Fundstelle abgeschaltet. Muster hier nachziehen.",
			pfad, muster.String())
	}
	return treffer[1]
}

// Zwilling des Postgres-Gates, gefunden über die Zwillings-Pflicht (sweeps.md Regel 5)
// am Tag seiner Einführung: Dependabot-PR #515 hob NUR das Dockerfile auf golang 1.27.0
// — go.mod (und damit die CI, die über go-version-file daran hängt) blieb auf 1.26.6.
// Getestet wurde seither mit einer anderen Toolchain, als das Prod-Binary baute. Der
// Kommentar über der FROM-Zeile verlangte die Paarung ausdrücklich („Beim naechsten
// go.mod-Go-Bump diese Zeile mitziehen") — eine Verabredung, die nichts erzwang.
//
// Patch-genau, nicht nur Major: CVE-Fixes leben in Stdlib-Patches, und
// GOTOOLCHAIN=local im Builder lädt nichts nach (Begründung im Dockerfile).
func TestGoToolchainDockerfileFolgtGoMod(t *testing.T) {
	ausGoMod := leseEinePin(t, "../go.mod", regexp.MustCompile(`(?m)^go (\d+\.\d+\.\d+)$`))
	ausDockerfile := leseEinePin(t, "../Dockerfile", regexp.MustCompile(`FROM golang:(\d+\.\d+\.\d+)-alpine`))
	if ausGoMod != ausDockerfile {
		t.Errorf("Go-Toolchain läuft auseinander: go.mod sagt %s, der Dockerfile-Builder %s — "+
			"bei einem Go-Bump beide zusammen ziehen (Dependabot hebt nur das Dockerfile!).",
			ausGoMod, ausDockerfile)
	}
	// Die dritte Stelle ist bewusst KEIN Literal: die CI liest go.mod. Verschwindet
	// diese Kopplung (jemand pinnt in ci.yml wieder eine Zahl), reißt das Gate.
	leseEinePin(t, "../.github/workflows/ci.yml", regexp.MustCompile(`go-version-file:\s*'?(go\.mod)'?`))

	// Dritter Ort, gefunden am 07.09.2026 beim Nachziehen von Dependabot-PR #589:
	// go.work trägt dieselbe Zahl noch einmal, und er stand hier nicht. Beim letzten
	// Sprung (b6f29aaa) wurde er von Hand mitgezogen; verlassen konnte man sich darauf
	// nicht.
	//
	// Die beiden Richtungen sind NICHT gleich gefährlich, und nur eine braucht dieses
	// Gate:
	//   - go.work HINTER go.mod: Das fängt Go selbst ab, sofort und unübersehbar
	//     („module . listed in go.work file requires go >= 1.27.1, but go.work lists go
	//     1.27.0"). Es beendet jedes `go build` und `go test` — auch dieses hier, das
	//     dann gar nicht erst läuft. Dafür braucht es keine Ratsche.
	//   - go.work VOR go.mod: Das lässt Go stillschweigend zu. Gebaut und getestet wird
	//     dann mit einer neueren Toolchain, als go.mod erklärt — genau der Zustand, den
	//     dieses Gate für den Dockerfile schon einmal gefunden hat. Diese Richtung ist
	//     der Grund für die Prüfung.
	ausGoWork := leseEinePin(t, "../go.work", regexp.MustCompile(`(?m)^go (\d+\.\d+\.\d+)$`))
	if ausGoWork != ausGoMod {
		t.Errorf("go.work läuft auseinander: go.mod sagt %s, go.work %s — gebaut und "+
			"getestet wird dann mit der Toolchain aus go.work, nicht mit der erklärten. "+
			"Beide zusammen ziehen.",
			ausGoMod, ausGoWork)
	}
}

// Dritter Zwilling: Das ausgelieferte Bundle baut der Dockerfile-node-Builder — die CI
// testete aber auf Node 24, was nirgends in Betrieb war (dieselbe Klasse wie der
// Postgres-15/16-Fund, nur im Frontend).
func TestNodeMajorUeberallGleich(t *testing.T) {
	sollMajor := leseEinePin(t, "../Dockerfile", regexp.MustCompile(`FROM node:(\d+)-alpine`))

	for _, workflow := range []string{"../.github/workflows/ci.yml", "../.github/workflows/security-scan.yml"} {
		inhalt, err := os.ReadFile(workflow)
		if err != nil {
			t.Fatalf("%s lesen: %v", workflow, err)
		}
		muster := regexp.MustCompile(`node-version:\s*"?(\d+)"?`)
		treffer := muster.FindAllStringSubmatch(string(inhalt), -1)
		if len(treffer) == 0 {
			t.Fatalf("in %s greift das Muster %q nicht mehr (Datei umformuliert?) — "+
				"das Gate wäre für diese Fundstelle abgeschaltet.", workflow, muster.String())
		}
		for _, tr := range treffer {
			if tr[1] != sollMajor {
				t.Errorf("Node-Major läuft auseinander: Dockerfile-Builder baut das Bundle "+
					"mit Node %s, %s testet mit Node %s.", sollMajor, workflow, tr[1])
			}
		}
	}
}

// Vierter Zwilling (07.09.2026): Die Jobs der Workflows und die Pflichtliste des Tag-Gates
// sind dasselbe Versprechen an zwei Orten — „bevor ein Tag ein Release und ein Image
// erzeugt, muss geprüft sein, was diese Anwendung prüft".
//
// Sie liefen auseinander, und zwar in beide Richtungen gleichzeitig: release.yml
// verlangte `build-and-test` und `docker-scan`. Der erste Name stimmte, der zweite zeigte
// ins Leere — einen Prüflauf dieses Namens gab es nie. In der alten jq-Zeile fiel das
// nicht auf: `[…] | all(.conclusion=="success")` ist für die leere Menge WAHR. Zugleich
// fehlten `frontend-test` und vor allem `e2e`. Am 06.09.2026 war main vier Stunden rot,
// genau in e2e; ein Tag in diesem Fenster hätte ein Release samt Image erzeugt, und das
// Gate hätte grün dazu genickt.
//
// Deshalb Gleichheit als MENGE, nicht Teilmenge: Ein neuer Job, den die Liste nicht kennt,
// ist genauso ein Loch wie ein Name in der Liste, den kein Workflow mehr baut.
//
// Seit dem 21.09.2026 steht die Liste EINMAL in scripts/tag-gate.sh —
// vorher wörtlich in release.yml und docker-publish.yml —, und die Quelle sind ZWEI
// Workflows: Zu den Jobs der CI kommen die vier Security-Jobs. Anlass war ein Fall, den es
// wirklich gab: Am Commit bbad2f39 (07.09.2026) waren alle vier CI-Jobs grün und der
// Trivy-Scan rot; ein Tag darauf hätte ein Release erzeugt.
//
// Verglichen wird der Name, unter dem GitHub den Prüflauf führt: das Feld `name:` des Jobs,
// und wo es fehlt, der Job-Schlüssel. Die Security-Jobs tragen ein `name:` — mit dem
// Schlüssel (`docker-scan`) in der Liste zeigte der Name wieder ins Leere.
func TestTagGateVerlangtAllePrueflaeufe(t *testing.T) {
	ausWorkflows := map[string]bool{}
	for _, workflow := range []string{"../.github/workflows/ci.yml", "../.github/workflows/security-scan.yml"} {
		namen := prueflaufNamen(t, workflow)
		// Nicht-leer-Garantie je Quelle: Eine umformulierte Datei, aus der kein Job mehr
		// gelesen wird, schaltete sonst ihre Hälfte des Gates still ab.
		if len(namen) == 0 {
			t.Fatalf("in %s wurde kein einziger Job gefunden (Datei umformuliert?) — das Gate wäre für sie abgeschaltet", workflow)
		}
		for name := range namen {
			ausWorkflows[name] = true
		}
	}

	imGate := map[string]bool{}
	for _, name := range tagGatePflichtliste(t) {
		imGate[name] = true
	}

	for name := range ausWorkflows {
		if !imGate[name] {
			t.Errorf("Prüflauf %q steht nicht in der Pflichtliste von scripts/tag-gate.sh — ein Tag "+
				"würde Release und Image erzeugen, ohne dass er grün sein muss.", name)
		}
	}
	for name := range imGate {
		if !ausWorkflows[name] {
			t.Errorf("scripts/tag-gate.sh verlangt %q, aber weder ci.yml noch security-scan.yml führen "+
				"einen Prüflauf dieses Namens — der Name zeigt ins Leere (der Fall 'docker-scan'). "+
				"Gemeint ist das Feld `name:` des Jobs, nicht sein Schlüssel.", name)
		}
	}
}

// Die Liste nützt nur, wenn beide Tag-Workflows sie auch befragen.
//
// Bis zum 17.09.2026 prüfte docker-publish.yml beim v-Tag nur Muster und main. Ein Tag auf
// rotem Stand erzeugte damit zwar kein Release, aber sehr wohl ein Image mit einer
// Versionsnummer — und eine Versionsnummer sagt „geprüft". Folgenlos war das nur, solange
// die Produktion selbst baut; wer aus dem Image deployt, zöge einen Stand, den kein Gate
// gesehen hat.
//
// Eine eigene Liste im Workflow ist seit dem 21.09.2026 ebenfalls ein Fehler: Sie wäre die
// zweite Abschrift, die dieses Skript gerade abgeschafft hat — und sie kennte die
// Security-Jobs nicht.
func TestTagWorkflowsRufenDasTagGate(t *testing.T) {
	for _, workflow := range []string{"../.github/workflows/release.yml", "../.github/workflows/docker-publish.yml"} {
		inhalt, err := os.ReadFile(workflow)
		if err != nil {
			t.Fatalf("%s lesen: %v", workflow, err)
		}
		// Kommentarzeilen zählen nicht: Ein Aufruf, der nur noch im Kommentar steht, prüft
		// nichts (Bugklasse „lügende Ratsche durch Kommentar").
		var ohneKommentare []string
		for _, zeile := range strings.Split(string(inhalt), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(zeile), "#") {
				ohneKommentare = append(ohneKommentare, zeile)
			}
		}
		text := strings.Join(ohneKommentare, "\n")
		if !regexp.MustCompile(`(?m)^\s*\./scripts/tag-gate\.sh\s*$`).MatchString(text) {
			t.Errorf("%s ruft ./scripts/tag-gate.sh nicht (mehr) auf — ein v-Tag erzeugte sein "+
				"Ergebnis, ohne dass irgendein Prüflauf grün sein muss.", workflow)
		}
		if regexp.MustCompile(`(?m)^\s*PFLICHT=`).MatchString(text) {
			t.Errorf("%s führt wieder eine eigene PFLICHT-Liste — die Liste steht einmal, in "+
				"scripts/tag-gate.sh.", workflow)
		}
	}
}

// tagGatePflichtliste liest die Pflichtliste aus scripts/tag-gate.sh: eine Zeile je Name,
// weil die Namen Leerzeichen tragen. Eine leere Liste ist ein Fehler — jedes Gate, das auf
// ihr aufbaut, wäre sonst still grün.
func tagGatePflichtliste(t *testing.T) []string {
	t.Helper()
	pflicht := leseEinePin(t, "../scripts/tag-gate.sh", regexp.MustCompile(`(?m)^PFLICHT="([^"]+)"`))
	var namen []string
	for _, zeile := range strings.Split(pflicht, "\n") {
		if name := strings.TrimSpace(zeile); name != "" {
			namen = append(namen, name)
		}
	}
	if len(namen) == 0 {
		t.Fatal("scripts/tag-gate.sh hat keine Pflichtliste mehr — dieses Gate wäre still grün")
	}
	return namen
}

// prueflaufNamen liest, unter welchen Namen GitHub die Jobs eines Workflows als Prüfläufe
// führt: das Feld `name:` direkt unter dem Job-Schlüssel, und wo es fehlt, der Schlüssel
// selbst. Bewusst ohne YAML-Bibliothek — das Gate soll an der Datei hängen, wie sie
// dasteht, und nicht an einer Abhängigkeit, die dieses Repo sonst nirgends braucht.
func prueflaufNamen(t *testing.T, pfad string) map[string]bool {
	t.Helper()
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s lesen: %v", pfad, err)
	}
	schluesselMuster := regexp.MustCompile(`^  ([A-Za-z][A-Za-z0-9_-]*):\s*$`)
	// Genau vier Leerzeichen: das `name:` des Jobs. Die `- name:` der Schritte stehen tiefer.
	nameMuster := regexp.MustCompile(`^    name:\s*(.+?)\s*$`)

	// Je Job-Schlüssel der Anzeigename; leer heißt: kein `name:`, der Schlüssel gilt.
	anzeige := map[string]string{}
	aktuell := ""
	inJobs := false
	for _, zeile := range strings.Split(string(inhalt), "\n") {
		if strings.HasPrefix(zeile, "jobs:") {
			inJobs = true
			continue
		}
		if !inJobs {
			continue
		}
		// Eine Zeile ohne Einrückung beendet den jobs-Block.
		if zeile != "" && !strings.HasPrefix(zeile, " ") {
			break
		}
		if treffer := schluesselMuster.FindStringSubmatch(zeile); treffer != nil {
			aktuell = treffer[1]
			anzeige[aktuell] = ""
			continue
		}
		if treffer := nameMuster.FindStringSubmatch(zeile); treffer != nil && aktuell != "" {
			anzeige[aktuell] = strings.Trim(treffer[1], `"'`)
		}
	}

	namen := map[string]bool{}
	for schluessel, name := range anzeige {
		if name == "" {
			name = schluessel
		}
		namen[name] = true
	}
	return namen
}
