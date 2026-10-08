package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/lusd"
	"bibliothek/pkg/closeutil"
	"bibliothek/repository"
)

// abgaengerKarenzTage liest die Karenzzeit aus den Einstellungen; ohne lesbare
// Einstellungen gilt die Vorgabe — derselbe Rückfall wie im nächtlichen Job.
func (s *Server) abgaengerKarenzTage(ctx context.Context) int {
	einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return repository.StandardAbgaengerKarenzTage
	}
	return repository.AbgaengerKarenzTageOderStandard(einst)
}

// readLusdUpload liest die hochgeladene Datei und übergibt sie dem Parser des Pakets lusd.
func readLusdUpload(r *http.Request) (lusd.Datei, error) {
	file, _, err := r.FormFile("csvFile")
	if err != nil {
		return lusd.Datei{}, fmt.Errorf("CSV-Datei fehlt: %w", err)
	}
	defer closeutil.LogClose(file, "lusd upload")

	content, err := io.ReadAll(file)
	if err != nil {
		return lusd.Datei{}, fmt.Errorf("CSV konnte nicht gelesen werden: %w", err)
	}
	return lusd.ParseDatei(content)
}

// lusdUploadHandler bündelt, was Vorschau und Import gemeinsam haben: Upload-Grenze,
// Parsen, Fehlerabbildung. apply unterscheidet die beiden Routen.
func (s *Server) lusdUploadHandler(apply bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { s.handleLusdUpload(w, r, apply) }
}

// handleLusdUpload liest die Datei, fährt den Lauf und antwortet mit Vorschau oder Ergebnis.
func (s *Server) handleLusdUpload(w http.ResponseWriter, r *http.Request, apply bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}
	datei, err := readLusdUpload(r)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, err)
		return
	}
	lauf := lusd.Lauf{Anwenden: apply, MassenabgangBestaetigt: apply && r.FormValue("confirm_graduates") == "true"}
	if apply {
		if lauf.Umbenennungen, err = leseUmbenennungsWahl(r.FormValue("umbenennungen")); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}
	}
	// Die Karenzzeit vor der Transaktion des Laufs lesen: eine Einstellung, kein Teil des
	// Bestands, den der Lauf vergleicht.
	lauf.KarenzTage = s.abgaengerKarenzTage(r.Context())
	res, err := lusd.Fuehre(r.Context(), s.DB.Pool, datei, lauf)
	if err != nil {
		apierrors.SendHTTPError(w, lusdFehlerStatus(err), err)
		return
	}
	if apply {
		// Der Import ist der einzige Weg, der Schülernamen unumkehrbar anonymisiert. Ins
		// Protokoll gehen Akteur, Modus und Zähler, keine Namen.
		s.protokolliereLusdImport(r, res, lauf.MassenabgangBestaetigt)
	}
	RespondJSON(w, http.StatusOK, res)
}

// lusdFehlerStatus ordnet den Fehler eines Laufs ein: Der Massenabgang wartet auf eine
// Bestätigung (409), eine überholte Umbenennungs-Auswahl ist eine Fehleingabe (400).
func lusdFehlerStatus(err error) int {
	var massErr *lusd.MassenabgangFehler
	var wahlErr *lusd.UmbenennungUngueltigFehler
	switch {
	case errors.As(err, &massErr):
		return http.StatusConflict
	case errors.As(err, &wahlErr):
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// leseUmbenennungsWahl liest das Formularfeld `umbenennungen` (JSON-Liste aus Zeile +
// schueler_id). Leer heißt: keine Paare bestätigt — dann läuft es wie bisher.
func leseUmbenennungsWahl(roh string) ([]lusd.UmbenennungWahl, error) {
	if strings.TrimSpace(roh) == "" {
		return nil, nil
	}
	var wahl []lusd.UmbenennungWahl
	if err := json.Unmarshal([]byte(roh), &wahl); err != nil {
		return nil, fmt.Errorf("Umbenennungs-Auswahl unlesbar: %w", err) //nolint:staticcheck // ST1005: nutzer-sichtbarer Text
	}
	return wahl, nil
}

// PostLusdPreviewHandler parst die CSV und liefert die Vorschau der Änderungen.
func (s *Server) PostLusdPreviewHandler() http.HandlerFunc { return s.lusdUploadHandler(false) }

// PostLusdImportHandler parst die CSV und wendet die Änderungen transaktional an.
// Ab der Schwelle des Massenabgangs (lusd.MassenabgangFehler) verlangt er das Formularfeld
// confirm_graduates=true (HTTP 409 sonst) — zweite, bewusste Bestätigung.
func (s *Server) PostLusdImportHandler() http.HandlerFunc { return s.lusdUploadHandler(true) }

func zaehleBestaetigte(paare []lusd.UmbenennungDiff) int {
	n := 0
	for _, p := range paare {
		if p.Bestaetigt {
			n++
		}
	}
	return n
}

// protokolliereLusdImport schreibt den Apply-Lauf ins Admin-Audit: Modus, Zähler je
// Kategorie und ob der Massenabgang bestätigt wurde. Ohne Namen.
func (s *Server) protokolliereLusdImport(r *http.Request, res *lusd.PreviewResult, massenabgangBestaetigt bool) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok || s.DB == nil || s.DB.Pool == nil || res == nil {
		return
	}
	details := map[string]any{
		"modus":                    res.Modus,
		"zeilen":                   res.TotalCsvRecords,
		"neu":                      len(res.NewStudents),
		"klassenwechsel":           len(res.ClassChanges),
		"adoptionen":               len(res.Adoptions),
		"rueckkehrer":              len(res.Rueckkehrer),
		"abgaenger":                len(res.Graduates),
		"nicht_im_export":          len(res.NichtImExport),
		"nicht_abgleichbar":        len(res.NichtAbgleichbar),
		"mehrdeutig":               len(res.Mehrdeutig),
		"dubletten_abweichend":     len(res.DublettenAbweichend),
		"umbenennungen_bestaetigt": zaehleBestaetigte(res.Umbenennungen),
		"karenz_tage":              res.KarenzTage,
		"massenabgang_bestaetigt":  massenabgangBestaetigt,
	}
	if err := repository.NewAuditRepository(s.DB.Pool).
		LogAdminAktion(r.Context(), claims.UserID, "LUSD_IMPORT", getIP(r), details); err != nil {
		log.Printf("LUSD-Import: Audit-Eintrag fehlgeschlagen: %v", err)
	}
}
