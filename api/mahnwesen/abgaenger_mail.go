package mahnwesen

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/mailservice"
	"bibliothek/pdf"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// abgaengerKlasse bündelt die Kontoauszüge einer Abgangsklasse mit ihrem Empfänger.
type abgaengerKlasse struct {
	Klasse     string
	Empfaenger string
	Eintraege  []pdf.KontoauszugEintrag
}

// AbgaengerKontoauszuegeVersenden mailt die Kontoauszüge der gewählten
// Abgangsklassen an die jeweilige Klassenleitung — eine E-Mail je Klasse, darin ein
// PDF je Abgänger.
//
// Warum ein PDF je Schüler und kein Sammel-Dokument: Die Klassenleitung gibt die
// Blätter einzeln weiter. Ein Sammel-PDF müsste dafür erst zerschnitten werden.
//
// Datenschutz wie im Mahnwesen: Es wird an die Lehrkraft gemailt, nicht an die
// (meist minderjährigen) Schüler, und jede Lehrkraft bekommt ausschließlich die
// eigene Klasse — kein TO/CC über mehrere Betroffene. override_email ist auch hier
// die einzige Abweichung und steht deshalb im Klartext im Protokoll.
//
// Ohne Sperre in den Ferien, anders als der Mahnlauf: Der Schulabgang fällt regelmäßig in die
// Sommerferien, und eine Sperre blockierte die Rückgabe genau dann, wenn sie stattfindet. Der
// Kontoauszug ist auch keine Mahnung, sondern die Übersicht eines beendeten Ausleihkontos.
//
// POST /api/abgaenger/mail
func (t *Tueren) AbgaengerKontoauszuegeVersenden() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		req, err := leseKlassenVersandAnfrage(r, t.Post().Konfig)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}

		if !t.Post().IstKonfiguriert() {
			apierrors.AntworteFehler(w, http.StatusServiceUnavailable, fmt.Errorf("SMTP nicht konfiguriert – Versand nicht möglich"))
			return
		}

		// Dieselbe Saison wie Liste und Druck: Außerhalb gäbe es nichts zu versenden —
		// und ein Lauf im Oktober schickte den Klassenleitungen die laufende Ausleihe.
		if fenster := abgaengerFensterFuer(t.Jetzt()); !fenster.Offen {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldungAusserhalbDerSaison(fenster), nil)
			return
		}

		// Dieselbe Abfrage, die auch der Druck benutzt (leerer Filter = alle Klassen).
		// Papier und Mail zeigen damit garantiert denselben Stand.
		eintraege, err := t.ladeAbgaengerKontoauszuege(ctx, "", nil)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		adressen, err := t.ladeKlassenleitungsAdressen(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		gewaehlt, unbekannt := waehleAbgaengerKlassen(eintraege, adressen, req.Klassen)
		if req.Umleitung != "" {
			gewaehlt = lenkeAbgaengerUm(gewaehlt, req.Umleitung)
		}

		audit := klassenVersandProtokoll{
			Klassen:     abgaengerKlassennamen(gewaehlt),
			AlleKlassen: req.Klassen == nil,
			Umleitung:   req.Umleitung,
			Unbekannt:   unbekannt,
		}

		// Absicht vor dem Versand protokollieren, siehe protokolliereKlassenVersand.
		audit.Phase = "start"
		t.protokolliereKlassenVersand(r, "ABGAENGER_KONTOAUSZUG_MAIL", audit)

		erg := versendeAbgaengerKontoauszuege(gewaehlt, erzeugeKontoauszugPDF, t.Post().Sende)

		audit.Phase = "ende"
		audit.Versendet, audit.Uebersprungen, audit.Fehlgeschlagen, audit.Abgebrochen = erg.Versendet, erg.Uebersprungen, erg.Fehlgeschlagen, erg.Abgebrochen
		t.protokolliereKlassenVersand(r, "ABGAENGER_KONTOAUSZUG_MAIL", audit)

		tuer.AntworteJSON(w, http.StatusOK, klassenVersandAntwort{
			Versendet:      erg.Versendet,
			Uebersprungen:  erg.Uebersprungen,
			Fehlgeschlagen: erg.Fehlgeschlagen,
			Abgebrochen:    erg.Abgebrochen,
			Meldung:        abgaengerVersandMeldung(erg, req.Umleitung),
		})
	}
}

// ladeKlassenleitungsAdressen lädt die Klassenleitungen als Nachschlagetabelle.
//
// Anders als bei der Mahnliste bricht ein Fehler hier den Versand ab: Ohne die Tabelle wäre
// jede Klasse „ohne Adresse", und der Lauf meldete „0 versendet, 12 übersprungen", ein stiller
// Ausfall, der wie ein Ergebnis aussieht.
func (t *Tueren) ladeKlassenleitungsAdressen(ctx context.Context) (map[string]string, error) {
	return repository.LadeKlassenleitungsAdressen(ctx, t.DB.Pool)
}

