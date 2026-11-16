package mailservice

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
)

// leseMailNachricht liest eine von baueMailNachricht erzeugte Nachricht mit den Lesern der
// Standardbibliothek zurück. Gelingt das ohne Fehler, stimmt der Aufbau nach MIME; die
// Kopfzeilen zeigen, ob etwas eingeschleust wurde.
func leseMailNachricht(t *testing.T, roh []byte) (*mail.Message, *multipart.Reader) {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(roh))
	if err != nil {
		t.Fatalf("Nachricht muss als RFC-822-Message parsebar sein: %v", err)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("mime.ParseMediaType: %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("mediaType = %q, erwartet %q", mediaType, "multipart/mixed")
	}
	if params["boundary"] == "" {
		t.Fatal("params[\"boundary\"] ist leer")
	}

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
	if err != nil {
		t.Fatalf("baueMailNachricht: %v", err)
	}

	msg, mr := leseMailNachricht(t, raw)
	if got := msg.Header.Get("From"); got != "bibliothek@example.com" {
		t.Errorf("From = %q, erwartet %q", got, "bibliothek@example.com")
	}
	if got := msg.Header.Get("To"); got != "empfaenger@example.com" {
		t.Errorf("To = %q, erwartet %q", got, "empfaenger@example.com")
	}
	if got := msg.Header.Get("Subject"); got != "Mahnliste Klasse 4a" {
		t.Errorf("Subject = %q, erwartet %q", got, "Mahnliste Klasse 4a")
	}
	if got := msg.Header.Get("MIME-Version"); got != "1.0" {
		t.Errorf("MIME-Version = %q, erwartet %q", got, "1.0")
	}

	bodyPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("mr.NextPart: %v", err)
	}
	if got := bodyPart.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, erwartet %q", got, "text/plain; charset=utf-8")
	}
	bodyText, err := io.ReadAll(bodyPart)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if string(bodyText) != req.Text {
		t.Errorf("string(bodyText) = %q, erwartet %q", string(bodyText), req.Text)
	}

	attPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("mr.NextPart: %v", err)
	}
	if got := attPart.Header.Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, erwartet %q", got, "application/pdf")
	}
	if got := attPart.Header.Get("Content-Transfer-Encoding"); got != "base64" {
		t.Errorf("Content-Transfer-Encoding = %q, erwartet %q", got, "base64")
	}

	_, dispParams, err := mime.ParseMediaType(attPart.Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("mime.ParseMediaType: %v", err)
	}
	if dispParams["filename"] != "mahnliste-4a.pdf" {
		t.Errorf("dispParams[\"filename\"] = %q, erwartet %q", dispParams["filename"], "mahnliste-4a.pdf")
	}

	// Der Inhalt eines Anhangs kommt durch Base64 hin und zurück unverändert an.
	encoded, err := io.ReadAll(attPart)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		t.Fatalf("base64.StdEncoding.DecodeString: %v", err)
	}
	if !bytes.Equal(decoded, req.Anhaenge[0].Inhalt) {
		t.Errorf("decoded = %q, erwartet %q", decoded, req.Anhaenge[0].Inhalt)
	}

	_, err = mr.NextPart()
	if err != io.EOF {
		t.Errorf("mr.NextPart() = %v, erwartet io.EOF: die Nachricht hat einen Teil zu viel", err)
	}
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
	if err != nil {
		t.Fatalf("baueMailNachricht: %v", err)
	}

	msg, mr := leseMailNachricht(t, raw)
	if got := msg.Header.Get("Bcc"); got != "" {
		t.Errorf("Bcc = %q, erwartet leer: Anhang-Name darf keine Top-Level-Header einschleusen", got)
	}

	bodyPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("mr.NextPart: %v", err)
	}
	_, _ = io.Copy(io.Discard, bodyPart) //nolint:errcheck

	attPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("mr.NextPart: %v", err)
	}
	if got := attPart.Header.Get("X-Injected"); got != "" {
		t.Errorf("X-Injected = %q, erwartet leer: ContentType darf keine Part-Header einschleusen", got)
	}
	if got := attPart.Header.Get("Bcc"); got != "" {
		t.Errorf("Bcc = %q, erwartet leer", got)
	}
	if got := attPart.Header.Get("Content-Disposition"); strings.Contains(got, "\n") {
		t.Errorf("Content-Disposition = %q, erwartet ohne %q", got, "\n")
	}
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
	if err != nil {
		t.Fatalf("baueMailNachricht: %v", err)
	}

	_, mr := leseMailNachricht(t, raw)
	bodyPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("mr.NextPart: %v", err)
	}
	_, _ = io.Copy(io.Discard, bodyPart) //nolint:errcheck

	attPart, err := mr.NextPart()
	if err != nil {
		t.Fatalf("mr.NextPart: %v", err)
	}
	_, dispParams, err := mime.ParseMediaType(attPart.Header.Get("Content-Disposition"))
	if err != nil {
		t.Fatalf("mime.ParseMediaType: %v", err)
	}
	if _, ok := dispParams["evil"]; ok {
		t.Errorf("dispParams hat den Schlüssel %q", "evil")
	}
	if strings.Contains(dispParams["filename"], `"`) {
		t.Errorf("dispParams[\"filename\"] = %q, erwartet ohne %q", dispParams["filename"], `"`)
	}
}
