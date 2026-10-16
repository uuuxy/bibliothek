package portal

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/mailservice"
	"bibliothek/repository"
)

// klassensatzReservierungAnfrage ist die Reservierung eines Klassensatzes aus „Mein Portal".
type klassensatzReservierungAnfrage struct {
	TitelID string `json:"titel_id" validate:"omitempty,uuid_oder_leer"`
	Klasse  string `json:"klasse"`
	Anzahl  int    `json:"anzahl"`
	Notiz   string `json:"notiz,omitempty"`
	// IdempotenzSchluessel vergibt die Oberfläche je Absenden. Ein Doppelklick schickt
	// denselben Schlüssel und legt keine zweite Reservierung an; ohne Schlüssel legt jedes
	// Absenden eine an.
	IdempotenzSchluessel string `json:"idempotency_key,omitempty"`
}

// KlassensatzReservierungAnlegen nimmt die Reservierung eines Klassensatzes durch eine
// Lehrkraft an.
// POST /api/reservierungen/klassensatz
func (t *Tueren) KlassensatzReservierungAnlegen() http.HandlerFunc {
	return t.legeKlassensatzReservierungAn
}

// legeKlassensatzReservierungAn prüft Titel und Bestand und legt die Reservierung an.
func (t *Tueren) legeKlassensatzReservierungAn(w http.ResponseWriter, r *http.Request) {
	var req klassensatzReservierungAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}
	if !pruefeKlassensatzAnfrage(w, &req) {
		return
	}

	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}

	ctx := r.Context()
	abfragen := repository.NewReservierungsAbfragen(t.DB.Pool)

	// Ein Datenbankfehler ist kein „nicht gefunden": Sonst endete ein Verbindungsabbruch
	// als 404, und die Lehrkraft läse „Buchtitel nicht gefunden" für einen Titel, der da ist.
	vorhanden, err := abfragen.IstTitelVorhanden(ctx, req.TitelID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	if !vorhanden {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Titel nicht gefunden", nil)
		return
	}

	// Mehr Exemplare, als die Bibliothek besitzt, ergäben eine Aufgabe, die sich nie erfüllen
	// lässt; die Menge wird deshalb gegen den Bestand ohne ausgesonderte Exemplare geprüft.
	bestand, err := abfragen.ZaehleTitelBestand(ctx, req.TitelID)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	if req.Anzahl > bestand {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf("Nur %d Exemplare im Bestand, %d lassen sich nicht reservieren", bestand, req.Anzahl), nil)
		return
	}

	id, neu, err := abfragen.LegeAn(ctx, req.TitelID, req.Klasse, req.Anzahl, leerAlsNil(req.Notiz), claims.KontoID, leerAlsNil(req.IdempotenzSchluessel))
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	// neu=false: Derselbe Idempotenz-Schlüssel lief schon durch (Doppelklick); die Antwort
	// nennt die bestehende Reservierung.
	status := "erstellt"
	if !neu {
		status = "bereits_vorhanden"
	}
	tuer.AntworteJSON(w, http.StatusCreated, map[string]string{"id": id, "status": status})
}

// KlassensatzReservierungen liefert der Bibliothek alle Reservierungen: die offenen in der
// Reihenfolge der Warteschlange, dahinter die erledigten.
// GET /api/reservierungen/klassensatz
func (t *Tueren) KlassensatzReservierungen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abfragen := repository.NewReservierungsAbfragen(t.DB.Pool)
		liste, err := abfragen.LadeAlle(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Eine leere Liste kommt als [] an, nicht als null.
		if liste == nil {
			liste = []repository.Reservierung{}
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// KlassensatzReservierungenAnzahl zählt die offenen Reservierungen für den Zähler am
// Menüpunkt.
// GET /api/reservierungen/klassensatz/anzahl
func (t *Tueren) KlassensatzReservierungenAnzahl() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abfragen := repository.NewReservierungsAbfragen(t.DB.Pool)
		anzahl, err := abfragen.ZaehleOffene(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]int{"anzahl": anzahl})
	}
}

// OffeneKlassensatzReservierungen liefert die Warteschlange für das
// Kollegiums-Portal: wer vor der eigenen Reservierung steht (Titel, Klasse, Menge) —
// bewusst ohne Personendaten des Anfordernden. Eigener Endpunkt, weil die volle
// Liste (GET /api/reservierungen/klassensatz) hinter view_orders liegt und dem
// Kollegium verschlossen bleibt.
// GET /api/reservierungen/klassensatz/offen
func (t *Tueren) OffeneKlassensatzReservierungen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		abfragen := repository.NewReservierungsAbfragen(t.DB.Pool)
		offene, err := abfragen.LadeOffene(r.Context())
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, offene)
	}
}

