package smtptest

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

// DateiAusMail schält einen Anhang aus der MIME-Nachricht und dekodiert ihn.
func DateiAusMail(t *testing.T, nachricht, namensteil string) string {
	t.Helper()
	teile := strings.Split(nachricht, "--"+mimeGrenze(t, nachricht))
	for _, teil := range teile {
		if !strings.Contains(teil, namensteil) {
			continue
		}
		leer := strings.Index(teil, "\r\n\r\n")
		if leer < 0 {
			t.Fatalf("Anhang %q hat keinen Rumpf", namensteil)
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
