package docs

import (
	"regexp"
	"strings"
	"testing"
)

// Gate: npm führt beim Installieren keine Skripte der Abhängigkeiten aus. Ein kompromittiertes
// Update bringt sein Schadprogramm meist in einem install- oder postinstall-Skript mit; es liefe
// auf dem Entwicklungsrechner mit Zugang zu GitHub, Servern und .env, in der CI und beim Bau des
// Images. Die Einstellung steht in frontend/.npmrc; der Bau des Images liest sie nur, wenn die
// Datei vor dem ersten npm ci im Image liegt.
//
// Blindheit: Ein npm install in einem anderen Ordner, ein Aufruf mit --ignore-scripts=false und
// ein Paket, dessen Schadprogramm erst beim Bauen oder im ausgelieferten Code wirkt.
func TestNpmInstalliertOhneSkripte(t *testing.T) {
	npmrc := lies(t, "../frontend/.npmrc")
	if !regexp.MustCompile(`(?m)^ignore-scripts=true$`).MatchString(npmrc) {
		t.Errorf("frontend/.npmrc setzt ignore-scripts=true nicht")
	}

	zeilen := strings.Split(lies(t, "../Dockerfile"), "\n")
	kopie, installation := -1, -1
	for i, z := range zeilen {
		z = strings.TrimSpace(z)
		if kopie < 0 && strings.HasPrefix(z, "COPY ") && strings.Contains(z, "frontend/.npmrc") {
			kopie = i
		}
		if installation < 0 && strings.HasPrefix(z, "RUN npm ci") {
			installation = i
		}
	}
	if installation < 0 {
		t.Fatal("kein RUN npm ci im Dockerfile — dieses Gate prüft dann nichts")
	}
	if kopie < 0 || kopie > installation {
		t.Errorf("das Dockerfile kopiert frontend/.npmrc nicht vor dem ersten RUN npm ci (Zeile %d)", installation+1)
	}
}
