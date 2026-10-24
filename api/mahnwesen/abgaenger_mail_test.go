package mahnwesen

import (
	"bibliothek/mailservice"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"bibliothek/pdf"
	"bibliothek/repository"
)

func eintrag(vorname, nachname, klasse string) pdf.KontoauszugEintrag {
	return pdf.KontoauszugEintrag{
		Schueler: pdf.KontoauszugSchueler{Vorname: vorname, Nachname: nachname, Klasse: klasse},
		Buecher:  []pdf.KontoauszugBuch{{Titel: "Buch", Barcode: "B1"}},
	}
}

var fakeKontoauszugPDF = func(_ pdf.KontoauszugEintrag) ([]byte, error) { return []byte("%PDF-fake"), nil }

// Gruppierung ist der Kern: Aus einer flachen Liste von Abgängern werden Klassen
// mit ihrem jeweiligen Empfänger — und niemand landet in der falschen Klasse.
func TestWaehleAbgaengerKlassen(t *testing.T) {
	eintraege := []pdf.KontoauszugEintrag{
		eintrag("Max", "Mustermann", "5a"),
		eintrag("Erika", "Musterfrau", "5a"),
		eintrag("Tom", "Krause", "6c"),
	}
	adressen := map[string]string{"5a": "lehrer5a@schule.de", "6c": "lehrer6c@schule.de"}

	t.Run("ohne Auswahl keine Klasse", func(t *testing.T) {
		if gewaehlt, _ := waehleAbgaengerKlassen(eintraege, adressen, nil); len(gewaehlt) != 0 {
			t.Fatalf("gewaehlt = %+v, erwartet keine Klasse", gewaehlt)
		}
	})

	t.Run("gruppiert die gewählten Klassen mit ihrem Empfänger", func(t *testing.T) {
		gewaehlt, unbekannt := waehleAbgaengerKlassen(eintraege, adressen, []string{"5a", "6c"})
		if len(gewaehlt) != 2 || unbekannt != nil {
			t.Fatalf("gewaehlt=%d unbekannt=%v, erwartet 2 und nil", len(gewaehlt), unbekannt)
		}
		if len(gewaehlt[0].Eintraege) != 2 || gewaehlt[0].Klasse != "5a" {
			t.Fatalf("5a = %+v, erwartet zwei Abgänger", gewaehlt[0])
		}
		if gewaehlt[0].Empfaenger != "lehrer5a@schule.de" {
			t.Fatalf("Empfänger = %q, erwartet lehrer5a@schule.de", gewaehlt[0].Empfaenger)
		}
	})

	t.Run("schneidet auf die Auswahl zu und meldet Unbekanntes", func(t *testing.T) {
		auswahl := []string{"6c", "05A"}
		gewaehlt, unbekannt := waehleAbgaengerKlassen(eintraege, adressen, auswahl)
		if len(gewaehlt) != 1 || gewaehlt[0].Klasse != "6c" {
			t.Fatalf("gewaehlt = %+v, erwartet genau [6c]", gewaehlt)
		}
		if len(unbekannt) != 1 || unbekannt[0] != "05A" {
			t.Fatalf("unbekannt = %v, erwartet [05A] — ein Format-Versatz darf nicht still verpuffen", unbekannt)
		}
	})

	t.Run("doppelte Auswahl mailt nicht doppelt", func(t *testing.T) {
		auswahl := []string{"5a", "5a"}
		gewaehlt, _ := waehleAbgaengerKlassen(eintraege, adressen, auswahl)
		if len(gewaehlt) != 1 {
			t.Fatalf("gewaehlt = %d, erwartet 1 — sonst bekommt die Klassenleitung zwei Mails", len(gewaehlt))
		}
	})

	t.Run("Klasse ohne hinterlegte Adresse behält leeren Empfänger", func(t *testing.T) {
		gewaehlt, _ := waehleAbgaengerKlassen(eintraege, map[string]string{}, []string{"5a", "6c"})
		if len(gewaehlt) != 2 {
			t.Fatalf("gewaehlt = %d Klassen, erwartet 2", len(gewaehlt))
		}
		for _, k := range gewaehlt {
			if k.Empfaenger != "" {
				t.Fatalf("Empfänger = %q, erwartet leer (kein Mapping vorhanden)", k.Empfaenger)
			}
		}
	})
}

