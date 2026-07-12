package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"
)

// ladeBulkMahnDaten lädt die PDF-Daten zu den Ausleih-IDs (vor der Transaktion).
// ok=false: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) ladeBulkMahnDaten(ctx context.Context, w http.ResponseWriter, ausleihIDs []string) ([]repository.MahnwesenKlasse, bool) {
	mahnRepo := repository.NewMahnwesenRepository(s.DB.Pool)
	klassen, err := mahnRepo.QueryUeberfaelligeByAusleiheIDs(ctx, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim abrufen der daten für pdf: %w", err))
		return nil, false
	}
	if len(klassen) == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("konnte daten für pdf nicht generieren"))
		return nil, false
	}
	return klassen, true
}

// erzeugeUndCommitBulkMahnung inkrementiert in einer Transaktion die Mahnstufen der
// gewählten Ausleihen, erzeugt das PDF und committet. ok=false: die Fehlerantwort wurde
// bereits geschrieben (Rollback greift via defer).
func (s *Server) erzeugeUndCommitBulkMahnung(ctx context.Context, w http.ResponseWriter, ausleihIDs []string, klassen []repository.MahnwesenKlasse) ([]byte, bool) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	// Rollback-Defer, welches wirksam wird, falls tx.Commit() nicht erreicht wird
	defer db.SafeRollback(ctx, tx)

	cmdTag, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET mahnstufe = mahnstufe + 1,
		    letztes_mahndatum = CURRENT_TIMESTAMP
		WHERE id = ANY($1)
	`, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim update der mahnstufen: %w", err))
		return nil, false
	}
	if cmdTag.RowsAffected() == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine der übergebenen Ausleih-IDs zum updaten gefunden"))
		return nil, false
	}

	pdfBytes, err := generateMahnPDF(klassen)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim generieren des pdfs: %w", err))
		return nil, false
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim commit der transaktion: %w", err))
		return nil, false
	}
	return pdfBytes, true
}

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

		// 1. Daten für das PDF abrufen (vor der Transaktion)
		klassen, ok := s.ladeBulkMahnDaten(ctx, w, req.AusleihIDs)
		if !ok {
			return
		}

		// 2.–5. Mahnstufen erhöhen, PDF erzeugen, committen
		pdfBytes, ok := s.erzeugeUndCommitBulkMahnung(ctx, w, req.AusleihIDs, klassen)
		if !ok {
			return
		}

		// 6. PDF an den Client senden
		filename := fmt.Sprintf("Mahnliste_Bulk_%s.pdf", time.Now().Format(dateFormatISO))

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
