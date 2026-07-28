package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// bulkOverdueRequest ist der optionale Body von POST /api/mail/send-bulk-overdue.
//
// Klassen ist bewusst ein Zeiger, weil „Feld fehlt" und „leeres Array" hier NICHT
// dasselbe bedeuten dürfen:
//
//	nil (Feld fehlt)  → alle überfälligen Klassen (Vertrag vor dem Konfigurations-Dialog)
//	[]  (leer gesendet) → 400
//
// Eine leere Auswahl heißt „niemand". Sie als „alle" zu lesen, wäre der teuerste
// denkbare Fehlgriff des Systems: ein Klick, hunderte ungewollte Mahnungen.
type bulkOverdueRequest struct {
	Klassen       *[]string `json:"klassen"`
	OverrideEmail string    `json:"override_email"`
}

// bulkOverdueResponse ist die Antwort von POST /api/mail/send-bulk-overdue.
type bulkOverdueResponse struct {
	SentCount    int    `json:"sent_count"`
	SkippedCount int    `json:"skipped_count"`
	Message      string `json:"message"`
}

// SendBulkOverdueHandler verschickt die Mahnliste der gewählten überfälligen Klassen
// an die jeweilige Klassenleitung — eine E-Mail pro Klasse an genau eine Adresse.
//
// Datenschutz by design (bewusst identisch zum Einzelversand /api/mahnwesen/senden):
// Es wird NICHT an einzelne, i.d.R. minderjährige Schüler gemailt, sondern an die
// Lehrkraft, die die Schüler informiert. Jede Lehrkraft erhält ausschließlich die
// eigene Klassenliste — es gibt also keine klassenübergreifende Offenlegung von
// Empfängern oder Mahn-Status (kein TO/CC über mehrere Betroffene). Während
// Ferien-/Schließzeiten ist der Versand gesperrt, und der Massenversand wird
// auditiert (Rechenschaftspflicht, Art. 5 (2) DSGVO).
//
// AUSNAHME override_email: Damit gehen die Listen ALLER gewählten Klassen an eine
// einzige, von Hand eingetippte Adresse (Vertretungsfall, Sekretariat, Probelauf).
// Das ist eine bewusste Abweichung von „jede Lehrkraft nur die eigene Klasse" — ein
// Empfänger sieht dann die Namen mehrerer Klassen. Deshalb: weiterhin eine Mail je
// Klasse (kein Sammel-PDF, kein CC), und die Adresse wird im Audit-Log namentlich
// festgehalten, nicht nur als Zähler.
//
// INVARIANTE: Der Mail-Versand erhöht die Mahnstufe NICHT. Die Stufenerhöhung ist ein
// physischer Verwaltungsakt und passiert ausschließlich beim PDF-Druck
// (mahnwesen_bulk.go). Die Mail ist ein „Friendly Reminder" VOR der Eskalation — und
// verhält sich bewusst identisch zum Einzelversand (system-konsistent, keine
// Button-abhängige Sonderlogik). Siehe docs/invarianten.md §1.
// POST /api/mail/send-bulk-overdue
func (s *Server) SendBulkOverdueHandler(mahnRepo *repository.MahnwesenRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Auswahl aus dem Body lesen. Fehlt der Body ganz, bleibt es beim alten
		//    Vertrag „alle überfälligen Klassen an ihre Klassenleitungen".
		req, err := parseBulkOverdueRequest(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// 2. Ferien-/Schließzeit-Sperre — identisch zum Einzelversand.
		isFerien, ferienName, err := mahnRepo.CheckFerienAktiv(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if isFerien {
			apierrors.SendHTTPError(w, http.StatusForbidden, fmt.Errorf("mahnwesen ist derzeit pausiert (Ferien/Schließzeit: %s)", ferienName))
			return
		}

		// 3. Ohne konfigurierten Mailserver kein Massenversand.
		if os.Getenv("SMTP_HOST") == "" {
			apierrors.SendHTTPError(w, http.StatusServiceUnavailable, fmt.Errorf("SMTP nicht konfiguriert – Massenversand nicht möglich"))
			return
		}

		// 4. Alle überfälligen Klassen laden (leerer Filter = alle Klassen) und die
		//    Auswahl des Aufrufers darauf anwenden.
		alle, err := mahnRepo.QueryUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		gewaehlt, unbekannt := waehleKlassen(alle, req.Klassen)

		// 5. Optional: Empfänger auf eine einzelne Adresse umlenken.
		if req.OverrideEmail != "" {
			gewaehlt = zieleAufOverride(gewaehlt, req.OverrideEmail)
		}

		// 6. Absicht protokollieren, BEVOR die erste Mail rausgeht. Ein Eintrag erst
		//    danach hätte genau im teuersten Fall eine Lücke: Bricht der Prozess nach
		//    300 versendeten Mahnungen ab, stünde nirgends, wer den Lauf ausgelöst hat
		//    und an welche Adresse er ging.
		s.logBulkOverdueAudit(r, bulkOverdueAudit{
			Phase:         "start",
			Klassen:       klassennamen(gewaehlt),
			AlleKlassen:   req.Klassen == nil,
			OverrideEmail: req.OverrideEmail,
			Unbekannt:     unbekannt,
		})

		// 7. Je Klasse eine eigene Mahnliste versenden.
		//    generateMahnPDF/SendEmail werden injiziert, damit die Skip- und
		//    Adressierungslogik ohne echten PDF-/Mailversand testbar bleibt.
		sent, skipped := versendeKlassenMahnungen(gewaehlt, generateMahnPDF, SendEmail)

		// 8. Ergebnis protokollieren.
		s.logBulkOverdueAudit(r, bulkOverdueAudit{
			Phase:         "ende",
			Klassen:       klassennamen(gewaehlt),
			AlleKlassen:   req.Klassen == nil,
			OverrideEmail: req.OverrideEmail,
			Unbekannt:     unbekannt,
			Sent:          sent,
			Skipped:       skipped,
		})

		RespondJSON(w, http.StatusOK, bulkOverdueResponse{
			SentCount:    sent,
			SkippedCount: skipped,
			Message:      bulkOverdueMeldung(sent, skipped, req.OverrideEmail),
		})
	}
}

// parseBulkOverdueRequest liest die optionale Auswahl aus dem Request-Body und
// weist alles zurück, was einen Massenversand an den falschen Empfänger schicken
// würde. Ein fehlender Body ist gültig (= alle Klassen, alter Vertrag), ein
// kaputter nicht.
func parseBulkOverdueRequest(r *http.Request) (bulkOverdueRequest, error) {
	var req bulkOverdueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return req, fmt.Errorf("ungültiger Request-Body: %w", err)
	}

	// Die Adresse kommt von Hand aus einem Eingabefeld. Ein Tippfehler darf hier
	// nicht in einen „erfolgreichen" Lauf ins Leere münden, deshalb dieselbe
	// Prüfung, die auch der Versand selbst anlegt (mail_sender.go).
	req.OverrideEmail = strings.TrimSpace(req.OverrideEmail)
	if req.OverrideEmail != "" {
		if _, err := mail.ParseAddress(req.OverrideEmail); err != nil {
			return req, fmt.Errorf("ungültige Empfänger-Adresse %q", req.OverrideEmail)
		}
	}

	if req.Klassen == nil {
		return req, nil
	}

	gesaeubert := make([]string, 0, len(*req.Klassen))
	for _, k := range *req.Klassen {
		if k = strings.TrimSpace(k); k != "" {
			gesaeubert = append(gesaeubert, k)
		}
	}
	if len(gesaeubert) == 0 {
		return req, errors.New("keine Klasse ausgewählt")
	}
	req.Klassen = &gesaeubert
	return req, nil
}

