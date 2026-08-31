package api

import (
	"bibliothek/mailservice"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
	"bibliothek/pdf"
	"bibliothek/repository"

	"github.com/pashagolub/pgxmock/v4"
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

	t.Run("nil = alle Klassen", func(t *testing.T) {
		gewaehlt, unbekannt := waehleAbgaengerKlassen(eintraege, adressen, nil)
		if len(gewaehlt) != 2 || unbekannt != nil {
			t.Fatalf("gewaehlt=%d unbekannt=%v, want 2 und nil", len(gewaehlt), unbekannt)
		}
		if len(gewaehlt[0].Eintraege) != 2 || gewaehlt[0].Klasse != "5a" {
			t.Fatalf("5a = %+v, want zwei Abgänger", gewaehlt[0])
		}
		if gewaehlt[0].Empfaenger != "lehrer5a@schule.de" {
			t.Fatalf("Empfänger = %q, want lehrer5a@schule.de", gewaehlt[0].Empfaenger)
		}
	})

	t.Run("schneidet auf die Auswahl zu und meldet Unbekanntes", func(t *testing.T) {
		auswahl := []string{"6c", "05A"}
		gewaehlt, unbekannt := waehleAbgaengerKlassen(eintraege, adressen, &auswahl)
		if len(gewaehlt) != 1 || gewaehlt[0].Klasse != "6c" {
			t.Fatalf("gewaehlt = %+v, want genau [6c]", gewaehlt)
		}
		if len(unbekannt) != 1 || unbekannt[0] != "05A" {
			t.Fatalf("unbekannt = %v, want [05A] — ein Format-Versatz darf nicht still verpuffen", unbekannt)
		}
	})

	t.Run("doppelte Auswahl mailt nicht doppelt", func(t *testing.T) {
		auswahl := []string{"5a", "5a"}
		gewaehlt, _ := waehleAbgaengerKlassen(eintraege, adressen, &auswahl)
		if len(gewaehlt) != 1 {
			t.Fatalf("gewaehlt = %d, want 1 — sonst bekommt die Klassenleitung zwei Mails", len(gewaehlt))
		}
	})

	t.Run("Klasse ohne hinterlegte Adresse behält leeren Empfänger", func(t *testing.T) {
		gewaehlt, _ := waehleAbgaengerKlassen(eintraege, map[string]string{}, nil)
		for _, k := range gewaehlt {
			if k.Empfaenger != "" {
				t.Fatalf("Empfänger = %q, want leer (kein Mapping vorhanden)", k.Empfaenger)
			}
		}
	})
}

// Eine Mail je Klasse, darin EIN PDF JE SCHÜLER — das ist die fachliche Zusage.
func TestVersendeAbgaengerKontoauszuege_EinPdfJeSchueler(t *testing.T) {
	var mails []MailRequest
	klassen := []abgaengerKlasse{{
		Klasse:     "5a",
		Empfaenger: "lehrer5a@schule.de",
		Eintraege:  []pdf.KontoauszugEintrag{eintrag("Max", "Mustermann", "5a"), eintrag("Erika", "Musterfrau", "5a")},
	}}

	erg := versendeAbgaengerKontoauszuege(klassen, fakeKontoauszugPDF,
		func(m MailRequest) error { mails = append(mails, m); return nil })

	if erg.Sent != 1 || erg.Skipped != 0 || erg.Failed != 0 {
		t.Fatalf("%+v, want sent=1", erg)
	}
	if len(mails) != 1 {
		t.Fatalf("%d Mails, want genau 1 je Klasse", len(mails))
	}
	if len(mails[0].Attachments) != 2 {
		t.Fatalf("%d Anhänge, want 2 (ein PDF je Schüler)", len(mails[0].Attachments))
	}
	if mails[0].Attachments[0].Name != "Kontoauszug_Max_Mustermann.pdf" {
		t.Fatalf("Anhangname = %q — Leerzeichen im Namen machen den Anhang unhandlich", mails[0].Attachments[0].Name)
	}
	if !strings.Contains(mails[0].Subject, "5a") {
		t.Fatalf("Betreff nennt die Klasse nicht: %q", mails[0].Subject)
	}
}

