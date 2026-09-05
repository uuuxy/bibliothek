package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/mailservice"
	"bibliothek/pdf"
	"bibliothek/repository"
)

// abgaengerKlasse bündelt die Kontoauszüge einer Abgangsklasse mit ihrem Empfänger.
type abgaengerKlasse struct {
	Klasse     string
	Empfaenger string
	Eintraege  []pdf.KontoauszugEintrag
}

// SendAbgaengerKontoauszuegeHandler mailt die Kontoauszüge der gewählten
// Abgangsklassen an die jeweilige Klassenleitung — eine E-Mail je Klasse, darin ein
// PDF je Abgänger.
//
// Warum ein PDF je Schüler und kein Sammel-Dokument: Die Klassenleitung gibt die
// Blätter einzeln weiter. Ein Sammel-PDF müsste dafür erst zerschnitten werden.
//
// Datenschutz wie im Mahnwesen: Es wird an die Lehrkraft gemailt, nicht an die
// (meist minderjährigen) Schüler, und jede Lehrkraft bekommt ausschließlich die
// eigene Klasse — kein TO/CC über mehrere Betroffene. override_email ist auch hier
// die einzige Abweichung und steht deshalb im Klartext im Audit-Log.
//
// BEWUSST OHNE FERIEN-SPERRE, anders als der Mahnlauf: Der Schulabgang fällt
// regelmäßig in die Sommerferien. Eine Sperre würde die Rückgabe-Abwicklung genau
// dann blockieren, wenn sie stattfindet. Der Kontoauszug ist auch keine Mahnung,
// sondern die Abschluss-Übersicht eines beendeten Ausleihkontos.
//
// POST /api/abgaenger/mail
func (s *Server) SendAbgaengerKontoauszuegeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		req, err := parseKlassenVersandRequest(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if !smtpKonfiguriert() {
			apierrors.SendHTTPError(w, http.StatusServiceUnavailable, fmt.Errorf("SMTP nicht konfiguriert – Versand nicht möglich"))
			return
		}

		// Dieselbe Saison wie Liste und Druck: Außerhalb gäbe es nichts zu versenden —
		// und ein Lauf im Oktober schickte den Klassenleitungen die laufende Ausleihe.
		if fenster := abgaengerFensterFuer(s.jetzt()); !fenster.Offen {
			apierrors.SendHTTPError(w, http.StatusConflict, abgaengerAusserhalbDerSaison(fenster))
			return
		}

		// Dieselbe Abfrage, die auch der Druck benutzt (leerer Filter = alle Klassen).
		// Papier und Mail zeigen damit garantiert denselben Stand.
		eintraege, err := s.queryAbgaengerKontoauszug(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		adressen, err := s.klassenlehrerAdressen(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		gewaehlt, unbekannt := waehleAbgaengerKlassen(eintraege, adressen, req.Klassen)
		if req.OverrideEmail != "" {
			gewaehlt = lenkeAbgaengerAufOverride(gewaehlt, req.OverrideEmail)
		}

		audit := klassenVersandAudit{
			Klassen:       abgaengerKlassennamen(gewaehlt),
			AlleKlassen:   req.Klassen == nil,
			OverrideEmail: req.OverrideEmail,
			Unbekannt:     unbekannt,
		}

		// Absicht vor dem Versand protokollieren — siehe logKlassenVersandAudit.
		audit.Phase = "start"
		s.logKlassenVersandAudit(r, "ABGAENGER_KONTOAUSZUG_MAIL", audit)

		erg := versendeAbgaengerKontoauszuege(gewaehlt, generateKontoauszugPDF, SendEmail)

		audit.Phase = "ende"
		audit.Sent, audit.Skipped, audit.Failed, audit.Abgebrochen = erg.Sent, erg.Skipped, erg.Failed, erg.Abgebrochen
		s.logKlassenVersandAudit(r, "ABGAENGER_KONTOAUSZUG_MAIL", audit)

		RespondJSON(w, http.StatusOK, bulkOverdueResponse{
			SentCount:    erg.Sent,
			SkippedCount: erg.Skipped,
			FailedCount:  erg.Failed,
			Abgebrochen:  erg.Abgebrochen,
			Message:      abgaengerVersandMeldung(erg, req.OverrideEmail),
		})
	}
}

// klassenlehrerAdressen lädt das Klassenlehrer-Mapping als Nachschlagetabelle.
//
// Anders als bei der Mahnliste (best-effort) bricht ein Fehler hier den Versand ab:
// Ohne Mapping wäre JEDE Klasse „ohne Adresse" und der Lauf meldete fröhlich
// „0 versendet, 12 übersprungen" — ein stiller Totalausfall, der wie ein Ergebnis
// aussieht.
func (s *Server) klassenlehrerAdressen(ctx context.Context) (map[string]string, error) {
	rows, err := s.DB.Pool.Query(ctx, `SELECT klasse, lehrer_email FROM klassen_lehrer_mapping`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	adressen := map[string]string{}
	for rows.Next() {
		var klasse, email string
		if err := rows.Scan(&klasse, &email); err != nil {
			return nil, err
		}
		// Normalisiert ablegen — „5A" im Mapping muss die Klasse „5a" treffen.
		adressen[repository.KlassenSchluessel(klasse)] = email
	}
	return adressen, rows.Err()
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

// lenkeAbgaengerAufOverride setzt den Empfänger aller gewählten Klassen auf eine
// Adresse. Kopie statt Mutation — die geladenen Daten bleiben unangetastet.
func lenkeAbgaengerAufOverride(klassen []abgaengerKlasse, empfaenger string) []abgaengerKlasse {
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

// generateKontoauszugPDF erzeugt den Kontoauszug EINES Abgängers — mit
// Unterschriftszeile, also exakt das Blatt, das auch aus dem Druck fällt.
func generateKontoauszugPDF(e pdf.KontoauszugEintrag) ([]byte, error) {
	return pdf.GenerateKontoauszugBatch([]pdf.KontoauszugEintrag{e}, true)
}

// versendeAbgaengerKontoauszuege verschickt je Klasse eine Mail mit einem PDF je
// Abgänger. generatePDF/sendMail sind injiziert, damit die Skip- und
// Adressierungslogik ohne echten PDF-/Mailversand testbar bleibt.
//
// Best-Effort auf zwei Ebenen: Ein fehlgeschlagenes Schüler-PDF kostet nur diesen
// einen Anhang, nicht die ganze Klasse — 29 fertige Kontoauszüge wegen eines
// kaputten zurückzuhalten wäre der teurere Fehler. Erst wenn KEIN Anhang übrig
// bleibt, zählt die Klasse als übersprungen.
// Drei Ausgänge statt zwei (31.08.2026): „übersprungen" ist Absicht (keine Adresse,
// keine Fälle), „fehlgeschlagen" ist ein Problem (alle PDFs kaputt, Versand
// abgelehnt). Vorher zählte der SMTP-Ausfall als übersprungen, die Rückmeldung
// erklärte ihn als „keine E-Mail hinterlegt oder keine offenen Ausleihen" — am
// Zeugnistag mit totem Relay stand da „0 versendet, 12 übersprungen", niemand
// sendete nach. Ein Mailserver-Fehler (ErrSMTPVersand) bricht ab wie beim
// Mahnlauf: Jede weitere Klasse hinge sonst bis zu 70 s am toten Relay; die
// restlichen zählen als fehlgeschlagen. Kontoauszüge doppelt zu senden ist
// unschädlich — der Lauf darf einfach wiederholt werden.
func versendeAbgaengerKontoauszuege(
	klassen []abgaengerKlasse,
	generatePDF func(pdf.KontoauszugEintrag) ([]byte, error),
	sendMail func(MailRequest) error,
) versandErgebnis {
	var erg versandErgebnis
	for i, kl := range klassen {
		if kl.Empfaenger == "" || len(kl.Eintraege) == 0 {
			erg.Skipped++
			continue
		}

		anhaenge := baueKontoauszugAnhaenge(kl, generatePDF)
		if len(anhaenge) == 0 {
			// Jedes einzelne PDF dieser Klasse ist gescheitert — das ist ein Problem,
			// keine Absicht.
			erg.Failed++
			continue
		}

		if err := sendMail(baueAbgaengerMailRequest(kl, anhaenge)); err != nil {
			log.Printf("abgaenger-kontoauszug: Versand an Klasse %s (%s) fehlgeschlagen: %v", kl.Klasse, kl.Empfaenger, err)
			erg.Failed++
			if errors.Is(err, mailservice.ErrSMTPVersand) {
				erg.Abgebrochen = true
				erg.Failed += len(klassen) - i - 1
				break
			}
			continue
		}
		erg.Sent++
	}
	return erg
}

func baueKontoauszugAnhaenge(
	kl abgaengerKlasse,
	generatePDF func(pdf.KontoauszugEintrag) ([]byte, error),
) []MailAttachment {
	// Leerzeichen und Schrägstriche aus dem Namen halten — ein Anhang „Kontoauszug_Max
	// Mustermann.pdf" ist beim Weiterreichen unhandlich, ein „/" darin sogar kaputt.
	sauber := strings.NewReplacer(" ", "_", "/", "-", "\\", "-")

	anhaenge := make([]MailAttachment, 0, len(kl.Eintraege))
	for _, e := range kl.Eintraege {
		pdfBytes, err := generatePDF(e)
		if err != nil {
			log.Printf("abgaenger-kontoauszug: PDF für %s %s (%s) fehlgeschlagen: %v",
				e.Schueler.Vorname, e.Schueler.Nachname, kl.Klasse, err)
			continue
		}
		anhaenge = append(anhaenge, MailAttachment{
			Name:        sauber.Replace(fmt.Sprintf("Kontoauszug_%s_%s.pdf", e.Schueler.Vorname, e.Schueler.Nachname)),
			ContentType: contentTypePDF,
			Data:        pdfBytes,
		})
	}
	return anhaenge
}

func baueAbgaengerMailRequest(kl abgaengerKlasse, anhaenge []MailAttachment) MailRequest {
	body := fmt.Sprintf(
		"Sehr geehrte Damen und Herren,\n\n"+
			"anbei erhalten Sie die Kontoauszüge der Abgänger der Klasse %s (Stand: %s).\n\n"+
			"Betroffene Schüler/innen: %d\n\n"+
			"Je Schüler/in liegt ein Blatt mit den noch nicht zurückgegebenen Medien bei.\n"+
			"Bitte geben Sie die Blätter aus; die Freigabezeile wird nach der Rückgabe\n"+
			"in der Bibliothek abgezeichnet.\n\n"+
			"Mit freundlichen Grüßen,\nSchulbibliothek",
		kl.Klasse,
		time.Now().Format(dateFormatDE),
		len(anhaenge),
	)

	return MailRequest{
		To:          kl.Empfaenger,
		Subject:     fmt.Sprintf("Kontoauszüge Abgänger – Klasse %s – %s", kl.Klasse, time.Now().Format(dateFormatDE)),
		Body:        body,
		Attachments: anhaenge,
	}
}

// abgaengerVersandMeldung nennt den Empfänger mit: „12 versendet" allein verrät
// nicht, ob die Auszüge an die Klassenleitungen oder an eine handgetippte Adresse
// gingen.
func abgaengerVersandMeldung(erg versandErgebnis, override string) string {
	if erg.Sent == 0 && erg.Skipped == 0 && erg.Failed == 0 {
		return "Keine der gewählten Klassen hat noch offene Ausleihen – nichts versendet."
	}
	ziel := "an die Klassenleitungen"
	if override != "" {
		ziel = "an " + override
	}
	text := fmt.Sprintf("Kontoauszüge von %d Klasse(n) %s versendet, %d übersprungen (keine E-Mail hinterlegt oder keine offenen Ausleihen).", erg.Sent, ziel, erg.Skipped)
	if erg.Failed > 0 {
		text += fmt.Sprintf(" %d FEHLGESCHLAGEN — nicht zugestellt.", erg.Failed)
	}
	if erg.Abgebrochen {
		text += " Lauf nach Mailserver-Fehler abgebrochen; bitte Mail-Server prüfen und erneut senden (doppelte Kontoauszüge sind unschädlich)."
	}
	return text
}
