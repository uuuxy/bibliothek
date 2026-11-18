package smtptest

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

// DateiAusMail schält einen Anhang aus der MIME-Nachricht und dekodiert ihn. Gesucht wird
// namensteil nur im Kopf eines Teils: Im Base64-Rumpf eines anderen Anhangs kann dieselbe
// Buchstabenfolge zufällig stehen.
func DateiAusMail(t *testing.T, nachricht, namensteil string) string {
	t.Helper()
	teile := strings.Split(nachricht, "--"+mimeGrenze(t, nachricht))
	for _, teil := range teile[1:] {
		leer := strings.Index(teil, "\r\n\r\n")
		if leer < 0 || !strings.Contains(teil[:leer], namensteil) {
			continue
		}
		roh := strings.NewReplacer("\r", "", "\n", "").Replace(teil[leer:])
		daten, err := base64.StdEncoding.DecodeString(strings.TrimSpace(roh))
		if err != nil {
			t.Fatalf("Anhang %q nicht dekodierbar: %v", namensteil, err)
		}
		return string(daten)
	}
	t.Fatalf("Anhang %q nicht in der Mail", namensteil)
	return ""
}

// PDFAusMail holt den ersten PDF-Anhang aus der Nachricht.
func PDFAusMail(t *testing.T, nachricht string) []byte {
	t.Helper()
	return []byte(DateiAusMail(t, nachricht, "application/pdf"))
}

var grenzeMuster = regexp.MustCompile(`boundary=([^\s;"]+)`)

func mimeGrenze(t *testing.T, nachricht string) string {
	t.Helper()
	treffer := grenzeMuster.FindStringSubmatch(nachricht)
	if treffer == nil {
		t.Fatal("keine MIME-Grenze in der Nachricht — die Mail ist kaputt")
	}
	return treffer[1]
}
