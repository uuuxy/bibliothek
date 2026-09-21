package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Gate: Jede Umgebungsvariable, die Compose dem Backend gibt, wird auch gelesen.
//
// Anlass (Register, Bestands-Durchgang 10.09.2026): `DB_HOST=postgres-db` und
// `SMTP_SENDER=${SMTP_FROM}` standen in beiden Compose-Dateien, und kein Go-Code fragte
// je danach. Solche Zeilen sind nicht bloß Ballast — sie sehen aus wie eine Einstellung.
// Wer den Datenbank-Host ändern will, ändert DB_HOST, startet neu und wundert sich; die
// Verbindung kommt aus DATABASE_URL.
//
// Die Gegenrichtung (Go liest etwas, das Compose NICHT durchreicht) ist der teurere Fall
// — er hat den nächtlichen Backup-Job zweimal still ausfallen lassen. Sie steht seit dem
// 21.09.2026 weiter unten: TestGelesenesReichtComposeDurch.
var composeVariable = regexp.MustCompile(`^\s+-\s+([A-Z0-9_]+)=`)

// backendBlock liefert den Abschnitt des backend-Dienstes einer Compose-Datei.
func backendBlock(t *testing.T, inhalt string) string {
	t.Helper()
	start := strings.Index(inhalt, "\n  backend:")
	if start < 0 {
		t.Fatal("kein backend-Dienst in der Compose-Datei — dieses Gate wäre still grün")
	}
	rest := inhalt[start+1:]
	// Der nächste Dienst auf derselben Ebene beendet den Block.
	if m := regexp.MustCompile(`\n  [a-z0-9_-]+:\n`).FindStringIndex(rest); m != nil {
		return rest[:m[0]]
	}
	return rest
}

func TestComposeVariablenWerdenGelesen(t *testing.T) {
	// Der Go-Code des Servers als eine Zeichenkette: gesucht wird der Name in
	// Anführungszeichen, so wie os.Getenv ihn nimmt.
	var quellen strings.Builder
	err := filepath.Walk("..", func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) //nolint:gosec // Repo-eigene Quellen
		if err != nil {
			return err
		}
		quellen.Write(inhalt)
		return nil
	})
	if err != nil {
		t.Fatalf("Go-Quellen sammeln: %v", err)
	}
	code := quellen.String()
	if len(code) < 100_000 {
		t.Fatalf("nur %d Bytes Go-Code gesammelt — der Sammler greift ins Leere", len(code))
	}

	for _, datei := range []string{"../docker-compose.yml", "../docker-compose.local.yml"} {
		inhalt := lies(t, datei)
		gesehen := map[string]bool{}
		var namen []string
		for _, zeile := range strings.Split(backendBlock(t, inhalt), "\n") {
			m := composeVariable.FindStringSubmatch(zeile)
			if m == nil || gesehen[m[1]] {
				continue
			}
			gesehen[m[1]] = true
			namen = append(namen, m[1])
		}
		if len(namen) < 10 {
			t.Fatalf("%s: nur %d Variablen gefunden — der Sammler greift ins Leere", datei, len(namen))
		}
		sort.Strings(namen)
		for _, name := range namen {
			if !strings.Contains(code, `"`+name+`"`) {
				t.Errorf("%s reicht %s in den Container, aber kein Go-Code liest es — "+
					"entweder die Zeile streichen oder den Leser bauen (eine Zeile, die nach "+
					"Einstellung aussieht und keine ist, kostet jemanden eine Stunde)", datei, name)
			}
		}
	}
}

// Fünf Variablen, die der Server liest und die docker-compose.yml bis zum 21.09.2026 nicht
// durchreichte (OFFEN.md 4.7): Im Container galt immer die eingebaute Vorgabe, egal was in
// der .env stand. Dieses Gate hält für genau diese fünf die FORM fest (${NAME:-}, leer =
// Vorgabe des Servers); dass überhaupt durchgereicht wird, prüft die allgemeine Ratsche
// TestGelesenesReichtComposeDurch darunter.
func TestComposeReichtDieFuenfVariablenDurch(t *testing.T) {
	block := backendBlock(t, lies(t, "../docker-compose.yml"))
	for _, name := range []string{
		"ALLOWED_ORIGIN", "RATE_LIMIT", "IMAP_PORT", "SMTP_ALLOW_INSECURE_TLS", "SMTP_ALLOW_PLAINTEXT",
	} {
		// Als ${NAME:-}: Leer ergibt die eingebaute Vorgabe. Ein fester Wert hier würde die
		// Vorgabe des Servers überstimmen, ohne dass es jemand in der .env sieht.
		if !strings.Contains(block, "      - "+name+"=${"+name+":-}\n") {
			t.Errorf("docker-compose.yml reicht %s nicht als ${%s:-} an das Backend durch — "+
				"im Container gälte immer die eingebaute Vorgabe, egal was in der .env steht", name, name)
		}
	}
}

