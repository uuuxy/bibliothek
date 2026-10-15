package bestellung

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/service"
	"bibliothek/inventur"
	"bibliothek/mailservice"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"
)

// bestellPosition ist eine Zeile des Warenkorbs.
type bestellPosition struct {
	TitelID         string  `json:"titel_id" validate:"required,uuid_oder_leer"`
	Menge           int     `json:"menge"`
	Preis           float64 `json:"preis"`
	MitVorabBarcode bool    `json:"generate_barcodes"`
}

// bestellungAnlegenAnfrage ist ein kompletter Warenkorb an genau einen Lieferanten. Die Bestellung
// entsteht daraus in einer Transaktion — schlägt eine Position fehl, geht keine raus.
type bestellungAnlegenAnfrage struct {
	LieferantID string            `json:"supplier_id" validate:"omitempty,uuid_oder_leer"`
	Positionen  []bestellPosition `json:"items" validate:"dive"`
	// IdempotenzSchluessel: von der Oberfläche je Absenden vergeben. Ein Doppelklick schickt
	// denselben Schlüssel; die zweite Anfrage bewirkt nichts (keine zweite Bestellung, keine
	// zweite Lieferanten-Mail). Wahlweise: Ohne Schlüssel läuft die Anfrage ohne diesen Schutz.
	IdempotenzSchluessel string `json:"idempotency_key,omitempty"`
	// Mittel: der Topf dieser Bestellung — 'land' (Lernmittelfreiheit) oder
	// 'schultraeger' (Schülerbücherei), siehe pkg/mitteltopf. Pflicht: Der Vermerk
	// muss auf der Bestellung stehen, und ein Standardwert wäre eine stille Zuordnung
	// zum falschen Topf. Ein gemischter Warenkorb schickt zwei Anfragen — eine je Topf.
	Mittel string `json:"mittel"`
}

// bestellAuftrag füllt den Auftrag an das Anlegen der Bestellung aus der Anfrage.
func bestellAuftrag(req bestellungAnlegenAnfrage) service.BestellAuftrag {
	positionen := make([]service.BestellAuftragPosition, 0, len(req.Positionen))
	for _, position := range req.Positionen {
		positionen = append(positionen, service.BestellAuftragPosition(position))
	}
	return service.BestellAuftrag{
		LieferantID:          req.LieferantID,
		Positionen:           positionen,
		IdempotenzSchluessel: req.IdempotenzSchluessel,
		Mittel:               req.Mittel,
	}
}

// BestellungAnlegen legt die Bestellung aus dem Warenkorb an und verschickt die Bestellmail.
func (t *Tueren) BestellungAnlegen(bestellDienst *service.BestellDienst) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.legeBestellungAn(w, r, bestellDienst)
	}
}

// legeBestellungAn speichert die Bestellung und verschickt danach die Mail an den Lieferanten;
// die Bestellung gilt auch, wenn der Versand ausbleibt.
func (t *Tueren) legeBestellungAn(w http.ResponseWriter, r *http.Request, bestellDienst *service.BestellDienst) {
	var req bestellungAnlegenAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	if req.LieferantID == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("supplier_id is required"))
		return
	}
	if len(req.Positionen) == 0 {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("order cart cannot be empty"))
		return
	}
	if !mitteltopf.IstGueltig(req.Mittel) {
		apierrors.SendHTTPError(w, http.StatusBadRequest, mitteltopf.ErrUngueltig)
		return
	}

	ctx := r.Context()

	res, err := bestellDienst.LegeAn(ctx, bestellAuftrag(req))
	if err != nil {
		apierrors.SendHTTPError(w, statusDesBestellfehlers(err), err)
		return
	}

	// Doppelklick: dieselbe Bestellung lief schon durch — keine zweite Mail, keine
	// zweite Bestellung. Die erste Anfrage hat Mail und Etiketten bereits erledigt.
	if res.BereitsVorhanden {
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung an %s war bereits erfasst (Doppelklick) — es wurde keine zweite Bestellung ausgelöst.", res.LieferantName),
			"ordered_qty": res.Menge,
		})
		return
	}

	if !mailservice.SMTPKonfiguriert() {
		log.Println("WARNUNG: Kein (echter) SMTP-Server hinterlegt. E-Mail-Versand übersprungen, die Bestellung ist gespeichert.")
		t.merkeBestellmailGescheitert(ctx, res.BestellungID)
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung erfasst (E-Mail-Versand an %s übersprungen - SMTP nicht konfiguriert).", res.LieferantName),
			"ordered_qty": res.Menge,
		})
		return
	}

	mitLink, err := t.sendeBestellmail(ctx, bestellmailDatenAus(res))
	if err != nil {
		// Die Bestellung ist gespeichert, die Mail nicht raus: Der Vermerk an der Bestellung
		// bleibt, wenn diese Meldung vom Bildschirm verschwunden ist.
		t.merkeBestellmailGescheitert(ctx, res.BestellungID)
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":      "warning",
			"message":     fmt.Sprintf("Bestellung gespeichert, aber E-Mail-Versand an %s fehlgeschlagen.", res.LieferantEmail),
			"ordered_qty": res.Menge,
		})
		return
	}

	status, meldung := bestellVersandMeldung(res.LieferantName, res.IstHauptlieferant && !mitLink)
	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"status":      status,
		"message":     meldung,
		"ordered_qty": res.Menge,
	})
}