// Eine Mail je Klasse, darin ein PDF je Schüler; das ist die fachliche Zusage.
func TestVersendeAbgaengerKontoauszuege_EinPDFJeSchueler(t *testing.T) {
	var mails []mailservice.Nachricht
	klassen := []abgaengerKlasse{{
		Klasse:     "5a",
		Empfaenger: "lehrer5a@schule.de",
		Eintraege:  []pdf.KontoauszugEintrag{eintrag("Max", "Mustermann", "5a"), eintrag("Erika", "Musterfrau", "5a")},
	}}

	erg := versendeAbgaengerKontoauszuege(t.Context(), klassen, fakeKontoauszugPDF,
		func(_ context.Context, m mailservice.Nachricht) error { mails = append(mails, m); return nil })

	if erg.Versendet != 1 || erg.Uebersprungen != 0 || erg.Fehlgeschlagen != 0 {
		t.Fatalf("%+v, erwartet sent=1", erg)
	}
	if len(mails) != 1 {
		t.Fatalf("%d Mails, erwartet genau 1 je Klasse", len(mails))
	}
	if len(mails[0].Anhaenge) != 2 {
		t.Fatalf("%d Anhänge, erwartet 2 (ein PDF je Schüler)", len(mails[0].Anhaenge))
	}
	if mails[0].Anhaenge[0].Name != "Kontoauszug_Max_Mustermann.pdf" {
		t.Fatalf("Anhangname = %q — Leerzeichen im Namen machen den Anhang unhandlich", mails[0].Anhaenge[0].Name)
	}
	if !strings.Contains(mails[0].Betreff, "5a") {
		t.Fatalf("Betreff nennt die Klasse nicht: %q", mails[0].Betreff)
	}
}

// Klassen ohne Adresse oder ohne offene Ausleihen bekommen keine Mail.
func TestVersendeAbgaengerKontoauszuege_SkipLogik(t *testing.T) {
	var empfaenger []string
	klassen := []abgaengerKlasse{
		{Klasse: "5a", Empfaenger: "lehrer5a@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Max", "M", "5a")}},
		{Klasse: "6b", Empfaenger: "", Eintraege: []pdf.KontoauszugEintrag{eintrag("Tom", "K", "6b")}},
		{Klasse: "7c", Empfaenger: "lehrer7c@schule.de", Eintraege: nil},
	}

	erg := versendeAbgaengerKontoauszuege(t.Context(), klassen, fakeKontoauszugPDF,
		func(_ context.Context, m mailservice.Nachricht) error {
			empfaenger = append(empfaenger, m.An)
			return nil
		})

	if erg.Versendet != 1 || erg.Uebersprungen != 2 || erg.Fehlgeschlagen != 0 {
		t.Fatalf("sent=%d skipped=%d failed=%d, erwartet 1/2/0", erg.Versendet, erg.Uebersprungen, erg.Fehlgeschlagen)
	}
	if len(empfaenger) != 1 || empfaenger[0] != "lehrer5a@schule.de" {
		t.Fatalf("Empfänger = %v, erwartet nur lehrer5a@schule.de", empfaenger)
	}
}

// Ein kaputtes Schüler-PDF darf nicht die ganze Klasse kosten: Der Rest geht raus,
// nur der eine Anhang fehlt. Fällt jedes PDF aus, ist die Klasse fehlgeschlagen, nicht
// „übersprungen": Das wäre Absicht.
func TestVersendeAbgaengerKontoauszuege_EinzelnesPDFScheitert(t *testing.T) {
	kaputt := func(e pdf.KontoauszugEintrag) ([]byte, error) {
		if e.Schueler.Vorname == "Kaputt" {
			return nil, errors.New("pdf kaputt")
		}
		return []byte("%PDF-fake"), nil
	}

	var mails []mailservice.Nachricht
	sammeln := func(_ context.Context, m mailservice.Nachricht) error { mails = append(mails, m); return nil }

	erg := versendeAbgaengerKontoauszuege(t.Context(), []abgaengerKlasse{{
		Klasse:     "5a",
		Empfaenger: "lehrer5a@schule.de",
		Eintraege:  []pdf.KontoauszugEintrag{eintrag("Kaputt", "Fall", "5a"), eintrag("Heile", "Welt", "5a")},
	}}, kaputt, sammeln)

	if erg.Versendet != 1 || erg.Uebersprungen != 0 || erg.Fehlgeschlagen != 0 {
		t.Fatalf("%+v, erwartet sent=1 — ein kaputtes PDF darf die Klasse nicht blockieren", erg)
	}
	if len(mails[0].Anhaenge) != 1 {
		t.Fatalf("%d Anhänge, erwartet 1 (nur der heile Kontoauszug)", len(mails[0].Anhaenge))
	}

	mails = nil
	erg = versendeAbgaengerKontoauszuege(t.Context(), []abgaengerKlasse{{
		Klasse:     "6b",
		Empfaenger: "lehrer6b@schule.de",
		Eintraege:  []pdf.KontoauszugEintrag{eintrag("Kaputt", "Fall", "6b")},
	}}, kaputt, sammeln)

	if erg.Versendet != 0 || erg.Uebersprungen != 0 || erg.Fehlgeschlagen != 1 {
		t.Fatalf("%+v, erwartet failed=1 — alle PDFs kaputt ist ein Problem, keine Absicht; und ohne Anhang keine leere Mail", erg)
	}
	if len(mails) != 0 {
		t.Fatalf("%d Mails verschickt, erwartet 0 — eine Mail ohne Kontoauszug ist nur Rauschen", len(mails))
	}
}

