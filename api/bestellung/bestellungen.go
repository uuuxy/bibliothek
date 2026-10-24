package bestellung

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/katalogdienste"
	"bibliothek/internal/service"
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
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte einen Lieferanten wählen", nil)
		return
	}
	if len(req.Positionen) == 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Der Warenkorb ist leer", nil)
		return
	}
	if !mitteltopf.IstGueltig(req.Mittel) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, mitteltopf.MeldungUngueltig, nil)
		return
	}

	ctx := r.Context()

	res, err := bestellDienst.LegeAn(ctx, bestellAuftrag(req))
	if err != nil {
		antworteAufBestellfehler(w, err)
		return
	}

	// Doppelklick: dieselbe Bestellung lief schon durch — keine zweite Mail, keine
	// zweite Bestellung. Die erste Anfrage hat Mail und Etiketten bereits erledigt.
	if res.BereitsVorhanden {
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Die Bestellung an %s war bereits erfasst, etwa nach einem Doppelklick. Eine zweite ist nicht ausgelöst.", res.LieferantName),
			"ordered_qty": res.Menge,
		})
		return
	}

	if !t.Post().IstKonfiguriert(ctx) {
		slog.Warn("Kein (echter) SMTP-Server hinterlegt. E-Mail-Versand übersprungen, die Bestellung ist gespeichert.")
		t.merkeBestellmailGescheitert(ctx, res.BestellungID)
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Bestellung gespeichert. Die Mail an %s ist nicht versendet, weil kein Mailserver eingetragen ist.", res.LieferantName),
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
			"message":     fmt.Sprintf("Bestellung gespeichert, aber die Mail an %s ließ sich nicht versenden", res.LieferantEmail),
			"ordered_qty": res.Menge,
		})
		return
	}

	meldung, warnung := bestellVersandMeldung(res.LieferantName, res.IstHauptlieferant && !mitLink)
	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"status":      versandStatus(warnung),
		"message":     meldung,
		"ordered_qty": res.Menge,
	})
}

// Sätze zu den Ablehnungen eines Warenkorbs (antworteAufBestellfehler).
const (
	meldungLieferantUnbekannt = "Den Lieferanten gibt es nicht mehr. Bitte einen anderen wählen."
	meldungTitelUnbekannt     = "Der Titel steht nicht mehr im Katalog. Bitte aus dem Warenkorb nehmen."
	meldungMengeUngueltig     = "Die Menge muss zwischen 1 und 200 liegen"
)

// antworteAufBestellfehler ordnet den benannten Ablehnungen von BestellDienst.LegeAn Status und
// Satz zu, mit der Position des Warenkorbs vorn, wenn der Dienst sie nennt. Alles andere ist ein
// Serverfehler, dessen Text die Tür nicht ausgibt.
func antworteAufBestellfehler(w http.ResponseWriter, err error) {
	var status int
	var satz string
	switch {
	case errors.Is(err, service.ErrLieferantUnbekannt):
		status, satz = http.StatusNotFound, meldungLieferantUnbekannt
	case errors.Is(err, service.ErrTitelUnbekannt):
		status, satz = http.StatusNotFound, meldungTitelUnbekannt
	case errors.Is(err, mitteltopf.ErrUngueltig):
		status, satz = http.StatusBadRequest, mitteltopf.MeldungUngueltig
	case errors.Is(err, service.ErrMengeUngueltig):
		status, satz = http.StatusBadRequest, meldungMengeUngueltig
	default:
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}
	var position *service.BestellpositionFehler
	if errors.As(err, &position) {
		satz = fmt.Sprintf("Position %d: %s", position.Position, satz)
	}
	apierrors.AntworteFehlerMitMeldung(w, status, satz, err)
}

// Zulauf liefert die bestellten, noch nicht eingetroffenen Exemplare, je Bestellung
// gruppiert (service.LadeZulauf).
func (t *Tueren) Zulauf() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		gruppen, err := service.LadeZulauf(ctx, t.DB.Pool)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
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
	return t.sucheBestellung(katalogdienste.NewMetadatenClient())
}

// sucheBestellung ist die Tür mit ihrem Client für die DNB als Parameter: Ein Test stellt
// die Antwort der DNB nach, wie bei isbnZuTitelTuer in api/katalog.
func (t *Tueren) sucheBestellung(metadaten *katalogdienste.MetadatenClient) http.HandlerFunc {
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
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
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
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Keine Exemplare angegeben", nil)
			return
		}

		ctx := r.Context()
		var bearbeiterID string
		if claims, ok := auth.ClaimsAusKontext(ctx); ok {
			bearbeiterID = claims.KontoID
		}

		protokollAbfragen := repository.NewProtokollAbfragen(t.DB.Pool)

		eingebucht, err := service.BucheWareneingang(ctx, t.DB.Pool, protokollAbfragen, service.WareneingangAngaben{
			ExemplarIDs:  req.ExemplarIDs,
			BearbeiterID: bearbeiterID,
			IPAdresse:    clientip.AusAnfrage(r),
		})
		if err != nil {
			if errors.Is(err, service.ErrNichtsEinzubuchen) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound,
					"Keine Exemplare zum Einbuchen gefunden. Sind sie schon freigegeben?", err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":         "success",
			"received_count": len(eingebucht),
			"received_items": eingebucht,
		})
	}
}
