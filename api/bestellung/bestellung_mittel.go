package bestellung

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/mitteltopf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Der Rückweg für den Topf einer Bestellung.
//
// Ohne diese Tür ließe sich eine Bestellung im falschen Topf nicht korrigieren: Der Topf wird
// beim Auslösen geschrieben. Für die Berichte, die nach Topf rechnen, stünde sie dauerhaft im
// falschen Block, und Alt-Bestellungen „ohne Zuordnung" (Migration 109, Backfill nur bei
// eindeutigen Fällen) blieben es für immer.
//
// Was die Korrektur nicht tut: Sie ändert weder das Anschreiben, das der Händler bekommen hat,
// noch die Kundennummer auf dem Beleg; beides ist die Abschrift dessen, was rausging.
// Korrigiert wird die eigene Zuordnung, aus der die Schule rechnet. Deshalb ist der Grund
// Pflicht, und der Eingriff steht im Protokoll der Verwaltung mit von/nach.

// mittelKorrigierenAnfrage ist die Eingabe für BestellungMittelKorrigieren.
type mittelKorrigierenAnfrage struct {
	Mittel string `json:"mittel"`
	Grund  string `json:"grund"`
}

// aktionBestellungMittelKorrigiert ist die Aktion im Protokoll der Verwaltung.
const aktionBestellungMittelKorrigiert = "BESTELLUNG_MITTEL_KORRIGIERT"

// BestellungMittelKorrigieren setzt den Topf einer bestehenden Bestellung neu.
//
// @Summary      Topf einer bestehenden Bestellung korrigieren
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Kennung der Bestellung"
// @Param        body  body      mittelKorrigierenAnfrage  true  "Neuer Topf und Grund"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /bestellungen/{id}/mittel [put]
func (t *Tueren) BestellungMittelKorrigieren() http.HandlerFunc {
	return t.korrigiereBestellungMittel
}

// korrigiereBestellungMittel steht wie trageBestaetigungNach auf der obersten Ebene —
// eine Closure zählt für die Komplexitätsmessung als eigene Ebene.
func (t *Tueren) korrigiereBestellungMittel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("missing bestellung id"))
		return
	}

	var req mittelKorrigierenAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}
	if !mitteltopf.IstGueltig(req.Mittel) {
		apierrors.AntworteFehler(w, http.StatusBadRequest, mitteltopf.ErrUngueltig)
		return
	}
	grund := strings.TrimSpace(req.Grund)
	if grund == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte einen Grund für die Korrektur angeben.", nil)
		return
	}

	vorher, err := repository.KorrigiereBestellungMittel(r.Context(), t.DB.Pool, id, req.Mittel)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehler(w, http.StatusNotFound, errors.New("bestellung not found"))
		return
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return
	}

	von := ""
	if vorher != nil {
		von = *vorher
	}
	geaendert := von != req.Mittel
	if geaendert {
		t.protokolliereMittelKorrektur(r, id, von, req.Mittel, grund)
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"id":        id,
		"mittel":    req.Mittel,
		"von":       von,
		"geaendert": geaendert,
	})
}

// protokolliereMittelKorrektur protokolliert die Korrektur, nach demselben Muster wie der
// LMF-Plan (api/lernmittel/lmf_plan_protokoll.go): Ohne Anmeldung oder ohne Datenbank gibt es
// nichts zu schreiben.
func (t *Tueren) protokolliereMittelKorrektur(r *http.Request, bestellungID, von, nach, grund string) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return
	}
	if t.DB == nil || t.DB.Pool == nil {
		return
	}
	details := map[string]any{"bestellung_id": bestellungID, "von": von, "nach": nach, "grund": grund}
	if err := repository.NewProtokollAbfragen(t.DB.Pool).
		ProtokolliereAdminAktion(r.Context(), claims.KontoID, aktionBestellungMittelKorrigiert, clientip.AusAnfrage(r), details); err != nil {
		log.Printf("Audit %s fehlgeschlagen: %v", aktionBestellungMittelKorrigiert, err)
	}
}