// waehleAbgaengerKlassen gruppiert die Kontoauszüge nach Klasse, hängt den
// Empfänger an und schneidet auf die Auswahl zu. Unbekannte Klassen werden
// zurückgemeldet statt verschluckt (sonst wäre ein Format-Versatz ein stiller
// Nullversand).
func waehleAbgaengerKlassen(
	eintraege []pdf.KontoauszugEintrag,
	adressen map[string]string,
	auswahl *[]string,
) (gewaehlt []abgaengerKlasse, unbekannt []string) {
	// Reihenfolge der Klassen folgt der Abfrage (nach Klasse sortiert), damit
	// Protokoll und Rückmeldung stabil bleiben.
	nachKlasse := map[string]*abgaengerKlasse{}
	reihenfolge := make([]string, 0)
	for _, e := range eintraege {
		k := e.Schueler.Klasse
		if _, ok := nachKlasse[k]; !ok {
			nachKlasse[k] = &abgaengerKlasse{Klasse: k, Empfaenger: adressen[repository.KlassenSchluessel(k)]}
			reihenfolge = append(reihenfolge, k)
		}
		nachKlasse[k].Eintraege = append(nachKlasse[k].Eintraege, e)
	}

	if auswahl == nil {
		for _, k := range reihenfolge {
			gewaehlt = append(gewaehlt, *nachKlasse[k])
		}
		return gewaehlt, nil
	}

	gesehen := map[string]bool{}
	for _, name := range *auswahl {
		if gesehen[name] {
			continue // doppelte Auswahl darf nicht doppelt mailen
		}
		gesehen[name] = true

		if k, ok := nachKlasse[name]; ok {
			gewaehlt = append(gewaehlt, *k)
		} else {
			unbekannt = append(unbekannt, name)
		}
	}
	return gewaehlt, unbekannt
}

// lenkeAbgaengerUm setzt den Empfänger aller gewählten Klassen auf eine
// Adresse. Kopie statt Mutation — die geladenen Daten bleiben unangetastet.
func lenkeAbgaengerUm(klassen []abgaengerKlasse, empfaenger string) []abgaengerKlasse {
	umgelenkt := make([]abgaengerKlasse, len(klassen))
	copy(umgelenkt, klassen)
	for i := range umgelenkt {
		umgelenkt[i].Empfaenger = empfaenger
	}
	return umgelenkt
}

func abgaengerKlassennamen(klassen []abgaengerKlasse) []string {
	namen := make([]string, 0, len(klassen))
	for _, k := range klassen {
		namen = append(namen, k.Klasse)
	}
	return namen
}

// erzeugeKontoauszugPDF erzeugt den Kontoauszug eines Abgängers mit Unterschriftszeile, also
// genau das Blatt, das auch aus dem Druck fällt.
func erzeugeKontoauszugPDF(e pdf.KontoauszugEintrag) ([]byte, error) {
	return pdf.ErzeugeKontoauszuege([]pdf.KontoauszugEintrag{e}, true)
}

// versendeAbgaengerKontoauszuege verschickt je Klasse eine Mail mit einem PDF je Abgänger.
// erzeugePDF und sendeMail kommen als Parameter, damit sich das Überspringen und die
// Adressierung ohne echten PDF- und Mailversand prüfen lassen.
//
// Best effort auf zwei Ebenen: Ein gescheitertes PDF kostet nur diesen einen Anhang, nicht die
// ganze Klasse; 29 fertige Kontoauszüge wegen eines kaputten zurückzuhalten wäre der teurere
// Fehler. Drei Ausgänge: „übersprungen" ist Absicht (keine Adresse, keine Auszüge),
// „fehlgeschlagen" ist ein Problem (alle PDFs kaputt, Versand abgelehnt); als „übersprungen"
// erklärt, sähe ein toter Mailserver am Zeugnistag aus wie Absicht, und niemand sendete nach.
// Ein Mailserver-Fehler (ErrSMTPVersand) bricht ab wie beim Mahnlauf, weil jede weitere Klasse
// sonst bis zu 70 s am toten Relay hinge; danach zählt eine Klasse ohne Adresse oder ohne
// Auszüge als übersprungen, jede andere als fehlgeschlagen. Kontoauszüge doppelt zu senden ist
// unschädlich, der Lauf darf wiederholt werden.
func versendeAbgaengerKontoauszuege(
	klassen []abgaengerKlasse,
	erzeugePDF func(pdf.KontoauszugEintrag) ([]byte, error),
	sendeMail func(mailservice.Nachricht) error,
) versandErgebnis {
	var erg versandErgebnis
	for _, kl := range klassen {
		if kl.Empfaenger == "" || len(kl.Eintraege) == 0 {
			erg.Uebersprungen++
			continue
		}
		if erg.Abgebrochen {
			erg.Fehlgeschlagen++
			continue
		}

		anhaenge := baueKontoauszugAnhaenge(kl, erzeugePDF)
		if len(anhaenge) == 0 {
			// Jedes einzelne PDF dieser Klasse ist gescheitert — das ist ein Problem,
			// keine Absicht.
			erg.Fehlgeschlagen++
			continue
		}

		if err := sendeMail(baueAbgaengerMail(kl, anhaenge)); err != nil {
			log.Printf("abgaenger-kontoauszug: Versand an Klasse %s (%s) fehlgeschlagen: %v", kl.Klasse, kl.Empfaenger, err)
			erg.Fehlgeschlagen++
			if errors.Is(err, mailservice.ErrSMTPVersand) {
				erg.Abgebrochen = true
			}
			continue
		}
		erg.Versendet++
	}
	return erg
}

