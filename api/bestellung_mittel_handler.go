package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Der Rückweg für den Topf einer Bestellung (Raster-Durchgang 10.09.2026, Frage 10).
//
// Eine Bestellung im falschen Topf ließ sich bis hierher nicht korrigieren: Der Topf
// wurde beim Auslösen geschrieben, und danach gab es keine Tür mehr. Für die Berichte,
// die nach Topf rechnen, stünde sie dauerhaft im falschen Block — und Alt-Bestellungen
// „ohne Zuordnung" (Migration 109, Backfill nur bei eindeutigen Fällen) blieben es für
// immer.
//
// Was die Korrektur NICHT tut: Sie ändert weder das Anschreiben, das der Händler bekommen
// hat, noch die Kundennummer auf dem Beleg — beides ist die Abschrift dessen, was
// rausging. Korrigiert wird die eigene Zuordnung, aus der die Schule rechnet. Deshalb
// ist der Grund Pflicht und der Eingriff steht im Admin-Audit-Log mit von/nach.

// MittelKorrekturRequest ist die Eingabe für UpdateBestellungMittelHandler.
type MittelKorrekturRequest struct {
	Mittel string `json:"mittel"`
	Grund  string `json:"grund"`
}

// auditBestellungMittelKorrigiert ist die Aktion im Admin-Audit-Log.
const auditBestellungMittelKorrigiert = "BESTELLUNG_MITTEL_KORRIGIERT"

// UpdateBestellungMittelHandler setzt den Topf einer bestehenden Bestellung neu.
//
// @Summary      Correct the funding pot of an existing order
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Order ID"
// @Param        body  body      MittelKorrekturRequest  true  "New pot and reason"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /bestellungen/{id}/mittel [put]
func (s *Server) UpdateBestellungMittelHandler() http.HandlerFunc {
	return s.korrigiereBestellungMittel
}

// korrigiereBestellungMittel steht wie bestaetigenBestellung auf der obersten Ebene —
// eine Closure zählt für die Komplexitätsmessung als eigene Ebene.
func (s *Server) korrigiereBestellungMittel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing bestellung id"))
		return
	}

	var req MittelKorrekturRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}
	if !repository.MittelGueltig(req.Mittel) {
		apierrors.SendHTTPError(w, http.StatusBadRequest, ErrMittelUngueltig)
		return
	}
	grund := strings.TrimSpace(req.Grund)
	if grund == "" {
		//nolint:staticcheck // ST1005: ganze Sätze — diese Meldung steht so vor der Bibliothekskraft.
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("Bitte einen Grund für die Korrektur angeben."))
		return
	}

	vorher, err := repository.KorrigiereBestellungMittel(r.Context(), s.DB.Pool, id, req.Mittel)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("bestellung not found"))
		return
	}
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	von := ""
	if vorher != nil {
		von = *vorher
	}
	geaendert := von != req.Mittel
	if geaendert {
		s.auditiereMittelKorrektur(r, id, von, req.Mittel, grund)
	}

	RespondJSON(w, http.StatusOK, map[string]any{
		"id":        id,
		"mittel":    req.Mittel,
		"von":       von,
		"geaendert": geaendert,
	})
}

// auditiereMittelKorrektur protokolliert die Korrektur — dasselbe Muster wie der LMF-Plan
// (lmf_plan_audit.go): ohne Anmeldung oder ohne Datenbank gibt es nichts zu schreiben.
func (s *Server) auditiereMittelKorrektur(r *http.Request, bestellungID, von, nach, grund string) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		return
	}
	if s.DB == nil || s.DB.Pool == nil {
		return
	}
	details := map[string]any{"bestellung_id": bestellungID, "von": von, "nach": nach, "grund": grund}
	if err := repository.NewAuditRepository(s.DB.Pool).
		LogAdminAktion(r.Context(), claims.UserID, auditBestellungMittelKorrigiert, getIP(r), details); err != nil {
		log.Printf("Audit %s fehlgeschlagen: %v", auditBestellungMittelKorrigiert, err)
	}
}
