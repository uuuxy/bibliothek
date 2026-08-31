package docs

import (
	"os"
	"regexp"
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
