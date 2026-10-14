package bestellung

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/buchetikett"
	"bibliothek/mailservice"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// aktionBestellmailErneutGesendet ist die Aktion im Protokoll.
const aktionBestellmailErneutGesendet = "BESTELLMAIL_ERNEUT_GESENDET"

// BestellmailErneutSenden schickt die Mail einer Bestellung noch einmal, deren Versand
// gescheitert ist. Die Mail entsteht aus der gespeicherten Bestellung über denselben Weg wie
// die erste (sendeBestellmail) und geht an die heutige Adresse des Lieferanten.
//
// @Summary      Bestellmail nach gescheitertem Versand erneut senden
// @Tags         orders
// @Produce      json
// @Param        id   path      string  true  "Kennung der Bestellung"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      502  {object}  map[string]string
// @Router       /bestellungen/{id}/mail [post]
func (t *Tueren) BestellmailErneutSenden() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.sendeBestellmailErneut(w, r)
	}
}

// sendeBestellmailErneut steht wie trageBestaetigungNach auf der obersten Ebene, eine
// Closure zählt für die Komplexitätsmessung als eigene Ebene.
func (t *Tueren) sendeBestellmailErneut(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	// Der Auftrag nimmt der Bestellung den Vermerk: Nur eine Anfrage sendet, ein zweiter
	// Klick bekommt 409. Jeder Ausgang ohne Mail setzt den Vermerk wieder.
	auftrag, err := repository.BeanspruchBestellmail(ctx, t.DB.Pool, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
		return
	case errors.Is(err, repository.ErrBestellmailNichtOffen):
		//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusConflict, errors.New(
			"Für diese Bestellung ist kein gescheiterter Versand offen: Die Mail ging raus, oder die Bestellung ist bestätigt. Sie wird nicht noch einmal gesendet."))
		return
	case err != nil:
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if !mailservice.SMTPKonfiguriert() {
		t.merkeBestellmailGescheitert(ctx, id)
		// 400 statt 500: Eine 500-Meldung ersetzt der Sanitizer, und der Hinweis, was zu
		// tun ist, erreichte den Bildschirm nicht.
		//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(
			"Es ist kein Mailserver eingerichtet (Einstellungen → Mail). Die Bestellmail wurde nicht gesendet."))
		return
	}

	daten, err := t.baueBestellmailErneut(ctx, id, auftrag)
	if err != nil {
		t.merkeBestellmailGescheitert(ctx, id)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	mitLink, err := t.sendeBestellmail(ctx, daten)
	if err != nil {
		t.merkeBestellmailGescheitert(ctx, id)
		// 502: Gescheitert ist der Mailserver, nicht dieser Server. Die Meldung bleibt lesbar.
		//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusBadGateway, fmt.Errorf(
			"Der Versand an %s ist erneut gescheitert. Die Bestellung bleibt als nicht versendet vermerkt.", auftrag.Empfaenger))
		return
	}

	t.merkeBestellmailVersendet(ctx, id, auftrag.Empfaenger)
	t.ProtokolliereVerwaltung(ctx, aktionBestellmailErneutGesendet, map[string]any{"bestellung_id": id})

	status, meldung := bestellVersandMeldung(auftrag.LieferantName, daten.IstHauptlieferant && !mitLink)
	tuer.AntworteJSON(w, http.StatusOK, map[string]any{"status": status, "message": meldung})
}

// aktionBestellmailVermerkEntfernt ist die Aktion im Protokoll.
const aktionBestellmailVermerkEntfernt = "BESTELLMAIL_VERMERK_ENTFERNT"

// BestellmailVermerkNehmen entfernt den Vermerk über den gescheiterten Versand, ohne zu
// senden: Die Bibliothek hat den Händler auf anderem Weg erreicht, etwa am Telefon. „Erneut
// senden" brächte die Bestellung sonst ein zweites Mal zu ihm.
//
// @Summary      Vermerk über den gescheiterten Versand entfernen, ohne zu senden
// @Tags         orders
// @Produce      json
// @Param        id   path      string  true  "Kennung der Bestellung"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /bestellungen/{id}/mail [delete]
func (t *Tueren) BestellmailVermerkNehmen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ctx := r.Context()

		err := repository.NimmBestellmailVermerk(ctx, t.DB.Pool, id)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
			return
		case errors.Is(err, repository.ErrBestellmailMitBestaetigung):
			//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(
				"Diese Bestellung ging mit einem Bestätigungs-Link raus. Hat der Händler sie auf anderem Weg erhalten, lässt sich seine Zusage in der Bestellung nachtragen."))
			return
		case errors.Is(err, repository.ErrBestellmailNichtOffen):
			//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(
				"Für diese Bestellung ist kein gescheiterter Versand offen: Die Mail ging raus, oder der Hinweis ist schon entfernt."))
			return
		case err != nil:
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		t.ProtokolliereVerwaltung(ctx, aktionBestellmailVermerkEntfernt, map[string]any{"bestellung_id": id})
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Der Hinweis ist entfernt. Die Bestellung wurde nicht noch einmal gesendet.",
		})
	}
}

// baueBestellmailErneut stellt aus der gespeicherten Bestellung zusammen, was die erste Mail
// aus dem Warenkorb bekam: Positionen, die Etiketten der Positionen mit Vorab-Barcode und,
// wo die Bestellung einen Bestätigungsschritt hat, einen neuen Link. Der alte Link ist nicht
// wiederherstellbar, gespeichert ist nur sein Hash.
func (t *Tueren) baueBestellmailErneut(ctx context.Context, bestellungID string, auftrag *repository.BestellmailAuftrag) (bestellmailDaten, error) {
	daten := bestellmailDaten{
		Empfaenger:   auftrag.Empfaenger,
		Kundennummer: auftrag.Kundennummer,
		Mittel:       auftrag.Mittel,
	}
	for _, p := range auftrag.Positionen {
		daten.Positionen = append(daten.Positionen, bestellterTitel{
			Titel: p.Titel, Autor: p.Autor, ISBN: p.ISBN, Verlag: p.Verlag, Menge: p.Menge,
		})
		daten.Exemplare += p.Menge
	}

	etiketten, err := buchetikett.LadeDerBestellung(ctx, t.DB.Pool, bestellungID)
	if err != nil {
		return bestellmailDaten{}, err
	}
	daten.Etiketten = etiketten

	mitBestaetigung, err := t.bestellungImBestaetigungsweg(ctx, bestellungID)
	if err != nil {
		return bestellmailDaten{}, err
	}
	daten.IstHauptlieferant = mitBestaetigung
	if mitBestaetigung && t.oeffentlicheAdresse(ctx) != "" {
		token, gueltigBis, err := t.erneuereBestaetigungsToken(ctx, bestellungID)
		if err != nil {
			return bestellmailDaten{}, err
		}
		daten.Token = token
		daten.LinkGueltigBis = &gueltigBis
	}
	return daten, nil
}

// merkeBestellmailVersendet hält an der Bestellung fest, an welche Adresse die Mail ging.
// Wie der Vermerk über den gescheiterten Versand hängt das Schreiben nicht an der Anfrage.
func (t *Tueren) merkeBestellmailVersendet(ctx context.Context, bestellungID, empfaenger string) {
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), fristBestellmailVermerk)
	defer abbruch()
	if err := repository.MerkeBestellmailVersendet(ctx, t.DB.Pool, bestellungID, empfaenger); err != nil {
		log.Printf("Bestellung %s: Empfänger des erneuten Versands nicht vermerkt: %v", bestellungID, err)
	}
}
