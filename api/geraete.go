package api

// geraete.go — Verwaltungs-Endpunkte der Geräteausleihe (Bereich im Medienkatalog).
//
// Rechte wie beim Buchbestand: Lesen view_books, Pflegen edit_books — Geräte SIND
// Bestand. Das G--Präfix ist Pflicht: Die Omnibox routet Scans darüber zum
// device_service; ein Gerät ohne G- wäre am Kiosk unauffindbar (die präfixlose
// Auflösung kennt nur Buch → Schüler → Lehrer → Suche).

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GeraetRequest ist die Eingabe für Anlegen und Pflegen.
type GeraetRequest struct {
	Modellname    string `json:"modellname"`
	Seriennummer  string `json:"seriennummer,omitempty"`
	BarcodeID     string `json:"barcode_id,omitempty"` // nur beim Anlegen; Barcodes kleben, sie wandern nicht
	Zubehoer      string `json:"zubehoer"`
	ZustandNotiz  string `json:"zustand_notiz,omitempty"`
	IstAusleihbar *bool  `json:"ist_ausleihbar,omitempty"`
}

// ListGeraeteHandler liefert die Geräteliste samt aktuellem Ausleiher.
// GET /api/geraete
func (s *Server) ListGeraeteHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		geraete, err := repo.ListGeraete(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Geräte", err)
		}
		RespondJSON(w, http.StatusOK, map[string]any{"data": geraete})
		return nil
	})
}

// CreateGeraetHandler legt ein Gerät an. POST /api/geraete
func (s *Server) CreateGeraetHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req GeraetRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		req.Modellname = strings.TrimSpace(req.Modellname)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)
		if req.Modellname == "" {
			return apierrors.BadRequest("Ein Modellname ist erforderlich", nil)
		}
		if !strings.HasPrefix(req.BarcodeID, "G-") || len(req.BarcodeID) < 4 {
			return apierrors.BadRequest("Der Barcode muss mit G- beginnen (z. B. G-IPAD-01) — "+
				"nur so findet der Kiosk-Scan das Gerät", nil)
		}

		id, err := repo.CreateGeraet(r.Context(), req.Modellname,
			nullableString(strings.TrimSpace(req.Seriennummer)), req.BarcodeID, strings.TrimSpace(req.Zubehoer))
		if err != nil {
			if errors.Is(err, repository.ErrGeraetBarcodeVergeben) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Anlegen des Geräts", err)
		}
		RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
		return nil
	})
}

// UpdateGeraetHandler pflegt Stammdaten und den Defekt-Status.
// PUT /api/geraete/{id}
func (s *Server) UpdateGeraetHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req GeraetRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		req.Modellname = strings.TrimSpace(req.Modellname)
		if req.Modellname == "" {
			return apierrors.BadRequest("Ein Modellname ist erforderlich", nil)
		}
		istAusleihbar := true
		if req.IstAusleihbar != nil {
			istAusleihbar = *req.IstAusleihbar
		}

		err := repo.UpdateGeraet(r.Context(), r.PathValue("id"), req.Modellname,
			strings.TrimSpace(req.Zubehoer), nullableString(strings.TrimSpace(req.ZustandNotiz)), istAusleihbar)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Gerät nicht gefunden", err)
			}
			return apierrors.Internal("Fehler beim Speichern des Geräts", err)
		}
		RespondSuccess(w)
		return nil
	})
}
