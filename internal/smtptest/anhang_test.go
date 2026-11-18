package smtptest

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Der Name eines Anhangs steht in seinem Kopf. Im Base64-Rumpf eines anderen Anhangs kann
// dieselbe Buchstabenfolge zufällig vorkommen; dann gab der Helfer den falschen Anhang zurück
// (TestAbgaengerkette fand Annas Blatt unter „Ben").
func TestDateiAusMail_NimmtDenNamenNurAusDemKopf(t *testing.T) {
	// Ein Rumpf, dessen Base64 „Ben" enthält: "\x05\xe9\xe7" kodiert zu "Ben".
	anna := string([]byte{0x05, 0xe9, 0xe7}) + "Annas Blatt"
	if kodiert := base64.StdEncoding.EncodeToString([]byte(anna)); !strings.Contains(kodiert, "Ben") {
		t.Fatalf("Vorbedingung: %q enthält kein „Ben\"", kodiert)
	}
	teil := func(name, inhalt string) string {
		return "--GRENZE\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"" +
			name + ".pdf\"\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
			base64.StdEncoding.EncodeToString([]byte(inhalt)) + "\r\n"
	}
	nachricht := "Content-Type: multipart/mixed; boundary=GRENZE\r\n\r\n" +
		"--GRENZE\r\nContent-Type: text/plain\r\n\r\nHallo\r\n" +
		teil("Kontoauszug_Anna", anna) + teil("Kontoauszug_Ben", "Bens Blatt") + "--GRENZE--\r\n"

	if got := DateiAusMail(t, nachricht, "Ben"); got != "Bens Blatt" {
		t.Errorf("DateiAusMail(…, \"Ben\") = %q, erwartet \"Bens Blatt\"", got)
	}
	if got := DateiAusMail(t, nachricht, "Anna"); got != anna {
		t.Errorf("DateiAusMail(…, \"Anna\") = %q, erwartet Annas Blatt", got)
	}
}