// waehleKlassen schneidet die geladenen Klassen auf die Auswahl zu und meldet
// zurück, welche angeforderten Klassen es gar nicht (mehr) gibt.
//
// Die Unbekannten werden nicht verschluckt, sondern auditiert: Sonst wäre ein
// Format-Versatz zwischen Frontend und DB („5a" vs. „05A") ein dauerhaft stiller
// Nullversand — der Lauf meldet Erfolg und niemand bekommt Post.
func waehleKlassen(alle []repository.MahnwesenKlasse, auswahl *[]string) (gewaehlt []repository.MahnwesenKlasse, unbekannt []string) {
	if auswahl == nil {
		return alle, nil
	}

	vorhanden := make(map[string]repository.MahnwesenKlasse, len(alle))
	for _, k := range alle {
		vorhanden[k.Klasse] = k
	}

	gewaehlt = make([]repository.MahnwesenKlasse, 0, len(*auswahl))
	gesehen := make(map[string]bool, len(*auswahl))
	for _, name := range *auswahl {
		if gesehen[name] {
			continue // doppelt gesendete Auswahl darf nicht doppelt mahnen
		}
		gesehen[name] = true

		if k, ok := vorhanden[name]; ok {
			gewaehlt = append(gewaehlt, k)
		} else {
			unbekannt = append(unbekannt, name)
		}
	}
	return gewaehlt, unbekannt
}

// zieleAufOverride lenkt den Empfänger ALLER gewählten Klassen auf eine Adresse um.
//
// Die Klassen werden kopiert, nicht mutiert: Die geladenen Repository-Daten bleiben
// unangetastet, und die Skip-Logik von versendeKlassenMahnungen greift unverändert
// weiter (Klassen ohne Fälle bleiben aussen vor, auch mit Override-Adresse).
func zieleAufOverride(klassen []repository.MahnwesenKlasse, empfaenger string) []repository.MahnwesenKlasse {
	umgelenkt := make([]repository.MahnwesenKlasse, len(klassen))
	copy(umgelenkt, klassen)
	for i := range umgelenkt {
		umgelenkt[i].LehrerEmail = empfaenger
	}
	return umgelenkt
}