// Klassen ohne Adresse oder ohne offene Ausleihen bekommen KEINE Mail.
func TestVersendeAbgaengerKontoauszuege_SkipLogik(t *testing.T) {
	var empfaenger []string
	klassen := []abgaengerKlasse{
		{Klasse: "5a", Empfaenger: "lehrer5a@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Max", "M", "5a")}},
		{Klasse: "6b", Empfaenger: "", Eintraege: []pdf.KontoauszugEintrag{eintrag("Tom", "K", "6b")}},
		{Klasse: "7c", Empfaenger: "lehrer7c@schule.de", Eintraege: nil},
	}

	erg := versendeAbgaengerKontoauszuege(klassen, fakeKontoauszugPDF,
		func(m MailRequest) error { empfaenger = append(empfaenger, m.To); return nil })

	if erg.Sent != 1 || erg.Skipped != 2 || erg.Failed != 0 {
		t.Fatalf("sent=%d skipped=%d failed=%d, want 1/2/0", erg.Sent, erg.Skipped, erg.Failed)
	}
	if len(empfaenger) != 1 || empfaenger[0] != "lehrer5a@schule.de" {
		t.Fatalf("Empfänger = %v, want nur lehrer5a@schule.de", empfaenger)
	}
}

// Ein kaputtes Schüler-PDF darf nicht die ganze Klasse kosten: Der Rest geht raus,
// nur der eine Anhang fehlt. Fällt JEDES PDF aus, ist die Klasse FEHLGESCHLAGEN —
// bis zum 31.08.2026 zählte sie als „übersprungen", also als Absicht.
func TestVersendeAbgaengerKontoauszuege_EinzelnesPdfScheitert(t *testing.T) {
	kaputt := func(e pdf.KontoauszugEintrag) ([]byte, error) {
		if e.Schueler.Vorname == "Kaputt" {
			return nil, errors.New("pdf kaputt")
		}
		return []byte("%PDF-fake"), nil
	}

	var mails []MailRequest
	sammeln := func(m MailRequest) error { mails = append(mails, m); return nil }

	erg := versendeAbgaengerKontoauszuege([]abgaengerKlasse{{
		Klasse:     "5a",
		Empfaenger: "lehrer5a@schule.de",
		Eintraege:  []pdf.KontoauszugEintrag{eintrag("Kaputt", "Fall", "5a"), eintrag("Heile", "Welt", "5a")},
	}}, kaputt, sammeln)

	if erg.Sent != 1 || erg.Skipped != 0 || erg.Failed != 0 {
		t.Fatalf("%+v, want sent=1 — ein kaputtes PDF darf die Klasse nicht blockieren", erg)
	}
	if len(mails[0].Attachments) != 1 {
		t.Fatalf("%d Anhänge, want 1 (nur der heile Kontoauszug)", len(mails[0].Attachments))
	}

	mails = nil
	erg = versendeAbgaengerKontoauszuege([]abgaengerKlasse{{
		Klasse:     "6b",
		Empfaenger: "lehrer6b@schule.de",
		Eintraege:  []pdf.KontoauszugEintrag{eintrag("Kaputt", "Fall", "6b")},
	}}, kaputt, sammeln)

	if erg.Sent != 0 || erg.Skipped != 0 || erg.Failed != 1 {
		t.Fatalf("%+v, want failed=1 — alle PDFs kaputt ist ein Problem, keine Absicht; und ohne Anhang keine leere Mail", erg)
	}
	if len(mails) != 0 {
		t.Fatalf("%d Mails verschickt, want 0 — eine Mail ohne Kontoauszug ist nur Rauschen", len(mails))
	}
}