// statusDesBestellfehlers ordnet die benannten Ablehnungen von BestellDienst.LegeAn ihrem Status zu.
// Alles andere ist ein Serverfehler, dessen Text die Tür nicht ausgibt.
func statusDesBestellfehlers(err error) int {
	switch {
	case errors.Is(err, service.ErrLieferantUnbekannt), errors.Is(err, service.ErrTitelUnbekannt):
		return http.StatusNotFound
	case errors.Is(err, mitteltopf.ErrUngueltig), errors.Is(err, service.ErrMengeUngueltig):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Zulauf liefert die bestellten, noch nicht eingetroffenen Exemplare, je Bestellung
// gruppiert (service.LadeZulauf).
func (t *Tueren) Zulauf() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		gruppen, err := service.LadeZulauf(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, gruppen)
	}
}

// bestellSucheAnfrage trägt den Suchbegriff der Bestellsuche — ISBN oder Freitext.
type bestellSucheAnfrage struct {
	Suchtext string `json:"query"`
}

// dnbAusfallKopf sagt der Bestellsuche, dass die DNB nicht geantwortet hat und die Liste nur
// den eigenen Katalog trägt. Ein Kopf statt eines Felds, weil die Antwort eine Liste ist, wie
// X-Treffer-Gesamt an der Katalogsuche (api/portal/opac.go).
const dnbAusfallKopf = "X-DNB-Ausfall"

// BestellSuche sucht einen Titel zum Bestellen, zuerst im eigenen Katalog und
// zusätzlich bei der DNB. Der Metadaten-Client wird einmal beim Registrieren der Route
// gebaut, nicht je Anfrage — er hält seinen eigenen HTTP-Client mit Zeitgrenze.
func (t *Tueren) BestellSuche() http.HandlerFunc {
	return t.sucheBestellung(inventur.NeuerMetadatenClient())
}

// sucheBestellung ist die Tür mit ihrem Client für die DNB als Parameter: Ein Test stellt
// die Antwort der DNB nach, wie bei isbnZuTitelTuer in api/katalog.
func (t *Tueren) sucheBestellung(metadaten *inventur.MetadatenClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req bestellSucheAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		suchtext := strings.TrimSpace(req.Suchtext)
		if suchtext == "" {
			tuer.AntworteJSON(w, http.StatusOK, []service.BestellTreffer{})
			return
		}

		ctx := r.Context()
		treffer, dnbAusfall, err := service.FindeZumBestellen(ctx, t.DB.Pool, metadaten, suchtext)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if dnbAusfall {
			w.Header().Set(dnbAusfallKopf, "1")
		}

		tuer.AntworteJSON(w, http.StatusOK, treffer)
	}
}

// wareneingangBuchenAnfrage sind die Exemplare, die aus dem Zulauf in den Bestand gehen.
type wareneingangBuchenAnfrage struct {
	ExemplarIDs []string `json:"exemplar_ids" validate:"omitempty,dive,required,uuid_oder_leer"`
}

// WareneingangBuchen bucht die markierten Exemplare aus dem Zulauf in den Bestand.
func (t *Tueren) WareneingangBuchen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req wareneingangBuchenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if len(req.ExemplarIDs) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("no exemplar_ids provided"))
			return
		}

		ctx := r.Context()
		var bearbeiterID string
		if claims, ok := auth.GetClaims(ctx); ok {
			bearbeiterID = claims.UserID
		}

		protokollAbfragen := repository.NewProtokollAbfragen(t.DB.Pool)

		eingebucht, err := service.BucheWareneingang(ctx, t.DB.Pool, protokollAbfragen, service.WareneingangAngaben{
			ExemplarIDs:  req.ExemplarIDs,
			BearbeiterID: bearbeiterID,
			IPAdresse:    clientip.AusAnfrage(r),
		})
		if err != nil {
			if errors.Is(err, service.ErrNichtsEinzubuchen) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":         "success",
			"received_count": len(eingebucht),
			"received_items": eingebucht,
		})
	}
}