// klassennamen reduziert die Klassen auf ihre Kürzel — für das Audit-Log, das den
// Umfang eines Laufs festhalten soll, nicht dessen Inhalt (Datenminimierung: keine
// Schülernamen im Log).
func klassennamen(klassen []repository.MahnwesenKlasse) []string {
	namen := make([]string, 0, len(klassen))
	for _, k := range klassen {
		namen = append(namen, k.Klasse)
	}
	return namen
}

// bulkOverdueMeldung formuliert die Rückmeldung so, dass der Empfänger darin
// vorkommt. „12 versendet" allein verrät nicht, ob sie an die Klassenleitungen
// oder an die von Hand eingetippte Adresse gingen.
func bulkOverdueMeldung(sent, skipped int, override string) string {
	if sent == 0 && skipped == 0 {
		return "Keine der gewählten Klassen hat noch überfällige Fälle – nichts versendet."
	}
	ziel := "an die Klassenleitungen"
	if override != "" {
		ziel = "an " + override
	}
	return fmt.Sprintf("%d Klassen-Mahnliste(n) %s versendet, %d übersprungen (keine E-Mail hinterlegt oder keine Fälle).", sent, ziel, skipped)
}

// versendeKlassenMahnungen erzeugt je Klasse das Mahn-PDF und mailt es an die
// hinterlegte Klassenleitung. Klassen ohne E-Mail oder ohne überfällige Schüler
// werden übersprungen. Ein Fehler bei einer einzelnen Klasse bricht den Lauf NICHT
// ab (Best-Effort), wird aber protokolliert und als "skipped" gezählt.
//
// generatePDF und sendMail sind injiziert (Produktion: generateMahnPDF/SendEmail),
// damit die Skip- und Adressierungslogik ohne echten PDF-/Mailversand testbar ist.
func versendeKlassenMahnungen(
	klassen []repository.MahnwesenKlasse,
	generatePDF func([]repository.MahnwesenKlasse) ([]byte, error),
	sendMail func(MailRequest) error,
) (sent, skipped int) {
	for _, kl := range klassen {
		if kl.LehrerEmail == "" || len(kl.Schueler) == 0 {
			skipped++
			continue
		}

		einzelKlasse := []repository.MahnwesenKlasse{kl}

		pdfBytes, err := generatePDF(einzelKlasse)
		if err != nil {
			log.Printf("bulk-overdue: PDF für Klasse %s fehlgeschlagen: %v", kl.Klasse, err)
			skipped++
			continue
		}

		totalSchueler, totalMedien := zaehleMahnStatistik(einzelKlasse)
		mailReq := baueMahnMailRequest(
			mahnwesenSendenRequest{Klasse: kl.Klasse, Email: kl.LehrerEmail},
			pdfBytes, totalSchueler, totalMedien,
		)

		if err := sendMail(mailReq); err != nil {
			log.Printf("bulk-overdue: Versand an Klasse %s (%s) fehlgeschlagen: %v", kl.Klasse, kl.LehrerEmail, err)
			skipped++
			continue
		}
		sent++
	}
	return sent, skipped
}

// bulkOverdueAudit ist der details-Payload des Audit-Eintrags BULK_OVERDUE_MAIL.
//
// Er beantwortet die Frage, die im Zweifel jemand von aussen stellt: Wer hat wann
// wie viele Mahnungen an WEN geschickt? Die Override-Adresse steht deshalb im
// Klartext im Log — sie ist der Unterschied zwischen „ging an die Klassenleitungen"
// und „ging an eine von Hand eingetippte Adresse".
type bulkOverdueAudit struct {
	// Phase trennt Absicht ("start") von Ergebnis ("ende"). Zwei Einträge je Lauf:
	// Nur so bleibt ein mittendrin abgebrochener Versand nachvollziehbar.
	Phase         string   `json:"phase"`
	Klassen       []string `json:"klassen"`
	AlleKlassen   bool     `json:"alle_klassen"`
	OverrideEmail string   `json:"override_email,omitempty"`
	Unbekannt     []string `json:"unbekannt,omitempty"`
	Sent          int      `json:"sent"`
	Skipped       int      `json:"skipped"`
}

// logBulkOverdueAudit protokolliert den Massenversand im audit_logs — analog zum
// Import-Audit.
//
// Der Payload wird marshalled, nicht zusammengesetzt: Seit die frei eingetippte
// Override-Adresse mit im Detail steht, würde ein handgebautes JSON-Literal bei
// einem Anführungszeichen in der Eingabe kaputtes JSON in eine jsonb-Spalte
// schreiben — der INSERT scheitert, und der Lauf verlöre still seinen Log-Eintrag.
func (s *Server) logBulkOverdueAudit(r *http.Request, details bulkOverdueAudit) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	payload, err := json.Marshal(details)
	if err != nil {
		log.Printf("bulk-overdue: Audit-Details nicht serialisierbar: %v", err)
		return
	}
	logExec(s.DB.Pool.Exec(r.Context(), "INSERT INTO audit_logs (admin_id, aktion, details, ip_adresse) VALUES ($1, $2, $3::jsonb, $4)", claims.UserID, "BULK_OVERDUE_MAIL", string(payload), getIP(r)))
}