// Die Gegenrichtung: Was der Go-Code aus der Umgebung liest, muss docker-compose.yml dem
// Backend auch geben — sonst gilt im Container immer die eingebaute Vorgabe, egal was in
// der .env steht. So fiel der nächtliche Backup-Job zweimal still aus, und so blieben fünf
// Variablen bis zum 21.09.2026 wirkungslos (OFFEN.md 4.7).
//
// composeAusnahmen sind die Namen, die der Backend-Container NICHT bekommen soll, je mit
// Begründung. Die Liste ist eine Ratsche in beide Richtungen: Ein neuer gelesener Name ohne
// Compose-Zeile ist rot, und ein Eintrag hier, den niemand mehr liest oder den Compose
// inzwischen durchreicht, ebenfalls — eine Liste, die Erledigtes weiterführt, verliert ihre
// Aussage.
var composeAusnahmen = map[string]string{
	"FOTOS_BEHALTEN":    "Werkzeug cmd/migrate-fotos, von Hand gestartet — kein Teil des Servers",
	"MYSQL_DSN":         "Werkzeug cmd/migrate (Littera-Übernahme), von Hand gestartet",
	"PG_DSN":            "Werkzeug cmd/migrate (Littera-Übernahme), von Hand gestartet",
	"TEST_DATABASE_URL": "internal/pgtest — nur für Tests, im Container gibt es keine Test-Datenbank",
}

var (
	// os.Getenv("NAME"), os.LookupEnv("NAME") — und dieselben Aufrufe mit einer Konstante.
	envLesestelle = regexp.MustCompile(`os\.(?:Getenv|LookupEnv)\(\s*([^)]+?)\s*\)`)
	// const name = "WERT" bzw. name = "WERT" in einem const-Block.
	envKonstante = regexp.MustCompile(`(?m)^\s*(?:const\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*"([A-Z][A-Z0-9_]*)"`)
)

func TestGelesenesReichtComposeDurch(t *testing.T) {
	var quellen strings.Builder
	err := filepath.Walk("..", func(pfad string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" || name == "frontend" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(pfad, ".go") || strings.HasSuffix(pfad, "_test.go") {
			return nil
		}
		inhalt, err := os.ReadFile(pfad) //nolint:gosec // Repo-eigene Quellen
		if err != nil {
			return err
		}
		quellen.Write(inhalt)
		quellen.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatalf("Go-Quellen sammeln: %v", err)
	}
	code := quellen.String()

	konstanten := map[string]string{}
	for _, m := range envKonstante.FindAllStringSubmatch(code, -1) {
		konstanten[m[1]] = m[2]
	}

	gelesen := map[string]bool{}
	for _, m := range envLesestelle.FindAllStringSubmatch(code, -1) {
		argument := m[1]
		if strings.HasPrefix(argument, `"`) {
			gelesen[strings.Trim(argument, `"`)] = true
			continue
		}
		// crypto.SchluesselVariable → SchluesselVariable. Lässt sich eine Lesestelle nicht
		// auflösen, sähe die Ratsche sie nicht — dann lieber rot als still grün.
		bezeichner := argument[strings.LastIndex(argument, ".")+1:]
		wert, ok := konstanten[bezeichner]
		if !ok {
			t.Errorf("os.Getenv(%s): Der Name lässt sich nicht auflösen — keine Konstante %q mit "+
				"einem Wert in GROSSBUCHSTABEN gefunden. Die Ratsche sähe diese Lesestelle nicht.",
				argument, bezeichner)
			continue
		}
		gelesen[wert] = true
	}
	// Die Nicht-leer-Garantie: Greift das Muster ins Leere, wäre alles Folgende still grün.
	if len(gelesen) < 25 {
		t.Fatalf("nur %d gelesene Umgebungsnamen erkannt — das Muster greift vermutlich nicht mehr", len(gelesen))
	}

	durchgereicht := map[string]bool{}
	for _, zeile := range strings.Split(backendBlock(t, lies(t, "../docker-compose.yml")), "\n") {
		if m := composeVariable.FindStringSubmatch(zeile); m != nil {
			durchgereicht[m[1]] = true
		}
	}

	var namen []string
	for name := range gelesen {
		namen = append(namen, name)
	}
	sort.Strings(namen)
	for _, name := range namen {
		if !durchgereicht[name] && composeAusnahmen[name] == "" {
			t.Errorf("Der Go-Code liest %s, docker-compose.yml gibt es dem Backend nicht — im Container "+
				"gälte immer die eingebaute Vorgabe.\n→ Als „- %s=${%s:-}“ in den backend-Block aufnehmen, "+
				"oder mit Begründung in composeAusnahmen eintragen.", name, name, name)
		}
	}
	for name, grund := range composeAusnahmen {
		switch {
		case !gelesen[name]:
			t.Errorf("composeAusnahmen führt %s (%s), aber kein Go-Code liest es mehr — austragen.", name, grund)
		case durchgereicht[name]:
			t.Errorf("composeAusnahmen führt %s, docker-compose.yml reicht es aber durch — austragen.", name)
		}
	}
}
