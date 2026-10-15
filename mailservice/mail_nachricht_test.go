package mailservice

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

// leseMailNachricht liest eine von baueMailNachricht erzeugte Nachricht mit den Lesern der
// Standardbibliothek zurück. Gelingt das ohne Fehler, stimmt der Aufbau nach MIME; die
// Kopfzeilen zeigen, ob etwas eingeschleust wurde.
func leseMailNachricht(t *testing.T, roh []byte) (*mail.Message, *multipart.Reader) {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(roh))
	require.NoError(t, err, "Nachricht muss als RFC-822-Message parsebar sein")

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/mixed", mediaType)
	require.NotEmpty(t, params["boundary"])

	return msg, multipart.NewReader(msg.Body, params["boundary"])
}

func TestBaueMailNachricht_StrukturUndInhalt(t *testing.T) {
	req := Nachricht{
		An:      "empfaenger@example.com",
		Betreff: "Mahnliste Klasse 4a",
		Text:    "Hallo,\r\nanbei die Mahnliste.",
		Anhaenge: []Anhang{{
			Name:        "mahnliste-4a.pdf",
			ContentType: "application/pdf",
			Inhalt:      []byte("%PDF-1.4 testinhalt"),
		}},
	}

	raw, err := baueMailNachricht(req, "bibliothek@example.com")
	require.NoError(t, err)

	msg, mr := leseMailNachricht(t, raw)
	assert.Equal(t, "bibliothek@example.com", msg.Header.Get("From"))
	assert.Equal(t, "empfaenger@example.com", msg.Header.Get("To"))
	assert.Equal(t, "Mahnliste Klasse 4a", msg.Header.Get("Subject"))
	assert.Equal(t, "1.0", msg.Header.Get("MIME-Version"))

	bodyPart, err := mr.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "text/plain; charset=utf-8", bodyPart.Header.Get("Content-Type"))
	bodyText, err := io.ReadAll(bodyPart)
	require.NoError(t, err)
	assert.Equal(t, req.Text, string(bodyText))

	attPart, err := mr.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "application/pdf", attPart.Header.Get("Content-Type"))
	assert.Equal(t, "base64", attPart.Header.Get("Content-Transfer-Encoding"))

	_, dispParams, err := mime.ParseMediaType(attPart.Header.Get("Content-Disposition"))
	require.NoError(t, err)
	assert.Equal(t, "mahnliste-4a.pdf", dispParams["filename"])

	// Der Inhalt eines Anhangs kommt durch Base64 hin und zurück unverändert an.
	encoded, err := io.ReadAll(attPart)
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	require.NoError(t, err)
	assert.Equal(t, req.Anhaenge[0].Inhalt, decoded)

	_, err = mr.NextPart()
	assert.Equal(t, io.EOF, err, "es darf keine weiteren Parts geben")
}

func TestBaueMailNachricht_AnhangHeaderInjection(t *testing.T) {
	// Name und ContentType versuchen, mit CRLF weitere Kopfzeilen einzuschleusen; schreibeAnhang
	// entfernt beides.
	req := Nachricht{
		An:      "empfaenger@example.com",
		Betreff: "Betreff",
		Text:    "Text",
		Anhaenge: []Anhang{{
			Name:        "liste\r\nBcc: attacker@evil.example\r\n.pdf",
			ContentType: "application/pdf\r\nX-Injected: ja",
			Inhalt:      []byte("daten"),
		}},
	}

	raw, err := baueMailNachricht(req, "bibliothek@example.com")
	require.NoError(t, err)

	msg, mr := leseMailNachricht(t, raw)
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
	// Ein Anführungszeichen im Namen darf den Text in Anführungszeichen des Parameters filename
	// nicht aufbrechen (eingeschleuste Parameter in Content-Disposition).
	req := Nachricht{
		An:      "empfaenger@example.com",
		Betreff: "Betreff",
		Text:    "Text",
		Anhaenge: []Anhang{{
			Name:        `boese".pdf"; evil="x`,
			ContentType: "application/pdf",
			Inhalt:      []byte("daten"),
		}},
	}

	raw, err := baueMailNachricht(req, "bibliothek@example.com")
	require.NoError(t, err)

	_, mr := leseMailNachricht(t, raw)
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