// Mit Umleitung gehen alle gewählten Klassen an diese eine Adresse — und die
// geladenen Daten bleiben unangetastet.
func TestLenkeAbgaengerUm(t *testing.T) {
	original := []abgaengerKlasse{
		{Klasse: "5a", Empfaenger: "lehrer5a@schule.de"},
		{Klasse: "6b", Empfaenger: ""},
	}
	umgelenkt := lenkeAbgaengerUm(original, "sekretariat@schule.de")

	for _, k := range umgelenkt {
		if k.Empfaenger != "sekretariat@schule.de" {
			t.Fatalf("Klasse %s → %q, erwartet sekretariat@schule.de", k.Klasse, k.Empfaenger)
		}
	}
	if original[0].Empfaenger != "lehrer5a@schule.de" {
		t.Fatalf("Original mutiert (%q)", original[0].Empfaenger)
	}
}

// Rückschritt: Die Klassenleitungen pflegen Menschen, die Klassen kommen aus dem
// LUSD-Import. „5A" gegen „5a" oder ein mitkopiertes Leerzeichen darf den Versand
// nicht kosten — vorher meldete die Oberfläche „keine E-Mail" und der Lauf
// übersprang die Klasse still, obwohl die Adresse hinterlegt war.
func TestWaehleAbgaengerKlassen_SchreibweiseDerKlassenleitungEgal(t *testing.T) {
	eintraege := []pdf.KontoauszugEintrag{eintrag("Max", "Mustermann", "5a")}
	adressen := map[string]string{
		repository.KlassenSchluessel(" 5A "): "nberger@philipp-reis-schule.de",
	}

	gewaehlt, _ := waehleAbgaengerKlassen(eintraege, adressen, []string{"5a"})

	if len(gewaehlt) != 1 || gewaehlt[0].Empfaenger != "nberger@philipp-reis-schule.de" {
		t.Fatalf("Empfänger = %q, erwartet nberger@philipp-reis-schule.de (Schreibweise darf egal sein)", gewaehlt[0].Empfaenger)
	}
}

// Ein SMTP-Ausfall ist fehlgeschlagen (nicht „übersprungen") und bricht den Lauf ab, dieselbe
// Regel wie beim Mahnlauf; als übersprungen erklärte ihn die Rückmeldung mit „keine E-Mail
// hinterlegt".
func TestVersendeAbgaengerKontoauszuege_MailausfallIstFehlgeschlagen(t *testing.T) {
	klassen := []abgaengerKlasse{
		{Klasse: "10a", Empfaenger: "kl10a@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Max", "M", "10a")}},
		{Klasse: "10b", Empfaenger: "kl10b@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Mia", "K", "10b")}},
	}

	// Einzelfehler (abgelehnte Adresse): failed, kein Abbruch.
	erg := versendeAbgaengerKontoauszuege(t.Context(), klassen, fakeKontoauszugPDF,
		func(_ context.Context, m mailservice.Nachricht) error {
			if m.An == "kl10a@schule.de" {
				return errors.New("550 mailbox unavailable")
			}
			return nil
		})
	if erg.Versendet != 1 || erg.Fehlgeschlagen != 1 || erg.Abgebrochen {
		t.Fatalf("%+v, erwartet sent=1 failed=1 ohne Abbruch", erg)
	}

	// Totes Relay (ErrSMTPVersand): Abbruch, die restlichen zählen als fehlgeschlagen.
	erg = versendeAbgaengerKontoauszuege(t.Context(), klassen, fakeKontoauszugPDF,
		func(context.Context, mailservice.Nachricht) error {
			return fmt.Errorf("%w: dial tcp: timeout", mailservice.ErrSMTPVersand)
		})
	if !erg.Abgebrochen || erg.Fehlgeschlagen != 2 || erg.Versendet != 0 {
		t.Fatalf("%+v, erwartet Abbruch mit failed=2", erg)
	}
}