func baueKontoauszugAnhaenge(
	kl abgaengerKlasse,
	erzeugePDF func(pdf.KontoauszugEintrag) ([]byte, error),
) []mailservice.Anhang {
	// Leerzeichen und Schrägstriche aus dem Namen halten — ein Anhang „Kontoauszug_Max
	// Mustermann.pdf" ist beim Weiterreichen unhandlich, ein „/" darin sogar kaputt.
	sauber := strings.NewReplacer(" ", "_", "/", "-", "\\", "-")

	anhaenge := make([]mailservice.Anhang, 0, len(kl.Eintraege))
	for _, e := range kl.Eintraege {
		pdfDaten, err := erzeugePDF(e)
		if err != nil {
			log.Printf("abgaenger-kontoauszug: PDF für %s %s (%s) fehlgeschlagen: %v",
				e.Schueler.Vorname, e.Schueler.Nachname, kl.Klasse, err)
			continue
		}
		anhaenge = append(anhaenge, mailservice.Anhang{
			Name:        sauber.Replace(fmt.Sprintf("Kontoauszug_%s_%s.pdf", e.Schueler.Vorname, e.Schueler.Nachname)),
			ContentType: tuer.ContentTypePDF,
			Inhalt:      pdfDaten,
		})
	}
	return anhaenge
}

func baueAbgaengerMail(kl abgaengerKlasse, anhaenge []mailservice.Anhang) mailservice.Nachricht {
	text := fmt.Sprintf(
		"Sehr geehrte Damen und Herren,\n\n"+
			"anbei erhalten Sie die Kontoauszüge der Abgänger der Klasse %s (Stand: %s).\n\n"+
			"Betroffene Schüler/innen: %d\n\n"+
			"Je Schüler/in liegt ein Blatt mit den noch nicht zurückgegebenen Medien bei.\n"+
			"Bitte geben Sie die Blätter aus; die Freigabezeile wird nach der Rückgabe\n"+
			"in der Bibliothek abgezeichnet.\n\n"+
			"Mit freundlichen Grüßen,\nSchulbibliothek",
		kl.Klasse,
		schulzeit.Jetzt().Format(tuer.DatumsformatDE),
		len(anhaenge),
	)

	return mailservice.Nachricht{
		An:       kl.Empfaenger,
		Betreff:  fmt.Sprintf("Kontoauszüge Abgänger – Klasse %s – %s", kl.Klasse, schulzeit.Jetzt().Format(tuer.DatumsformatDE)),
		Text:     text,
		Anhaenge: anhaenge,
	}
}

// abgaengerVersandMeldung nennt den Empfänger mit: „12 versendet" allein verrät
// nicht, ob die Auszüge an die Klassenleitungen oder an eine handgetippte Adresse
// gingen.
func abgaengerVersandMeldung(erg versandErgebnis, umleitung string) string {
	if erg.Versendet == 0 && erg.Uebersprungen == 0 && erg.Fehlgeschlagen == 0 {
		return "Keine der gewählten Klassen hat noch offene Ausleihen. Nichts versendet."
	}
	return alsMeldung(append([]string{fmt.Sprintf(
		"Kontoauszüge von %s %s versendet, %d übersprungen (keine E-Mail hinterlegt oder keine offenen Ausleihen)",
		klassen(erg.Versendet), versandZiel(umleitung), erg.Uebersprungen)},
		versandAusfall(erg, "Doppelte Kontoauszüge schaden nicht")...)...)
}
