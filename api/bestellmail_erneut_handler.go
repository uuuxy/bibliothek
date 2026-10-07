package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// auditBestellmailErneutGesendet ist die Aktion im Protokoll.
const auditBestellmailErneutGesendet = "BESTELLMAIL_ERNEUT_GESENDET"

// SendeBestellmailErneutHandler schickt die Mail einer Bestellung noch einmal, deren Versand
// gescheitert ist. Die Mail entsteht aus der gespeicherten Bestellung über denselben Weg wie
// die erste (sendeBestellmail) und geht an die heutige Adresse des Lieferanten.
//
// @Summary      Resend the order e-mail after a failed dispatch
// @Tags         orders
// @Produce      json
// @Param        id   path      string  true  "Order ID"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      502  {object}  map[string]string
// @Router       /bestellungen/{id}/mail [post]
func (s *Server) SendeBestellmailErneutHandler(pdfSvc *PDFService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.sendeBestellmailErneut(w, r, pdfSvc)
	}
}

// sendeBestellmailErneut steht wie bestaetigenBestellung auf der obersten Ebene, eine
// Closure zählt für die Komplexitätsmessung als eigene Ebene.
func (s *Server) sendeBestellmailErneut(w http.ResponseWriter, r *http.Request, pdfSvc *PDFService) {
	id := r.PathValue("id")
	ctx := r.Context()

	// Der Auftrag nimmt der Bestellung den Vermerk: Nur eine Anfrage sendet, ein zweiter
	// Klick bekommt 409. Jeder Ausgang ohne Mail setzt den Vermerk wieder.
	auftrag, err := repository.BeanspruchBestellmail(ctx, s.DB.Pool, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
		return
	case errors.Is(err, repository.ErrBestellmailNichtOffen):
		//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusConflict, errors.New(
			"Für diese Bestellung ist kein gescheiterter Versand vermerkt. Die Mail wird nicht noch einmal gesendet."))
		return
	case err != nil:
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if !smtpKonfiguriert() {
		s.merkeBestellmailGescheitert(ctx, id)
		// 400 statt 500: Eine 500-Meldung ersetzt der Sanitizer, und der Hinweis, was zu
		// tun ist, erreichte den Bildschirm nicht.
		//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New(
			"Es ist kein Mailserver eingerichtet (Einstellungen → Mail). Die Bestellmail wurde nicht gesendet."))
		return
	}

	daten, err := s.baueBestellmailErneut(ctx, id, auftrag)
	if err != nil {
		s.merkeBestellmailGescheitert(ctx, id)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	mitLink, err := s.sendeBestellmail(ctx, pdfSvc, daten)
	if err != nil {
		s.merkeBestellmailGescheitert(ctx, id)
		// 502: Gescheitert ist der Mailserver, nicht dieser Server. Die Meldung bleibt lesbar.
		//nolint:staticcheck // ST1005: ganze Sätze, diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusBadGateway, fmt.Errorf(
			"Der Versand an %s ist erneut gescheitert. Die Bestellung bleibt als nicht versendet vermerkt.", auftrag.Empfaenger))
		return
	}

	s.merkeBestellmailVersendet(ctx, id, auftrag.Empfaenger)
	s.protokolliereVerwaltung(ctx, auditBestellmailErneutGesendet, map[string]any{"bestellung_id": id})

	status, meldung := bestellVersandMeldung(auftrag.LieferantName, daten.IstHauptlieferant && !mitLink)
	RespondJSON(w, http.StatusOK, map[string]any{"status": status, "message": meldung})
}

// baueBestellmailErneut stellt aus der gespeicherten Bestellung zusammen, was die erste Mail
// aus dem Warenkorb bekam: Positionen, die Etiketten der Positionen mit Vorab-Barcode und,
// wo die Bestellung einen Bestätigungsschritt hat, einen neuen Link. Der alte Link ist nicht
// wiederherstellbar, gespeichert ist nur sein Hash.
func (s *Server) baueBestellmailErneut(ctx context.Context, bestellungID string, auftrag *repository.BestellmailAuftrag) (bestellmailDaten, error) {
	daten := bestellmailDaten{
		Empfaenger:   auftrag.Empfaenger,
		Kundennummer: auftrag.Kundennummer,
		Mittel:       auftrag.Mittel,
	}
	for _, p := range auftrag.Positionen {
		daten.Positionen = append(daten.Positionen, OrderedItem{
			Titel: p.Titel, Autor: p.Autor, ISBN: p.ISBN, Verlag: p.Verlag, Menge: p.Menge,
		})
		daten.Exemplare += p.Menge
	}

	etiketten, err := s.ladeBestellEtiketten(ctx, bestellungID)
	if err != nil {
		return bestellmailDaten{}, err
	}
	daten.Etiketten = etiketten

	mitBestaetigung, err := s.bestellungImBestaetigungsweg(ctx, bestellungID)
	if err != nil {
		return bestellmailDaten{}, err
	}
	daten.IstHauptlieferant = mitBestaetigung
	if mitBestaetigung && s.oeffentlicheAdresse(ctx) != "" {
		token, gueltigBis, err := s.erneuereBestaetigungsToken(ctx, bestellungID)
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
func (s *Server) merkeBestellmailVersendet(ctx context.Context, bestellungID, empfaenger string) {
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), fristBestellmailVermerk)
	defer abbruch()
	if err := repository.MerkeBestellmailVersendet(ctx, s.DB.Pool, bestellungID, empfaenger); err != nil {
		log.Printf("Bestellung %s: Empfänger des erneuten Versands nicht vermerkt: %v", bestellungID, err)
	}
}
