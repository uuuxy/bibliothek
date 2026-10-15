package docs

import (
	"strings"
	"testing"
)

// Zwei Ratschen an der Deployment-Anleitung. Sie ist die Datei, die jemand abtippt, wenn er
// den Server neu aufsetzt; eine Anweisung, die dem Rest derselben Datei widerspricht, landet
// so auf dem Server.

// §2.1 legt für die ganze Anleitung fest: Schlüssel in Hex, nie in Base64, denn ein
// POSTGRES_PASSWORD mit `+` oder `/` zerlegt die DSN. Ein Befehl, der davon abweicht oder zu
// kurz ist, liefert einen Schlüssel, den der Server ablehnt, oder ein Passwort, an dem die
// Verbindung scheitert.
func TestDeploymentErzeugtSchluesselNurInHex(t *testing.T) {
	anleitung := lies(t, "DEPLOYMENT.md")
	if strings.Contains(anleitung, "rand -base64") {
		t.Error("DEPLOYMENT.md erzeugt einen Schlüssel mit `openssl rand -base64` — §2.1 schließt das " +
			"für die ganze Anleitung aus (ein POSTGRES_PASSWORD mit + oder / zerlegt die DSN)")
	}
	// Der AES-Schlüssel ist 32 Byte lang; `-hex 16` wären 16.
	for _, kurz := range []string{"rand -hex 16", "rand -hex 8"} {
		if strings.Contains(anleitung, kurz) {
			t.Errorf("DEPLOYMENT.md nennt `openssl %s` — zu kurz für die verlangten 32 Byte "+
				"(APP_ENCRYPTION_KEY: 64 Hex-Zeichen)", kurz)
		}
	}
}

// Die Anleitung beschreibt, was das Tag-Gate prüft; nennt sie weniger Prüfläufe, hält der
// Leser das Gate für durchlässiger, als es ist. Jeder Name aus der Pflichtliste in
// scripts/tag-gate.sh steht deshalb auch in der Anleitung, wörtlich, so wie GitHub den
// Prüflauf führt.
func TestDeploymentNenntDiePflichtlisteDesReleaseGates(t *testing.T) {
	anleitung := lies(t, "DEPLOYMENT.md")
	for _, name := range tagGatePflichtliste(t) {
		if !strings.Contains(anleitung, name) {
			t.Errorf("scripts/tag-gate.sh verlangt den Prüflauf %q, DEPLOYMENT.md §8 nennt ihn nicht — "+
				"wer die Anleitung liest, hält das Gate für durchlässiger, als es ist", name)
		}
	}
}
