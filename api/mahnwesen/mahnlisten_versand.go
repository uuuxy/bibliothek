package mahnwesen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/mailservice"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"
)

// klassenVersandAnfrage ist der freiwillige Rumpf jedes Massenversands an Klassen (Mahnlauf,
// Kontoauszüge der Abgänger). Ein Typ für beide, weil es die gefährliche Regel darunter
// (leere Auswahl heißt „niemand", nicht „alle") nur an
// einer Stelle geben.
//
// Klassen ist bewusst ein Zeiger, weil „Feld fehlt" und „leeres Array" hier nicht
// dasselbe bedeuten dürfen:
//
//	nil (Feld fehlt)  → alle überfälligen Klassen (Vertrag vor dem Konfigurations-Dialog)
//	[]  (leer gesendet) → 400
//
// Eine leere Auswahl heißt „niemand". Sie als „alle" zu lesen, wäre der teuerste
// denkbare Fehlgriff des Systems: ein Klick, hunderte ungewollte Mahnungen.
type klassenVersandAnfrage struct {
	Klassen   *[]string `json:"klassen"`
	Umleitung string    `json:"override_email"`
}

// klassenVersandAntwort ist die Antwort von POST /api/mail/send-bulk-overdue und
// POST /api/abgaenger/mail. failed_count zählt Klassen, deren Versand fehlschlug, getrennt von
// skipped (keine Adresse, keine Fälle): Sonst sähe ein Ausfall aus wie Absicht.
type klassenVersandAntwort struct {
	Versendet      int    `json:"sent_count"`
	Uebersprungen  int    `json:"skipped_count"`
	Fehlgeschlagen int    `json:"failed_count"`
	Abgebrochen    bool   `json:"abgebrochen"`
	Meldung        string `json:"message"`
}

// MahnlistenVersenden verschickt die Mahnliste der gewählten überfälligen Klassen
// an die jeweilige Klassenleitung — eine E-Mail pro Klasse an genau eine Adresse.
//
// Datenschutz: Die Liste geht nicht an einzelne, meist minderjährige Schüler, sondern an die
// Lehrkraft, die die Schüler informiert. Jede Lehrkraft erhält nur die eigene Klassenliste, es
// gibt also keine Offenlegung von Empfängern oder Mahnstand über Klassen hinweg (kein TO/CC
// über mehrere Betroffene). Der Massenversand steht im Protokoll (Rechenschaftspflicht,
// Art. 5 (2) DSGVO).
//
// Ausnahme override_email: Damit gehen die Listen aller gewählten Klassen an eine einzige, von
// Hand eingetippte Adresse (Vertretung, Sekretariat, Probelauf). Ein Empfänger sieht dann die
// Namen mehrerer Klassen. Deshalb bleibt es bei einer Mail je Klasse (kein Sammel-PDF, kein
// CC), und die Adresse steht namentlich im Protokoll, nicht nur als Zähler.
//
// Invariante: Der Mailversand erhöht die Mahnstufe nicht. Sie steigt nur beim Druck der
// Mahnbriefe (mahnbriefe.go), dem Verwaltungsakt; die Mail ist eine Erinnerung vor der
// Eskalation. Siehe docs/invarianten.md §1.
// POST /api/mail/send-bulk-overdue
func (t *Tueren) MahnlistenVersenden(mahnAbfragen *repository.MahnwesenAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Auswahl aus dem Rumpf lesen. Fehlt der Rumpf ganz, bleibt es beim alten
		//    Vertrag „alle überfälligen Klassen an ihre Klassenleitungen".
		req, err := leseKlassenVersandAnfrage(r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		// 2. Ohne konfigurierten Mailserver kein Massenversand.
		if !mailservice.SMTPKonfiguriert() {
			apierrors.SendHTTPError(w, http.StatusServiceUnavailable, fmt.Errorf("SMTP nicht konfiguriert – Massenversand nicht möglich"))
			return
		}

		// 3. Alle überfälligen Klassen laden (leerer Filter = alle Klassen) und die
		//    Auswahl des Aufrufers darauf anwenden.
		alle, err := mahnAbfragen.LadeUeberfaelligeNachKlasse(ctx, "")
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		gewaehlt, unbekannt := waehleKlassen(nurKlassen(alle), req.Klassen)

		// 4. Optional: Empfänger auf eine einzelne Adresse umlenken.
		if req.Umleitung != "" {
			gewaehlt = lenkeKlassenUm(gewaehlt, req.Umleitung)
		}

		// 5. Absicht protokollieren, bevor die erste Mail rausgeht. Ein Eintrag erst
		//    danach hätte genau im teuersten Fall eine Lücke: Bricht der Prozess nach
		//    300 versendeten Mahnungen ab, stünde nirgends, wer den Lauf ausgelöst hat
		//    und an welche Adresse er ging.
		t.protokolliereKlassenVersand(r, "BULK_OVERDUE_MAIL", klassenVersandProtokoll{
			Phase:       "start",
			Klassen:     klassennamen(gewaehlt),
			AlleKlassen: req.Klassen == nil,
			Umleitung:   req.Umleitung,
			Unbekannt:   unbekannt,
		})

		// 6. Je Klasse eine eigene Mahnliste versenden. erzeugeMahnliste und SendEmail kommen
		//    als Parameter, damit sich das Überspringen und die Adressierung ohne echten
		//    PDF- und Mailversand prüfen lassen.
		erg := versendeKlassenMahnungen(gewaehlt, erzeugeMahnliste, mailservice.SendEmail)

		// 7. Ergebnis protokollieren.
		t.protokolliereKlassenVersand(r, "BULK_OVERDUE_MAIL", klassenVersandProtokoll{
			Phase:          "ende",
			Klassen:        klassennamen(gewaehlt),
			AlleKlassen:    req.Klassen == nil,
			Umleitung:      req.Umleitung,
			Unbekannt:      unbekannt,
			Versendet:      erg.Versendet,
			Uebersprungen:  erg.Uebersprungen,
			Fehlgeschlagen: erg.Fehlgeschlagen,
			Abgebrochen:    erg.Abgebrochen,
		})

		tuer.AntworteJSON(w, http.StatusOK, klassenVersandAntwort{
			Versendet:      erg.Versendet,
			Uebersprungen:  erg.Uebersprungen,
			Fehlgeschlagen: erg.Fehlgeschlagen,
			Abgebrochen:    erg.Abgebrochen,
			Meldung:        mahnlistenMeldung(erg, req.Umleitung),
		})
	}
}