// EigeneKlassensatzReservierungen liefert der angemeldeten Lehrkraft ihre Reservierungen,
// offene und kürzlich abgeschlossene samt Notiz der Bibliothek.
// GET /api/reservierungen/klassensatz/eigene, Muster und Recht wie GET /api/anliegen/eigene.
func (t *Tueren) EigeneKlassensatzReservierungen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}
		abfragen := repository.NewReservierungsAbfragen(t.DB.Pool)
		eigene, err := abfragen.LadeEigene(r.Context(), claims.KontoID)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, eigene)
	}
}

// KlassensatzReservierungErledigen schließt die Reservierung eines Klassensatzes ab.
// PUT /api/reservierungen/klassensatz/{id}/erledigen  { "notiz": "24 von 30, Rest bei der 8a" }
//
// Der Rumpf ist freiwillig (leer heißt ohne Notiz), damit ein Browser mit altem Stand der
// Oberfläche weiter abschließen kann.
func (t *Tueren) KlassensatzReservierungErledigen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung fehlt", nil)
			return
		}
		notiz, ok := leseErledigtNotiz(w, r)
		if !ok {
			return
		}

		abfragen := repository.NewReservierungsAbfragen(t.DB.Pool)
		erledigt, err := abfragen.Erledige(r.Context(), id, notiz)

		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if erledigt == nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Reservierung nicht gefunden oder schon erledigt", nil)
			return
		}

		// Die Bereit-Mail ist das Signal an die Lehrkraft; einen Menüpunkt dafür gibt es nicht.
		// Das Abschließen gilt, auch wenn der Versand scheitert: Die Antwort nennt den
		// Ausgang, und die Oberfläche bittet dann, die Lehrkraft selbst zu benachrichtigen.
		// Ohne Adresse am Konto geht keine Mail.
		mailStatus := mailStatusKeineAdresse
		if erledigt.AnfragendeMail != nil && *erledigt.AnfragendeMail != "" {
			mailStatus = mailStatusVersendet
			if err := t.Post().Sende(r.Context(), mailservice.Nachricht{
				An:      *erledigt.AnfragendeMail,
				Betreff: fmt.Sprintf("Ihr Klassensatz liegt bereit: %s", erledigt.TitelName),
				Text:    klassensatzBereitText(erledigt),
			}); err != nil {
				log.Printf("Klassensatz-Bereit-Mail an %s fehlgeschlagen: %v", *erledigt.AnfragendeMail, err)
				mailStatus = mailStatusFehlgeschlagen
			}
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"mail": mailStatus})
	}
}

// leseErledigtNotiz liest die freiwillige Notiz aus dem Rumpf. Ohne Rumpf ist sie leer; ein
// Rumpf, der sich nicht lesen lässt, ist ein 400.
func leseErledigtNotiz(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.Body == nil || r.ContentLength == 0 {
		return "", true
	}
	var body struct {
		Notiz string `json:"notiz"`
	}
	if !tuer.LeseAnfrage(w, r, &body) {
		return "", false
	}
	return kuerze(strings.TrimSpace(body.Notiz), 500), true
}

// klassensatzBereitText ist die Bereit-Mail. Sie trägt die Notiz der Lehrkraft, damit die
// Antwort der Bibliothek einen Bezug hat, und diese Antwort (Teilmenge, Abholort), wie die
// Mail zum Anliegen.
func klassensatzBereitText(e *repository.ErledigteReservierung) string {
	text := fmt.Sprintf("Ihre Klassensatz-Reservierung ist fertig zusammengestellt und liegt in der Bibliothek zur Abholung bereit.\n\n"+
		"  Titel:  %s\n  Klasse: %s\n  Anzahl: %d Exemplare\n",
		e.TitelName, e.Klasse, e.Anzahl)
	if e.Notiz != "" {
		text += fmt.Sprintf("  Ihre Notiz: %s\n", e.Notiz)
	}
	if e.ErledigtNotiz != "" {
		text += fmt.Sprintf("\nNotiz der Bibliothek:\n  %s\n", e.ErledigtNotiz)
	}
	return text + "\nDiese Mail wurde automatisch beim Abschliessen der Reservierung verschickt."
}

// pruefeKlassensatzAnfrage prüft die Pflichtfelder und setzt die Anzahl (Vorgabe 1,
// höchstens 200). Bei false ist die Fehlerantwort schon geschrieben.
func pruefeKlassensatzAnfrage(w http.ResponseWriter, req *klassensatzReservierungAnfrage) bool {
	// Klasse oder Kurs ist Freitext wie beim Anliegen; Trimmen und Kürzen übernimmt deshalb die
	// Tür.
	req.Klasse = kuerze(strings.TrimSpace(req.Klasse), 50)
	if req.TitelID == "" || req.Klasse == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte Titel und Klasse angeben", nil)
		return false
	}
	if req.Anzahl <= 0 {
		req.Anzahl = 1
	}
	if req.Anzahl > 200 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Höchstens 200 Exemplare je Reservierung", nil)
		return false
	}
	return true
}

// leerAlsNil macht aus einem leeren Text einen fehlenden Wert für Spalten, die NULL erlauben.
func leerAlsNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
