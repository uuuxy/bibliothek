package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"
)

type BulkPrintRequest struct {
	AusleihIDs []string `json:"ausleih_ids"`
}

// BulkPrintMahnungenHandler verarbeitet ein Array von Ausleih-IDs,
// inkrementiert deren Mahnstufe, aktualisiert das Mahndatum und generiert das PDF.
// Alles geschieht in einer PostgreSQL-Transaktion mit striktem Rollback bei Fehlern.
// POST /api/admin/mahnungen/bulk-print
func (s *Server) BulkPrintMahnungenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BulkPrintRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if len(req.AusleihIDs) == 0 {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("ausleih_ids array darf nicht leer sein"))
			return
		}

		ctx := r.Context()
		mahnRepo := repository.NewMahnwesenRepository(s.DB.Pool)

		// 1. Daten für das PDF abrufen (vor der Transaktion)
		klassen, err := mahnRepo.QueryUeberfaelligeByAusleiheIDs(ctx, req.AusleihIDs)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim abrufen der daten für pdf: %w", err))
			return
		}

		if len(klassen) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("konnte daten für pdf nicht generieren"))
			return
		}

		// 2. Transaktion starten
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		// Rollback-Defer, welches wirksam wird, falls tx.Commit() nicht erreicht wird
		defer db.SafeRollback(ctx, tx)

		// 3. Bulk Update der Mahnstufe und des Mahndatums für alle ausgewählten IDs
		updateQuery := `
			UPDATE ausleihen
			SET mahnstufe = mahnstufe + 1,
			    letztes_mahndatum = CURRENT_TIMESTAMP
			WHERE id = ANY($1)
		`
		cmdTag, err := tx.Exec(ctx, updateQuery, req.AusleihIDs)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim update der mahnstufen: %w", err))
			return
		}

		if cmdTag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine der übergebenen Ausleih-IDs zum updaten gefunden"))
			return
		}

		// 4. PDF generieren
		pdfBytes, err := generateMahnPDF(klassen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim generieren des pdfs: %w", err))
			return
		}

		// 5. Alles war erfolgreich -> Transaktion committen
		if err := tx.Commit(ctx); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim commit der transaktion: %w", err))
			return
		}

		// 6. PDF an den Client senden
		filename := fmt.Sprintf("Mahnliste_Bulk_%s.pdf", time.Now().Format("2006-01-02"))

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
