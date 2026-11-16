package mailservice

import (
	"errors"
	"mime"
	"net/mail"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Die Kopfzeilen auf dem Weg, über den jede echte Mail geht. Neun Stellen bauen eine Nachricht
// (Mahnungen, Bestellungen, Laufzettel der Abgänger, Antworten auf Anliegen, Alarme); geprüft
// wird an der einen Stelle, an der die Kopfzeilen entstehen.
//
// Der Umschlag (MAIL FROM / RCPT TO) bestimmt, wohin zugestellt wird, die Kopfzeilen der
// Nachricht bestimmen, was der Empfänger sieht: ein zweites From, ein Reply-To auf eine fremde
// Adresse, ein vorzeitiges Ende des Kopfteils und damit gefälschter Text vor dem Inhalt.
func TestBaueMailNachricht_WeistUmbruecheInAdressenAb(t *testing.T) {
	faelle := []struct {
		name string
		req  Nachricht
		from string
	}{
		{
			name: "Empfänger mit eingeschmuggelter Kopfzeile",
			req:  Nachricht{An: "lehrer@schule.de\r\nBcc: fremd@example.org", Betreff: "Mahnung", Text: "Text"},
			from: "bibliothek@schule.de",
		},
		{
			name: "Absender mit Umbruch",
			req:  Nachricht{An: "lehrer@schule.de", Betreff: "Mahnung", Text: "Text"},
			from: "bibliothek@schule.de\nReply-To: fremd@example.org",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			_, err := baueMailNachricht(f.req, f.from)
			if err == nil {
				t.Fatal("die Nachricht wurde gebaut — eine Adresse mit Zeilenumbruch muss abgewiesen werden")
			}
			if !errors.Is(err, errUmbruchInKopfzeile) {
				t.Errorf("unerwarteter Fehler: %v", err)
			}
		})
	}
}

// Ein Betreff aus einer Vorlage ist etwas anderes als eine Adresse: Dort bleibt beim
// Bearbeiten leicht ein Umbruch stehen. Der wird geglättet, nicht abgewiesen, aber er darf keine
// zweite Kopfzeile öffnen.
func TestBaueMailNachricht_GlaettetBetreffStattIhnAbzuweisen(t *testing.T) {
	req := Nachricht{
		An:      "lehrer@schule.de",
		Betreff: "Erinnerung\r\nX-Gefaelscht: ja",
		Text:    "Text",
	}

	roh, err := baueMailNachricht(req, "bibliothek@schule.de")
	if err != nil {
		t.Fatalf("ein Betreff mit Umbruch soll geglättet werden, nicht scheitern: %v", err)
	}

	kopf := string(roh)
	if i := strings.Index(kopf, "\r\n\r\n"); i > 0 {
		kopf = kopf[:i]
	}
	// Es geht um Zeilenanfänge: Der Text darf im Betreff stehen bleiben (dort ist er
	// harmlos), er darf nur keine eigene Kopfzeile eröffnen.
	for _, zeile := range strings.Split(kopf, "\r\n") {
		if strings.HasPrefix(zeile, "X-Gefaelscht:") {
			t.Errorf("der Umbruch hat eine zweite Kopfzeile geöffnet:\n%s", kopf)
		}
	}
	if !strings.Contains(kopf, "Subject: Erinnerung X-Gefaelscht: ja") {
		t.Errorf("Betreff wurde nicht wie erwartet geglättet:\n%s", kopf)
	}
}

// Eine Kopfzeile ist ASCII (RFC 5322, 2.2); ein Betreff mit Umlaut geht als kodiertes Wort hinaus
// (RFC 2047), keine Zeile ist länger als 998 Zeichen (2.1.1), und jede Mail trägt Datum und
// Kennung (3.6.1, 3.6.4). Nicht jeder Mailserver ergänzt sie, und Filter werten eine Mail ohne sie
// als verdächtig. Beide Bauweisen, die einfache Textmail und die mit Anhängen.
func TestMailKopf_NachRFC5322(t *testing.T) {
	// Der längste Betreff, den mail_vorlagen.betreff (VARCHAR(255)) fasst, aus lauter Umlauten:
	// kodiert über 1.500 Zeichen.
	langerBetreff := strings.Repeat("ü", 255)
	for _, betreff := range []string{"Überfällige Bücher der Klasse 05F1", langerBetreff} {
		textmail, err := baueTextNachricht("bibliothek@schule.example", "lehrer@schule.example", betreff, "Text")
		if err != nil {
			t.Fatal(err)
		}
		anhangmail, err := baueMailNachricht(Nachricht{An: "lehrer@schule.example", Betreff: betreff, Text: "Text"}, "bibliothek@schule.example")
		if err != nil {
			t.Fatal(err)
		}
		kennungen := map[string]bool{}
		for name, roh := range map[string][]byte{"Textmail": textmail, "Mail mit Anhängen": anhangmail} {
			kopf, _, _ := strings.Cut(string(roh), "\r\n\r\n")
			for i, b := range []byte(kopf) {
				if b > 127 {
					t.Errorf("%s: Byte %d im Kopf ist kein ASCII:\n%s", name, i, kopf)
					break
				}
			}
			for _, zeile := range strings.Split(kopf, "\r\n") {
				if len(zeile) > 998 {
					t.Errorf("%s: Kopfzeile mit %d Zeichen, erlaubt sind 998", name, len(zeile))
				}
			}
			nachricht, err := mail.ReadMessage(strings.NewReader(string(roh)))
			if err != nil {
				t.Fatalf("%s: mail.ReadMessage: %v", name, err)
			}
			if got, err := new(mime.WordDecoder).DecodeHeader(nachricht.Header.Get("Subject")); err != nil || got != betreff {
				t.Errorf("%s: Betreff dekodiert zu %q (%v), erwartet %q", name, got, err, betreff)
			}
			if datum, err := nachricht.Header.Date(); err != nil || time.Since(datum).Abs() > time.Minute {
				t.Errorf("%s: Date %q (%v), erwartet den Augenblick des Bauens", name, nachricht.Header.Get("Date"), err)
			}
			kennung := nachricht.Header.Get("Message-ID")
			if !regexp.MustCompile(`^<[A-Za-z0-9._-]+@schule\.example>$`).MatchString(kennung) || kennungen[kennung] {
				t.Errorf("%s: Message-ID %q, erwartet eine eigene <…@schule.example>", name, kennung)
			}
			kennungen[kennung] = true
			if got := nachricht.Header.Get("MIME-Version"); got != "1.0" {
				t.Errorf("%s: MIME-Version %q, erwartet 1.0", name, got)
			}
		}
	}
}
