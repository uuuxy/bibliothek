package docs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Gate für scripts/pruefe_secrets.sh: Das Skript kennt die Beispielwerte der Vorlage und
// des Servers und meldet jede Umgebung, in der sie gelten.
//
// Blindheit: Geprüft wird das Skript an erdachten .env-Dateien. Ob es am Server je läuft
// (nur von Hand), sieht der Test nicht; ebenso wenig einen schwachen Wert, der kein
// Beispiel ist, und einen Beispielwert, der weder in .env.example noch in
// api.IstBekanntesDefaultGeheimnis steht.

// pruefeSecrets lässt scripts/pruefe_secrets.sh über eine .env laufen und liefert Ausgabe
// und Exit-Code.
func pruefeSecrets(t *testing.T, env string) (ausgabe string, exitCode int) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(pfad, []byte(env), 0o600); err != nil {
		t.Fatalf(".env schreiben: %v", err)
	}
	roh, err := exec.Command("bash", "../scripts/pruefe_secrets.sh", pfad).CombinedOutput()
	if err != nil {
		var ende *exec.ExitError
		if !errors.As(err, &ende) {
			t.Fatalf("pruefe_secrets.sh ließ sich nicht starten: %v", err)
		}
		exitCode = ende.ExitCode()
	}
	return string(roh), exitCode
}

// saubereEnv ist eine .env, an der das Skript nichts auszusetzen hat. Die Tests ändern
// daran je eine Zeile; die Werte sind erkennbar keine echten Geheimnisse.
func saubereEnv(ersetze map[string]string) string {
	werte := [][2]string{
		{"JWT_SECRET", "nur-fuer-den-test-nur-fuer-den-test-0001"},
		{"APP_ENCRYPTION_KEY", "nur-fuer-den-test-nur-fuer-0002x"},
		{"POSTGRES_PASSWORD", "nur-fuer-den-test-0003"},
		{"BACKUP_ENCRYPTION_KEY", "nur-fuer-den-test-nur-fuer-den-test-0004"},
		{"COOKIE_SECURE", "true"},
		{"IMAP_HOST", "imap.example.test"},
		{"APP_ENV", "production"},
	}
	var b strings.Builder
	for _, w := range werte {
		wert := w[1]
		if neu, ok := ersetze[w[0]]; ok {
			wert = neu
		}
		b.WriteString(w[0] + "=" + wert + "\n")
	}
	return b.String()
}

const meldungBeispielwert = " nutzt einen bekannten Default-Wert"

// Ohne diesen Treffer wären alle Meldungen darunter auch dann grün, wenn das Skript jede
// .env beanstandet.
func TestPruefeSecrets_SaubereEnvIstInOrdnung(t *testing.T) {
	ausgabe, code := pruefeSecrets(t, saubereEnv(nil))
	if code != 0 || strings.Contains(ausgabe, "✗") {
		t.Fatalf("eine saubere .env wird beanstandet (Exit %d):\n%s", code, ausgabe)
	}
}

// Wer .env.example kopiert, hat die drei Beispielwerte der Vorlage in der Datei. Das
// Skript liest sie aus der Vorlage selbst, damit seine Liste ihr nicht davonläuft.
func TestPruefeSecrets_KenntDieBeispielwerteDerVorlage(t *testing.T) {
	ausgabe, code := pruefeSecrets(t, lies(t, "../.env.example"))
	if code != 1 {
		t.Errorf("die unveränderte Vorlage endet mit Exit %d, erwartet 1", code)
	}
	for _, name := range []string{"JWT_SECRET", "APP_ENCRYPTION_KEY", "POSTGRES_PASSWORD"} {
		if !strings.Contains(ausgabe, name+meldungBeispielwert) {
			t.Errorf("der Beispielwert der Vorlage für %s wird nicht als Beispiel gemeldet:\n%s", name, ausgabe)
		}
	}
}

// Was der Server als Beispiel-Geheimnis kennt (api.IstBekanntesDefaultGeheimnis), meldet
// auch das Skript. Die Liste kommt aus dem Quelltext des Servers, nicht aus diesem Test.
func TestPruefeSecrets_KenntDieBeispielwerteDesServers(t *testing.T) {
	quelle := lies(t, "../api/betriebsbereitschaft.go")
	start := strings.Index(quelle, "func IstBekanntesDefaultGeheimnis")
	if start < 0 {
		t.Fatal("IstBekanntesDefaultGeheimnis nicht gefunden — Gate nachziehen")
	}
	ende := strings.Index(quelle[start:], "return true")
	if ende < 0 {
		t.Fatal("Ende der Liste in IstBekanntesDefaultGeheimnis nicht gefunden")
	}
	var werte []string
	wertMuster := regexp.MustCompile(`"([^"]+)"`)
	for _, zeile := range strings.Split(quelle[start:start+ende], "\n") {
		if kommentar := strings.Index(zeile, "//"); kommentar >= 0 {
			zeile = zeile[:kommentar]
		}
		for _, m := range wertMuster.FindAllStringSubmatch(zeile, -1) {
			werte = append(werte, m[1])
		}
	}
	if len(werte) < 5 {
		t.Fatalf("nur %d Beispielwerte im Server gefunden — die Probe läuft ins Leere", len(werte))
	}
	for _, wert := range werte {
		ausgabe, _ := pruefeSecrets(t, saubereEnv(map[string]string{"JWT_SECRET": wert}))
		if !strings.Contains(ausgabe, "JWT_SECRET"+meldungBeispielwert) {
			t.Errorf("den Beispielwert %q verweigert der Server, das Skript meldet ihn nicht", wert)
		}
	}
}

// APP_ENV=local, development oder test erlaubt die Beispiel-Geheimnisse und die
// Anmeldung mit jedem Passwort. Der Server liest den Wert ohne Rücksicht auf
// Großschreibung und Leerzeichen, das Skript also auch.
func TestPruefeSecrets_MeldetJedeSpielwiese(t *testing.T) {
	for _, wert := range []string{"local", "development", "test", "Local", "TEST", " local "} {
		ausgabe, code := pruefeSecrets(t, saubereEnv(map[string]string{"APP_ENV": wert}))
		if code != 1 || !strings.Contains(ausgabe, "✗ APP_ENV=") {
			t.Errorf("APP_ENV=%q wird nicht als kritisch gemeldet (Exit %d)", wert, code)
		}
	}
}