// leseKlassenVersandAnfrage liest die freiwillige Auswahl aus dem Rumpf und weist alles
// zurück, was einen Massenversand an den falschen Empfänger schicken würde. Ein fehlender Rumpf
// ist gültig (alle Klassen), ein kaputter nicht.
func leseKlassenVersandAnfrage(r *http.Request) (klassenVersandAnfrage, error) {
	var req klassenVersandAnfrage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return req, fmt.Errorf("ungültiger Request-Body: %w", err)
	}

	// Die Adresse kommt von Hand aus einem Eingabefeld. Ein Tippfehler darf hier
	// nicht in einen „erfolgreichen" Lauf ins Leere münden, deshalb dieselbe
	// Prüfung, die auch der Versand selbst anlegt (mailservice/mail_sender.go).
	req.Umleitung = ergaenzeSchulDomain(strings.TrimSpace(req.Umleitung))
	if req.Umleitung != "" {
		if _, err := mail.ParseAddress(req.Umleitung); err != nil {
			return req, fmt.Errorf("ungültige Empfänger-Adresse %q", req.Umleitung)
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

// ergaenzeSchulDomain vervollständigt eine von Hand eingetippte Adresse ohne „@"
// um die Domäne der Schule: „mueller" wird zu „mueller@philipp-reis-schule.de".
//
// Alle Dienstadressen der Schule liegen auf derselben Domäne — sie jedes Mal
// mitzutippen ist reine Fehlerquelle, und ein Vertipper darin fällt niemandem auf
// (der Lauf meldet Erfolg, die Post kommt nie an).
//
// Die Domäne wird nicht im Code festgeschrieben, sondern aus der Absenderadresse
// des Systems abgeleitet — derselben, mit der auch verschickt wird. Damit folgt sie
// automatisch der Konfiguration in der Oberfläche und stimmt auch nach einem
// Domänenwechsel noch. Ist keine konfiguriert, bleibt die Eingabe unverändert und
// läuft in die normale Adressprüfung — lieber eine klare Fehlermeldung als eine
// geratene Domäne.
//
// Eine vollständige Adresse (mit „@") bleibt unangetastet: Der Versand an eine
// externe Stelle — Schulamt, Vertretung, private Adresse — muss möglich bleiben.
func ergaenzeSchulDomain(eingabe string) string {
	if eingabe == "" || strings.Contains(eingabe, "@") {
		return eingabe
	}

	konfig, err := mailservice.VersandKonfig()
	if err != nil {
		return eingabe
	}
	absender := konfig.Absender
	if absender == "" {
		absender = konfig.Benutzer
	}
	at := strings.LastIndex(absender, "@")
	if at < 0 {
		return eingabe
	}
	return eingabe + absender[at:]
}

// nurKlassen lässt die Gruppe der Ehemaligen fallen. Sie ist keine Klasse: Ihre Liste geht
// an keine Klassenleitung und auch an keine von Hand genannte Adresse.
func nurKlassen(gruppen []repository.MahnwesenKlasse) []repository.MahnwesenKlasse {
	klassen := make([]repository.MahnwesenKlasse, 0, len(gruppen))
	for _, g := range gruppen {
		if !g.Ehemalige {
			klassen = append(klassen, g)
		}
	}
	return klassen
}

// waehleKlassen schneidet die geladenen Klassen auf die Auswahl zu und meldet
// zurück, welche angeforderten Klassen es gar nicht (mehr) gibt.
//
// Die Unbekannten werden nicht verschluckt, sondern auditiert: Sonst wäre ein
// Format-Versatz zwischen Oberfläche und Datenbank („5a" gegen „05A") ein dauerhaft stiller
// Nullversand: Der Lauf meldet Erfolg, und niemand bekommt Post.
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

// lenkeKlassenUm lenkt den Empfänger aller gewählten Klassen auf eine Adresse um.
//
// Die Klassen werden kopiert, nicht verändert: Die geladenen Daten bleiben unangetastet, und
// versendeKlassenMahnungen überspringt weiter Klassen ohne Fälle, auch mit umgeleiteter
// Adresse.
func lenkeKlassenUm(klassen []repository.MahnwesenKlasse, empfaenger string) []repository.MahnwesenKlasse {
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

// mahnlistenMeldung formuliert die Rückmeldung so, dass der Empfänger darin
// vorkommt. „12 versendet" allein verrät nicht, ob sie an die Klassenleitungen
// oder an die von Hand eingetippte Adresse gingen.
func mahnlistenMeldung(erg versandErgebnis, umleitung string) string {
	if erg.Versendet == 0 && erg.Uebersprungen == 0 && erg.Fehlgeschlagen == 0 {
		return "Keine der gewählten Klassen hat noch überfällige Fälle – nichts versendet."
	}
	ziel := "an die Klassenleitungen"
	if umleitung != "" {
		ziel = "an " + umleitung
	}
	text := fmt.Sprintf("%d Klassen-Mahnliste(n) %s versendet, %d übersprungen (keine E-Mail hinterlegt oder keine Fälle).", erg.Versendet, ziel, erg.Uebersprungen)
	if erg.Fehlgeschlagen > 0 {
		text += fmt.Sprintf(" %d FEHLGESCHLAGEN — nicht zugestellt.", erg.Fehlgeschlagen)
	}
	if erg.Abgebrochen {
		text += " Lauf nach Mailserver-Fehler abgebrochen; bitte Mail-Server prüfen und erneut senden (eine Klasse wird nie doppelt gemahnt — der Mailversand erhöht die Mahnstufe nicht)."
	}
	return text
}

// versandErgebnis trennt drei Ausgänge: versendet, übersprungen (keine Adresse, keine Fälle:
// Absicht) und fehlgeschlagen (PDF- oder Versandfehler: ein Problem). Abgebrochen: Nach einem
// Mailserver-Fehler (ErrSMTPVersand) werden die übrigen Klassen nicht mehr versucht, weil
// jeder weitere Versuch bis zu 70 s am toten Relay hinge; sie zählen als fehlgeschlagen.
type versandErgebnis struct {
	Versendet, Uebersprungen, Fehlgeschlagen int
	Abgebrochen                              bool
}

// versendeKlassenMahnungen erzeugt je Klasse das Mahn-PDF und mailt es an die
// hinterlegte Klassenleitung. Klassen ohne E-Mail oder ohne überfällige Schüler
// werden übersprungen. Ein Fehler bei einer einzelnen Klasse (PDF, abgelehnte Adresse)
// bricht den Lauf nicht ab, zählt aber als fehlgeschlagen — nicht als übersprungen. Ein
// Mailserver-Fehler (ErrSMTPVersand) bricht ab: Das Relay ist weg, jede weitere Klasse
// hinge nur bis zur Frist; die Restlichen zählen als fehlgeschlagen.
//
// erzeugePDF und sendeMail kommen als Parameter (im Betrieb erzeugeMahnliste und SendEmail),
// damit sich das Überspringen und die Adressierung ohne echten PDF- und Mailversand prüfen
// lassen.
func versendeKlassenMahnungen(
	klassen []repository.MahnwesenKlasse,
	erzeugePDF func([]repository.MahnwesenKlasse) ([]byte, error),
	sendeMail func(mailservice.MailRequest) error,
) versandErgebnis {
	var erg versandErgebnis
	for _, kl := range klassen {
		if kl.LehrerEmail == "" || len(kl.Schueler) == 0 {
			erg.Uebersprungen++
			continue
		}
		if erg.Abgebrochen {
			erg.Fehlgeschlagen++
			continue
		}

		einzelKlasse := []repository.MahnwesenKlasse{kl}

		pdfDaten, err := erzeugePDF(einzelKlasse)
		if err != nil {
			log.Printf("bulk-overdue: PDF für Klasse %s fehlgeschlagen: %v", kl.Klasse, err)
			erg.Fehlgeschlagen++
			continue
		}

		schuelerGesamt, medienGesamt := mahnStatistik(einzelKlasse)
		nachricht := baueMahnMail(
			mahnlisteEmpfaenger{Klasse: kl.Klasse, Email: kl.LehrerEmail},
			pdfDaten, schuelerGesamt, medienGesamt,
		)

		if err := sendeMail(nachricht); err != nil {
			log.Printf("bulk-overdue: Versand an Klasse %s (%s) fehlgeschlagen: %v", kl.Klasse, kl.LehrerEmail, err)
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

// klassenVersandProtokoll sind die Angaben (details) im Protokoll jedes Massenversands an
// Klassen (BULK_OVERDUE_MAIL, ABGAENGER_KONTOAUSZUG_MAIL).
//
// Er beantwortet die Frage, die im Zweifel jemand von aussen stellt: Wer hat wann
// wie viele Mahnungen an wen geschickt? Die Umleitung steht deshalb im
// Klartext im Log — sie ist der Unterschied zwischen „ging an die Klassenleitungen"
// und „ging an eine von Hand eingetippte Adresse".
type klassenVersandProtokoll struct {
	// Phase trennt Absicht ("start") von Ergebnis ("ende"). Zwei Einträge je Lauf:
	// Nur so bleibt ein mittendrin abgebrochener Versand nachvollziehbar.
	Phase          string   `json:"phase"`
	Klassen        []string `json:"klassen"`
	AlleKlassen    bool     `json:"alle_klassen"`
	Umleitung      string   `json:"override_email,omitempty"`
	Unbekannt      []string `json:"unbekannt,omitempty"`
	Versendet      int      `json:"sent"`
	Uebersprungen  int      `json:"skipped"`
	Fehlgeschlagen int      `json:"failed,omitempty"`
	Abgebrochen    bool     `json:"abgebrochen,omitempty"`
}

// protokolliereKlassenVersand schreibt einen Massenversand ins Protokoll (audit_logs). Die
// Aktion kommt vom Aufrufer, die Angaben sind für alle Massenversände an Klassen dieselben.
//
// Die Angaben werden mit json.Marshal erzeugt, nicht zusammengesetzt: Ein Anführungszeichen in
// der frei eingetippten Umleitung machte ein handgebautes JSON-Literal kaputt, der INSERT in die
// jsonb-Spalte scheiterte, und der Lauf verlöre still seinen Eintrag.
func (t *Tueren) protokolliereKlassenVersand(r *http.Request, aktion string, details klassenVersandProtokoll) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	inhalt, err := json.Marshal(details)
	if err != nil {
		log.Printf("%s: Audit-Details nicht serialisierbar: %v", aktion, err)
		return
	}
	// Vom Kontext der Anfrage abgekoppelt: Der Eintrag am Ende kommt nach dem Versand, und ein
	// Lauf über viele Klassen überlebt die 15-s-fristMiddleware; mit r.Context() schlüge der
	// INSERT still fehl, und genau der Eintrag mit den Zahlen fehlte. Die Claims bleiben erhalten
	// (WithoutCancel behält die Werte).
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
	defer abbruch()
	t.SchreibeAdminProtokoll(ctx, claims.UserID, aktion, clientip.AusAnfrage(r), string(inhalt))
}