// Nach dem Abbruch zählt der Lauf wie der Mahnlauf weiter: Eine Klasse ohne Adresse oder ohne
// Auszüge ist übersprungen, jede andere fehlgeschlagen, und für keine entsteht noch ein PDF.
// Zählten alle restlichen als fehlgeschlagen, stünde in der Rückmeldung eine Klasse zum
// Nachsenden, für die es nichts zu senden gibt.
func TestVersendeAbgaengerKontoauszuege_NachAbbruchBleibtUebersprungenUebersprungen(t *testing.T) {
	klassen := []abgaengerKlasse{
		{Klasse: "10a", Empfaenger: "kl10a@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Max", "M", "10a")}},
		{Klasse: "10b", Empfaenger: "kl10b@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Mia", "K", "10b")}},
		{Klasse: "10c", Empfaenger: "", Eintraege: []pdf.KontoauszugEintrag{eintrag("Ole", "P", "10c")}},
		{Klasse: "10d", Empfaenger: "kl10d@schule.de", Eintraege: nil},
		{Klasse: "10e", Empfaenger: "kl10e@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Ida", "S", "10e")}},
	}
	pdfs := 0
	erg := versendeAbgaengerKontoauszuege(t.Context(), klassen,
		func(e pdf.KontoauszugEintrag) ([]byte, error) {
			pdfs++
			return fakeKontoauszugPDF(e)
		},
		func(_ context.Context, m mailservice.Nachricht) error {
			if m.An == "kl10b@schule.de" {
				return fmt.Errorf("%w: dial tcp: timeout", mailservice.ErrSMTPVersand)
			}
			return nil
		})
	if erg.Versendet != 1 || erg.Fehlgeschlagen != 2 || erg.Uebersprungen != 2 || !erg.Abgebrochen {
		t.Errorf("%+v, erwartet sent=1 failed=2 (10b und die nicht versuchte 10e) skipped=2 (10c, 10d) abgebrochen", erg)
	}
	if pdfs != 2 {
		t.Errorf("%d PDFs erzeugt, erwartet 2: nach dem Abbruch entsteht keins mehr", pdfs)
	}
	if msg := abgaengerVersandMeldung(erg, ""); !strings.Contains(msg, "2 übersprungen") || !strings.Contains(msg, "Für 2 Klassen nicht zugestellt") {
		t.Errorf("die Rückmeldung nennt die Zahlen nicht: %s", msg)
	}
}

// Die Rückmeldung eines Versands ist ein Satz ohne Punkt, mit Ausfall oder Abbruch mehrere mit
// Punkt; die Zahl der Klassen steht in Einzahl oder Mehrzahl, ohne Klammer.
func TestVersandMeldung_Saetze(t *testing.T) {
	faelle := []struct {
		name string
		ist  string
		soll string
	}{
		{"Kontoauszüge, eine Klasse", abgaengerVersandMeldung(versandErgebnis{Versendet: 1}, ""),
			"Kontoauszüge von 1 Klasse an die Klassenleitungen versendet, 0 übersprungen (keine E-Mail hinterlegt oder keine offenen Ausleihen)"},
		{"Mahnlisten mit Umleitung und Abbruch",
			mahnlistenMeldung(versandErgebnis{Versendet: 3, Uebersprungen: 1, Fehlgeschlagen: 1, Abgebrochen: true}, "test@schule.de"),
			"Mahnlisten von 3 Klassen an test@schule.de versendet, 1 übersprungen (keine E-Mail hinterlegt oder keine Fälle). " +
				"Für 1 Klasse nicht zugestellt. Der Lauf brach nach einem Fehler des Mailservers ab. Bitte den Mailserver " +
				"prüfen und erneut senden. Keine Klasse wird doppelt gemahnt, weil der Mailversand die Mahnstufe nicht erhöht."},
		{"nichts zu versenden", mahnlistenMeldung(versandErgebnis{}, ""), "Keine der gewählten Klassen hat noch überfällige Fälle. Nichts versendet."},
	}
	for _, f := range faelle {
		if f.ist != f.soll {
			t.Errorf("%s:\n  ist  %q\n  soll %q", f.name, f.ist, f.soll)
		}
	}
}
