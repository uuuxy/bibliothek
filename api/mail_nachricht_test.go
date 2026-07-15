package api

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseMailNachricht liest eine von baueMailNachricht erzeugte Nachricht mit den
// Standardparsern zurück. Dass das ohne Fehler gelingt, beweist die MIME-Struktur;
// die Header-Maps zeigen, ob eine Injection durchgeschlagen hat.
func parseMailNachricht(t *testing.T, raw []byte) (*mail.Message, *multipart.Reader) {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	require.NoError(t, err, "Nachricht muss als RFC-822-Message parsebar sein")

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/mixed", mediaType)
	require.NotEmpty(t, params["boundary"])

	return msg, multipart.NewReader(msg.Body, params["boundary"])
}

func TestBaueMailNachricht_StrukturUndInhalt(t *testing.T) {
	req := MailRequest{
		To:      "empfaenger@example.com",
		Subject: "Mahnliste Klasse 4a",
		Body:    "Hallo,\r\nanbei die Mahnliste.",
		Attachments: []MailAttachment{{
			Name:        "mahnliste-4a.pdf",
			ContentType: "application/pdf",
			Data:        []byte("%PDF-1.4 testinhalt"),
		}},
	}

	raw, err := baueMailNachricht(req, "bibliothek@example.com")
	require.NoError(t, err)

	msg, mr := parseMailNachricht(t, raw)
	assert.Equal(t, "bibliothek@example.com", msg.Header.Get("From"))
	assert.Equal(t, "empfaenger@example.com", msg.Header.Get("To"))
	assert.Equal(t, "Mahnliste Klasse 4a", msg.Header.Get("Subject"))
	assert.Equal(t, "1.0", msg.Header.Get("MIME-Version"))

	bodyPart, err := mr.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "text/plain; charset=utf-8", bodyPart.Header.Get("Content-Type"))
	bodyText, err := io.ReadAll(bodyPart)
	require.NoError(t, err)
	assert.Equal(t, req.Body, string(bodyText))

	attPart, err := mr.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "application/pdf", attPart.Header.Get("Content-Type"))
	assert.Equal(t, "base64", attPart.Header.Get("Content-Transfer-Encoding"))

	_, dispParams, err := mime.ParseMediaType(attPart.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	assert.Equal(t, "mahnliste-4a.pdf", dispParams["filename"])

	// Anhang-Daten müssen die Base64-Rundreise unverändert überleben.
	encoded, err := io.ReadAll(attPart)
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	require.NoError(t, err)
	assert.Equal(t, req.Attachments[0].Data, decoded)

	_, err = mr.NextPart()
	assert.Equal(t, io.EOF, err, "es darf keine weiteren Parts geben")
}

func TestBaueMailNachricht_AnhangHeaderInjection(t *testing.T) {
	// Name und ContentType versuchen, per CRLF zusätzliche Header einzuschleusen —
	// beides muss in schreibeAnhang neutralisiert werden.
	req := MailRequest{
		To:      "empfaenger@example.com",
		Subject: "Betreff",
		Body:    "Text",
		Attachments: []MailAttachment{{
			Name:        "liste\r\nBcc: attacker@evil.example\r\n.pdf",
			ContentType: "application/pdf\r\nX-Injected: ja",
			Data:        []byte("daten"),
		}},
	}

	raw, err := baueMailNachricht(req, "bibliothek@example.com")
	require.NoError(t, err)

	msg, mr := parseMailNachricht(t, raw)
	assert.Empty(t, msg.Header.Get("Bcc"), "Anhang-Name darf keine Top-Level-Header einschleusen")

	bodyPart, err := mr.NextPart()
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, bodyPart) //nolint:errcheck

	attPart, err := mr.NextPart()
	require.NoError(t, err)
	assert.Empty(t, attPart.Header.Get("X-Injected"), "ContentType darf keine Part-Header einschleusen")
	assert.Empty(t, attPart.Header.Get("Bcc"))
	assert.NotContains(t, attPart.Header.Get("Content-Disposition"), "\n")
}

func TestBaueMailNachricht_AnhangNameMitAnfuehrungszeichen(t *testing.T) {
	// Ein Anführungszeichen im Namen darf den quoted-string des filename-Parameters
	// nicht aufbrechen (Parameter-Injection in Content-Disposition).
	req := MailRequest{
		To:      "empfaenger@example.com",
		Subject: "Betreff",
		Body:    "Text",
		Attachments: []MailAttachment{{
			Name:        `boese".pdf"; evil="x`,
			ContentType: "application/pdf",
			Data:        []byte("daten"),
		}},
	}

	raw, err := baueMailNachricht(req, "bibliothek@example.com")
	require.NoError(t, err)

	_, mr := parseMailNachricht(t, raw)
	bodyPart, err := mr.NextPart()
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, bodyPart) //nolint:errcheck

	attPart, err := mr.NextPart()
	require.NoError(t, err)
	_, dispParams, err := mime.ParseMediaType(attPart.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	assert.NotContains(t, dispParams, "evil")
	assert.NotContains(t, dispParams["filename"], `"`)
}
