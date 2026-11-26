package smtptest

import (
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

// TextAusMail liefert den Text der Nachricht, wie ihn der Empfänger liest. Der Text geht in
// quoted-printable hinaus: Umlaute stehen dort als =C3=BC, und eine lange Zeile ist mit „=“
// umbrochen; ein Satz oder ein Link lässt sich im rohen Inhalt deshalb nicht finden.
func TextAusMail(t *testing.T, nachricht string) string {
	t.Helper()
	m, err := mail.ReadMessage(strings.NewReader(nachricht))
	if err != nil {
		t.Fatalf("Nachricht nicht lesbar: %v", err)
	}
	art, parameter, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("Content-Type nicht lesbar: %v", err)
	}
	if !strings.HasPrefix(art, "multipart/") {
		return leseText(t, m.Body, m.Header.Get("Content-Transfer-Encoding"))
	}
	teile := multipart.NewReader(m.Body, parameter["boundary"])
	for {
		teil, err := teile.NextPart()
		if err != nil {
			t.Fatalf("kein Textteil in der Nachricht: %v", err)
		}
		if strings.HasPrefix(teil.Header.Get("Content-Type"), "text/plain") {
			// multipart.Reader entfernt quoted-printable selbst und löscht dabei die Kopfzeile.
			return leseText(t, teil, teil.Header.Get("Content-Transfer-Encoding"))
		}
	}
}

func leseText(t *testing.T, rumpf io.Reader, kodierung string) string {
	t.Helper()
	if strings.EqualFold(kodierung, "quoted-printable") {
		rumpf = quotedprintable.NewReader(rumpf)
	}
	text, err := io.ReadAll(rumpf)
	if err != nil {
		t.Fatalf("Text nicht lesbar: %v", err)
	}
	return string(text)
}
