package docs

import (
	"regexp"
	"strings"
	"testing"
)

// Zwei Ratschen an der Deployment-Anleitung. Sie ist die Datei, die jemand abtippt, wenn
// er den Server neu aufsetzt — und die beiden Funde vom 10.09.2026 (Register) waren genau
// das: Anweisungen, die dem Rest derselben Datei widersprachen.

// §2.1 legt für die GANZE Anleitung fest: Schlüssel in Hex, nie in Base64 — ein
// POSTGRES_PASSWORD mit `+` oder `/` zerlegt die DSN. §2.3 sagte trotzdem
// `openssl rand -base64 48`, und der zweite Befehl daneben lieferte 16 statt 32 Byte.
// Wer der einen Stelle folgte statt der anderen, bekam einen Schlüssel, den der Server
// ablehnt — oder ein Passwort, an dem die Verbindung scheitert.
func TestDeploymentErzeugtSchluesselNurInHex(t *testing.T) {
	anleitung := lies(t, "DEPLOYMENT.md")
	if strings.Contains(anleitung, "rand -base64") {
		t.Error("DEPLOYMENT.md erzeugt einen Schlüssel mit `openssl rand -base64` — §2.1 schließt das " +
			"für die ganze Anleitung aus (ein POSTGRES_PASSWORD mit + oder / zerlegt die DSN)")
	}
	// Der AES-Schlüssel ist 32 Byte lang; `-hex 16` wären 16. Die kurzen Formen kommen
	// aus der alten Fassung von §2.3.
	for _, kurz := range []string{"rand -hex 16", "rand -hex 8"} {
		if strings.Contains(anleitung, kurz) {
			t.Errorf("DEPLOYMENT.md nennt `openssl %s` — zu kurz für die verlangten 32 Byte "+
				"(APP_ENCRYPTION_KEY: 64 Hex-Zeichen)", kurz)
		}
	}
}

// Die Anleitung beschreibt, was das Release-Gate prüft. Sie stand auf dem Stand vor dem
// 06.09.2026 („nur build-and-test, e2e gehört nicht dazu") — inzwischen verlangt
// release.yml alle vier CI-Jobs. Eine Anleitung, die weniger verspricht als die Schranke
// hält, ist harmlos; eine, die MEHR verspricht, wäre gefährlich. Beides fängt derselbe
// Abgleich: Jeder Name aus der Pflichtliste steht auch in der Anleitung.
func TestDeploymentNenntDiePflichtlisteDesReleaseGates(t *testing.T) {
	pflicht := leseEinePin(t, "../.github/workflows/release.yml", regexp.MustCompile(`(?m)^\s*PFLICHT="([^"]+)"`))
	namen := strings.Fields(pflicht)
	if len(namen) == 0 {
		t.Fatal("release.yml hat keine Pflichtliste mehr — dieses Gate wäre still grün")
	}
	anleitung := lies(t, "DEPLOYMENT.md")
	for _, name := range namen {
		if !strings.Contains(anleitung, name) {
			t.Errorf("release.yml verlangt den Job %q, DEPLOYMENT.md §8 nennt ihn nicht — "+
				"wer die Anleitung liest, hält das Gate für durchlässiger, als es ist", name)
		}
	}
}