// Mit Override gehen ALLE gewählten Klassen an diese eine Adresse — und die
// geladenen Daten bleiben unangetastet.
func TestLenkeAbgaengerAufOverride(t *testing.T) {
	original := []abgaengerKlasse{
		{Klasse: "5a", Empfaenger: "lehrer5a@schule.de"},
		{Klasse: "6b", Empfaenger: ""},
	}
	umgelenkt := lenkeAbgaengerAufOverride(original, "sekretariat@schule.de")

	for _, k := range umgelenkt {
		if k.Empfaenger != "sekretariat@schule.de" {
			t.Fatalf("Klasse %s → %q, want sekretariat@schule.de", k.Klasse, k.Empfaenger)
		}
	}
	if original[0].Empfaenger != "lehrer5a@schule.de" {
		t.Fatalf("Original mutiert (%q)", original[0].Empfaenger)
	}
}

// Eine leere Auswahl heißt „niemand": abgewiesen, bevor eine Query läuft.
func TestSendAbgaengerKontoauszuege_LeereAuswahlOhneDBZugriff(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	server := &Server{DB: &db.Database{Pool: mock}}
	req := httptest.NewRequest(http.MethodPost, "/api/abgaenger/mail", strings.NewReader(`{"klassen":[]}`))
	rec := httptest.NewRecorder()
	server.SendAbgaengerKontoauszuegeHandler()(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekam %d: %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unerwarteter DB-Zugriff: %v", err)
	}
}

// Regression: Das Mapping wird von Menschen gepflegt, die Klassen kommen aus dem
// LUSD-Import. „5A" gegen „5a" oder ein mitkopiertes Leerzeichen darf den Versand
// nicht kosten — vorher meldete die Oberfläche „keine E-Mail" und der Lauf
// übersprang die Klasse still, obwohl die Adresse hinterlegt war.
func TestWaehleAbgaengerKlassen_SchreibweiseDesMappingsEgal(t *testing.T) {
	eintraege := []pdf.KontoauszugEintrag{eintrag("Max", "Mustermann", "5a")}
	adressen := map[string]string{
		repository.KlassenSchluessel(" 5A "): "pflasch@philipp-reis-schule.de",
	}

	gewaehlt, _ := waehleAbgaengerKlassen(eintraege, adressen, nil)

	if len(gewaehlt) != 1 || gewaehlt[0].Empfaenger != "pflasch@philipp-reis-schule.de" {
		t.Fatalf("Empfänger = %q, want pflasch@philipp-reis-schule.de (Schreibweise darf egal sein)", gewaehlt[0].Empfaenger)
	}
}

// SMTP-Ausfall ist FEHLGESCHLAGEN (nicht „übersprungen") und bricht den Lauf ab —
// dieselbe Regel wie beim Mahnlauf. Bis zum 31.08.2026 zählte er als übersprungen,
// und die Rückmeldung erklärte ihn als „keine E-Mail hinterlegt".
func TestVersendeAbgaengerKontoauszuege_MailausfallIstFehlgeschlagen(t *testing.T) {
	klassen := []abgaengerKlasse{
		{Klasse: "10a", Empfaenger: "kl10a@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Max", "M", "10a")}},
		{Klasse: "10b", Empfaenger: "kl10b@schule.de", Eintraege: []pdf.KontoauszugEintrag{eintrag("Mia", "K", "10b")}},
	}

	// Einzelfehler (abgelehnte Adresse): failed, kein Abbruch.
	erg := versendeAbgaengerKontoauszuege(klassen, fakeKontoauszugPDF,
		func(m MailRequest) error {
			if m.To == "kl10a@schule.de" {
				return errors.New("550 mailbox unavailable")
			}
			return nil
		})
	if erg.Sent != 1 || erg.Failed != 1 || erg.Abgebrochen {
		t.Fatalf("%+v, want sent=1 failed=1 ohne Abbruch", erg)
	}

	// Totes Relay (ErrSMTPVersand): Abbruch, die restlichen zählen als fehlgeschlagen.
	erg = versendeAbgaengerKontoauszuege(klassen, fakeKontoauszugPDF,
		func(MailRequest) error { return fmt.Errorf("%w: dial tcp: timeout", mailservice.ErrSMTPVersand) })
	if !erg.Abgebrochen || erg.Failed != 2 || erg.Sent != 0 {
		t.Fatalf("%+v, want Abbruch mit failed=2", erg)
	}
}
