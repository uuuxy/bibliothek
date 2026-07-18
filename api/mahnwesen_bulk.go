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

// erzeugeUndCommitBulkMahnung führt den gesamten Bulk-Mahnlauf in EINER Transaktion aus:
// Mahnstufe hochzählen (das UPDATE sperrt die betroffenen Zeilen für die Tx-Dauer), dann
// exakt diesen festgeschriebenen Zustand fürs PDF auslesen, dann committen. Das Auslesen
// passiert bewusst INNERHALB der Transaktion — läge es davor, könnte ein zwischen
// Aufbereiten und Druck zurückgegebenes Buch aufs Papier geraten, ohne dass seine
// Mahnstufe steigt (TOCTOU). ok=false: die Fehlerantwort wurde bereits geschrieben
// (Rollback greift via defer).
func (s *Server) erzeugeUndCommitBulkMahnung(ctx context.Context, w http.ResponseWriter, ausleihIDs []string) ([]byte, bool) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	// Rollback-Defer, welches wirksam wird, falls tx.Commit() nicht erreicht wird
	defer db.SafeRollback(ctx, tx)

	// 1. Mahnstufe hochzählen. Dies ist der EINZIGE Ort, der mahnstufe erhöht — der
	// PDF-Druck ist der physische Verwaltungsakt. Der Mail-Versand bumpt bewusst NICHT
	// (Friendly Reminder, siehe mahnwesen_bulk_mail.go / docs/invarianten.md §1).
	// rueckgabe_am IS NULL: bereits zurückgegebene Bücher werden nicht gemahnt. Das UPDATE
	// nimmt zugleich einen Write-Lock auf die getroffenen Zeilen — eine parallele Rückgabe
	// derselben Ausleihe blockiert bis zu unserem Commit.
	cmdTag, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET mahnstufe = mahnstufe + 1,
		    letztes_mahndatum = CURRENT_TIMESTAMP
		WHERE id = ANY($1) AND rueckgabe_am IS NULL
	`, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim update der mahnstufen: %w", err))
		return nil, false
	}
	if cmdTag.RowsAffected() == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine offene Ausleihe zu den übergebenen IDs gefunden"))
		return nil, false
	}

	// 2. Exakt den soeben aktualisierten Zustand fürs PDF lesen — in DERSELBEN Tx.
	klassen, err := repository.NewMahnwesenRepository(s.DB.Pool).QueryUeberfaelligeByAusleiheIDsTx(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim abrufen der daten für pdf: %w", err))
		return nil, false
	}
	if len(klassen) == 0 {
		// Nach RowsAffected>0 nicht zu erwarten; defensiv: keine Mahnung ohne PDF
		// festschreiben — der Rollback (defer) nimmt den Mahnstufen-Bump zurück.
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("dateninkonsistenz: keine PDF-Daten trotz aktualisierter Mahnstufen"))
		return nil, false
	}

	// 3. PDF erzeugen …
	pdfBytes, err := generateMahnPDF(klassen)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim generieren des pdfs: %w", err))
		return nil, false
	}

	// 4. … und erst nach erfolgreichem PDF committen.
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

		// Mahnstufen erhöhen, aktualisierten Zustand auslesen, PDF erzeugen, committen —
		// alles in einer Transaktion (siehe erzeugeUndCommitBulkMahnung).
		pdfBytes, ok := s.erzeugeUndCommitBulkMahnung(ctx, w, req.AusleihIDs)
		if !ok {
			return
		}

		// PDF an den Client senden
		filename := fmt.Sprintf("Mahnliste_Bulk_%s.pdf", time.Now().Format(dateFormatISO))

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
