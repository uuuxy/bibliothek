package system

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/lmfplan"
	"bibliothek/repository"
)

// Einstellungen returns all system settings.
// @Summary      Get system settings
// @Description  Retrieves global configuration values like loan limits, grace periods, and feature flags.
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200  {object}  repository.SystemEinstellungen
// @Failure      500  {object}  map[string]string
// @Router       /einstellungen [post]
func (t *Tueren) Einstellungen(einstellungsAbfragen repository.EinstellungsAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		ctx := r.Context()
		einstellungen, err := einstellungsAbfragen.Lade(ctx)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Einstellungen", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, einstellungen)
		return nil
	})
}

// EinstellungenSchreiben persists system settings.
// @Summary      Update system settings
// @Description  Saves global configuration values. Nur mitgeschickte Felder werden geschrieben; fehlende bleiben unveraendert. Requires admin privileges.
// @Tags         system
// @Accept       json
// @Produce      json
// @Param        settings  body      repository.EinstellungenPatch  true  "Nur die Felder der gespeicherten Kategorie"
// @Success      200       {object}  map[string]string
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /einstellungen/speichern [post]
func (t *Tueren) EinstellungenSchreiben(einstellungsAbfragen repository.EinstellungsAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return t.schreibeEinstellungen(w, r, einstellungsAbfragen)
	})
}

// schreibeEinstellungen prüft den Patch, speichert ihn, protokolliert die Änderung und meldet neue
// Sitzungsfristen an die offenen Arbeitsplätze.
func (t *Tueren) schreibeEinstellungen(w http.ResponseWriter, r *http.Request, einstellungsAbfragen repository.EinstellungsAbfragen) error {
	// Patch statt vollem Objekt: Der Rumpf trägt nur die Felder einer Kategorie
	// (repository/einstellungen_patch.go). Ein volles Objekt dekodierte hier zu lauter Nullwerten
	// und setzte beim Speichern die übrigen Kategorien zurück.
	// Streng dekodiert (LeseAnfrageStreng): Der Rumpf entsteht aus einer
	// geschlossenen Liste benannter Schlüssel, ein unbekanntes Feld ist hier also
	// immer ein Fehler — und zwar der teuerste, weil er sonst still verschwindet
	// und die Oberfläche trotzdem "gespeichert" meldet.
	var req repository.EinstellungenPatch
	if !tuer.LeseAnfrageStreng(w, r, &req) {
		return nil // Error is already sent by LeseAnfrageStreng
	}

	if req.IstLeer() {
		return apierrors.BadRequest("Es wurde keine einzige Einstellung mitgeschickt.",
			errors.New("leerer Einstellungs-Patch"))
	}
	// Zahlen: innerhalb ihrer Spanne oder gar nicht gespeichert
	// (repository/einstellungen_zahlen.go). Beide Prüfungen stehen vor dem
	// Protokoll-Eintrag, damit dort nicht die Eingabe steht, während in der Datenbank
	// etwas anderes liegt.
	if err := req.PruefeZahlen(); err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}
	if err := normalisiereLernmittelAngaben(&req); err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}

	if err := einstellungsAbfragen.Schreibe(r.Context(), &req); err != nil {
		return apierrors.Internal("Fehler beim Speichern der Einstellungen", err)
	}
	t.protokolliereEinstellungen(r, req)

	// Die Sitzungsfristen holt jeder Tab nur einmal beim Anmelden (App.svelte).
	// Das Signal erreicht auch offene Tabs an anderen Arbeitsplätzen; die Werte
	// holen sich die Clients per GET /api/einstellungen/sitzung — dort sitzt die
	// Vorgaben-Logik, hier wäre sie dupliziert.
	if t.Broker != nil && (req.ThekeLeerenMinuten != nil || req.SperreMinuten != nil) {
		t.Broker.Broadcast("sitzungsfristen", "{}")
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

// normalisiereLernmittelAngaben bringt Sommerferien, Eingangsjahrgänge und Stichtag in
// Normalform. An ihnen hängt jede Lernmittel-Frist: Unlesbares wird nicht gespeichert, statt
// beim Lesen still auf die Vorgabe zurückzufallen.
func normalisiereLernmittelAngaben(req *repository.EinstellungenPatch) error {
	if req.Sommerferien != nil {
		norm, err := lmfplan.NormalisiereSommerferien(*req.Sommerferien)
		if err != nil {
			return err
		}
		req.Sommerferien = &norm
	}
	if req.LMFEingangsjahrgaenge != nil {
		norm, err := repository.NormalisiereEingangsjahrgaenge(*req.LMFEingangsjahrgaenge)
		if err != nil {
			return err
		}
		req.LMFEingangsjahrgaenge = &norm
	}
	if req.LMFStichtag != nil {
		norm, err := repository.NormalisiereLMFStichtag(*req.LMFStichtag)
		if err != nil {
			return err
		}
		req.LMFStichtag = &norm
	}
	return nil
}

// protokolliereEinstellungen schreibt den gespeicherten Patch ins Protokoll; die IP-Adresse
// wird nicht gespeichert.
func (t *Tueren) protokolliereEinstellungen(r *http.Request, req repository.EinstellungenPatch) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	details, merr := json.Marshal(req)
	if merr != nil {
		log.Printf("audit: Settings-Details konnten nicht serialisiert werden: %v", merr)
		return
	}
	t.SchreibeAdminProtokoll(r.Context(), claims.UserID, "UPDATE_SETTINGS", "", string(details))
}

// sitzungsEinstellungen sind die zwei Inaktivitäts-Fristen des Clients: Minuten bis die
// Theken-Ansicht den geladenen Schüler fallen lässt, Minuten bis zum Sperrbildschirm.
// 0 = aus.
type sitzungsEinstellungen struct {
	ThekeLeerenMinuten int `json:"theke_leeren_minuten"`
	SperreMinuten      int `json:"sperre_minuten"`
}

// SitzungsEinstellungen liefert die Sitzungs-Fristen an jeden angemeldeten Client.
func (t *Tueren) SitzungsEinstellungen(einstellungsAbfragen repository.EinstellungsAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		einstellungen, err := einstellungsAbfragen.Lade(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Einstellungen", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, sitzungsEinstellungen{
			ThekeLeerenMinuten: repository.TageOderVorgabe(einstellungen.ThekeLeerenMinuten, repository.ThekeLeerenMinutenVorgabe),
			SperreMinuten:      repository.TageOderVorgabe(einstellungen.SperreMinuten, repository.SperreMinutenVorgabe),
		})
		return nil
	})
}
