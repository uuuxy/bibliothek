package api

import (
	"strings"
	"testing"
)

// Erreicht der Bestätigungs-Link die Bestellmail?
//
// Diese Tests bewachen die Stelle, an der das Feature still ausfallen kann, ohne dass
// irgendein anderer Test rot wird: Die öffentliche Seite funktioniert auch dann noch
// tadellos, wenn der Link gar nicht erst verschickt wurde — nur klickt ihn dann niemand.
// Das Projekt hat genau diese Bugklasse schon einmal getroffen (Commit „Signatur
// erreicht den Lieferanten-Mailanhang").
//
// Besonders wichtig, weil der Mailtext über den Vorlagen-Editor frei änderbar ist: Wer
// die Vorlage umformuliert und dabei den Platzhalter verliert, darf den Ablauf nicht
// mitnehmen.

const testLink = "https://bib.example.invalid/bestellung/TESTTOKEN"

// Vorlage ohne Platzhalter — der Normalfall, denn die ausgelieferte Vorlage aus
// Migration 052 kennt {{.BestaetigungsLink}} nicht. Der Link muss trotzdem mit.
func TestBestellmail_LinkWirdAngehaengtWennVorlageIhnNichtKennt(t *testing.T) {
	_, body := resolveBestellMail("Betreff", "Sehr geehrte Damen und Herren,\n\nanbei die Bestellung.", "K-1", 2, 5, testLink)

	if !strings.Contains(body, testLink) {
		t.Fatalf("Link fehlt in der Mail — der Lieferant bekäme nichts zum Anklicken.\nBody:\n%s", body)
	}
	if !strings.Contains(body, "anbei die Bestellung.") {
		t.Error("der ursprüngliche Vorlagentext wurde beim Anhängen verloren")
	}
}

// Die ausgelieferte Vorlage (Migration 052 = derselbe Text wie der Fallback) enthält
// keinen Platzhalter. Fiele das eines Tages um, wäre der Anhänge-Pfad tot Code — und
// dieser Test der einzige Ort, an dem das auffällt.
func TestBestellmail_AusgelieferteVorlageBrauchtDenAnhaengePfad(t *testing.T) {
	if strings.Contains(bestellMailFallbackBody, "{{.BestaetigungsLink}}") {
		t.Skip("Vorlage trägt den Platzhalter jetzt selbst — Anhänge-Pfad nicht mehr nötig")
	}
	_, body := resolveBestellMail(bestellMailFallbackBetreff, bestellMailFallbackBody, "K-1", 2, 5, testLink)
	if !strings.Contains(body, testLink) {
		t.Fatal("die ausgelieferte Vorlage verschickt den Link nicht")
	}
}

// Vorlage MIT Platzhalter: Der Link gehört an die vorgesehene Stelle — und darf nicht
// zusätzlich unten angehängt werden.
func TestBestellmail_PlatzhalterWirdErsetztUndNichtVerdoppelt(t *testing.T) {
	vorlage := "Guten Tag,\n\nBestätigung hier: {{.BestaetigungsLink}}\n\nViele Grüße"
	_, body := resolveBestellMail("Betreff", vorlage, "K-1", 2, 5, testLink)

	if strings.Contains(body, "{{.BestaetigungsLink}}") {
		t.Error("Platzhalter blieb unersetzt im Text stehen")
	}
	if anzahl := strings.Count(body, testLink); anzahl != 1 {
		t.Errorf("Link steht %d× in der Mail, erwartet genau 1× (kein doppelter Anhang)", anzahl)
	}
}

// Lieferant ohne Bestätigungsschritt (oder keine öffentliche Adresse hinterlegt): Es
// darf weder ein Anhang noch ein sichtbarer Platzhalter-Rest in der Mail landen.
func TestBestellmail_OhneLinkBleibtDieMailSauber(t *testing.T) {
	faelle := map[string]string{
		"Vorlage ohne Platzhalter": "Sehr geehrte Damen und Herren,\n\nanbei die Bestellung.",
		"Vorlage mit Platzhalter":  "Bestätigung: {{.BestaetigungsLink}}",
	}
	for name, vorlage := range faelle {
		t.Run(name, func(t *testing.T) {
			_, body := resolveBestellMail("Betreff", vorlage, "K-1", 2, 5, "")

			if strings.Contains(body, "{{.BestaetigungsLink}}") {
				t.Error("unaufgelöster Platzhalter in der Mail an den Lieferanten")
			}
			if strings.Contains(body, "Etiketten wählen, drucken") {
				t.Error("Link-Absatz angehängt, obwohl es gar keinen Link gibt")
			}
		})
	}
}

// Die Adresse kommt aus einem Eingabefeld — sie kann alles Mögliche enthalten.
func TestBestaetigungsLink_AdresseWirdNormalisiert(t *testing.T) {
	faelle := []struct {
		name, basis, token, erwartet string
	}{
		{"mit Schema", "https://bib.schule.de", "ABC", "https://bib.schule.de/bestellung/ABC"},
		{"Schrägstrich am Ende", "https://bib.schule.de/", "ABC", "https://bib.schule.de/bestellung/ABC"},
		{"ohne Schema", "bib.schule.de", "ABC", "https://bib.schule.de/bestellung/ABC"},
		{"http bleibt http", "http://intern.schule", "ABC", "http://intern.schule/bestellung/ABC"},
		{"Leerzeichen", "  https://bib.schule.de  ", "ABC", "https://bib.schule.de/bestellung/ABC"},
		// Leer heißt: kein Link. Ein "/bestellung/ABC" ohne Host wäre in einer Mail wertlos
		// und sähe trotzdem nach einem echten Link aus.
		{"keine Adresse", "", "ABC", ""},
		{"kein Token", "https://bib.schule.de", "", ""},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := bestaetigungsLink(f.basis, f.token); got != f.erwartet {
				t.Errorf("bestaetigungsLink(%q, %q) = %q, want %q", f.basis, f.token, got, f.erwartet)
			}
		})
	}
}

// Zwei Bestellungen dürfen nie denselben Token bekommen, und der gespeicherte Hash darf
// den Token nicht preisgeben.
func TestBestaetigungsToken_EinmaligUndNurAlsHashGespeichert(t *testing.T) {
	tokenA, hashA, err := neuerBestaetigungsToken()
	if err != nil {
		t.Fatalf("Token erzeugen: %v", err)
	}
	tokenB, hashB, err := neuerBestaetigungsToken()
	if err != nil {
		t.Fatalf("Token erzeugen: %v", err)
	}

	if tokenA == tokenB || hashA == hashB {
		t.Fatal("zwei Aufrufe lieferten denselben Token — der Zufall funktioniert nicht")
	}
	if strings.Contains(hashA, tokenA) || hashA == tokenA {
		t.Fatal("der gespeicherte Hash enthält den Token im Klartext")
	}
	if len(hashA) != 64 {
		t.Errorf("Hashlänge = %d, erwartet 64 (SHA-256 als Hex)", len(hashA))
	}
	if hashBestaetigungsToken(tokenA) != hashA {
		t.Error("hashBestaetigungsToken liefert für denselben Token einen anderen Wert — der Lookup fände die Bestellung nie")
	}
}
