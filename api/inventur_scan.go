package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// ladeExemplarFuerScan lädt die für die Inventur-Logik nötigen Exemplardetails.
// ok=false: die Fehlerantwort (404 bei unbekanntem Barcode, sonst 500) wurde bereits geschrieben.
func ladeExemplarFuerScan(ctx context.Context, invRepo *repository.InventoryRepository, w http.ResponseWriter, barcodeID string) (*repository.InventoryScanResult, bool) {
	res, err := invRepo.GetExemplarForInventoryScan(ctx, barcodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return nil, false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	return res, true
}

// inventurWarnungen sammelt nicht-blockierende Hinweise zu einem gescannten Exemplar.
// Der Scope ist KEINE Warnung mehr, sondern eine harte Abweisung (siehe Handler): ein
// Buch außerhalb des Scopes darf gar nicht erst in dieser Session verbucht werden.
func inventurWarnungen(isLent bool) []string {
	var warnungen []string
	if isLent {
		warnungen = append(warnungen, "Buch ist laut System aktuell ausgeliehen.")
	}
	return warnungen
}

// InventurScanRequest is the payload for checking in an item during inventory.
type InventurScanRequest struct {
	SessionID string `json:"session_id"`
	BarcodeID string `json:"barcode_id"`
}

// InventurScanResponse provides feedback to the frontend after a scan attempt.
type InventurScanResponse struct {
	BarcodeID string   `json:"barcode_id"`
	Titel     string   `json:"titel"`
	CoverURL  string   `json:"cover_url,omitempty"`
	Status    string   `json:"status"`
	Warnungen []string `json:"warnungen,omitempty"`
}

// InventurScanHandler verbucht einen Exemplar-Scan in einer laufenden Session.
// @Summary      Scan a copy during inventory
// @Description  Records that a physical copy was present, bound to the given session.
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Param        body  body      InventurScanRequest   true  "Session and barcode"
// @Success      200   {object}  InventurScanResponse
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /inventur/scan [post]
func (s *Server) InventurScanHandler() http.HandlerFunc {
	return s.handleInventurScan
}

// handleInventurScan verbucht einen einzelnen Inventur-Scan. Bewusst als Top-Level-Methode
// statt Inline-Closure, damit die vielen Frühabbrüche nicht zusätzlich als Verschachtelung
// zählen (SonarQube S3776).
func (s *Server) handleInventurScan(w http.ResponseWriter, r *http.Request) {
	var req InventurScanRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}
	if req.SessionID == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("session_id fehlt"))
		return
	}

	ctx := r.Context()
	invRepo := repository.NewInventoryRepository(s.DB.Pool)

	// Session muss offen sein — sonst ist der Scan gegenstandslos (404).
	session, err := invRepo.LadeInventurSession(ctx, req.SessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("keine laufende Inventur zu dieser Session"))
			return
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	res, ok := ladeExemplarFuerScan(ctx, invRepo, w, req.BarcodeID)
	if !ok {
		return
	}
	if res.IsAusgesondert {
		apierrors.SendHTTPError(w, http.StatusConflict, fmt.Errorf("exemplar %s ist bereits ausgesondert", req.BarcodeID))
		return
	}

	imScope, err := invRepo.ExemplarImScope(ctx, res.CopyID, session.Scope())
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// Cross-Contamination-Schutz: Ein Exemplar außerhalb des Session-Scopes darf NICHT
	// in dieser Session verbucht werden (409, nicht stillschweigend absorbieren). Täte es
	// das doch, läge der Scan session-gebunden im fremden Scope — beim Abschluss der
	// ZUSTÄNDIGEN Fach-Session fehlte das Exemplar dann in deren Erfassungen und würde
	// dort fälschlich als VERLUST gebucht, obwohl das Buch physisch vorliegt.
	//
	// Bewusst als STRUKTURIERTE 409-Antwort (nicht als rohe Fehlermeldung): Das Buch
	// existiert ja, es liegt nur im falschen Scope. Das Frontend soll den echten Titel und
	// den Warntext zeigen — nicht "Unbekanntes Buch" — und den Scan als Warnung rendern,
	// ohne ihn mitzuzählen. Der Status "ausser_scope" macht den Fall clientseitig
	// unterscheidbar.
	if !imScope {
		RespondJSON(w, http.StatusConflict, InventurScanResponse{
			BarcodeID: req.BarcodeID,
			Titel:     res.Title,
			CoverURL:  res.CoverURL,
			Status:    "ausser_scope",
			Warnungen: []string{"Buch gehört nicht zum Scope dieser Inventur — es wurde NICHT erfasst. Bitte im zuständigen Inventur-Bereich scannen."},
		})
		return
	}

	if err := invRepo.RecordInventurScan(ctx, req.SessionID, res.CopyID); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	RespondJSON(w, http.StatusOK, InventurScanResponse{
		BarcodeID: req.BarcodeID,
		Titel:     res.Title,
		CoverURL:  res.CoverURL,
		Status:    "erfasst",
		Warnungen: inventurWarnungen(res.IsLent),
	})
}
