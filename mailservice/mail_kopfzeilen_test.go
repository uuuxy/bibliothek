package mailservice

import (
	"errors"
	"strings"
	"testing"
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
