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

	// 1. Mahnstufe hochzählen; nur dieser Druck zählt sie, der Mail-Versand nicht
	// (mahnwesen_bulk_mail.go, docs/invarianten.md §1). Die Auswahl kommt aus der
	// Oberfläche und kann älter sein als eine Verlängerung oder Rückgabe.
	mahnRepo := repository.NewMahnwesenRepository(s.DB.Pool)
	gezaehlt, err := mahnRepo.ZaehleMahnungTx(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim update der mahnstufen: %w", err))
		return nil, false
	}

	// 2. Exakt den soeben aktualisierten Zustand fürs PDF lesen — in DERSELBEN Tx. Das
	// Blatt hängt nicht am Zählen: Ein Buch steigt höchstens einmal am Tag, gedruckt wird
	// es nach einem Papierstau auch ein zweites Mal.
	klassen, err := mahnRepo.QueryUeberfaelligeByAusleiheIDsTx(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim abrufen der daten für pdf: %w", err))
		return nil, false
	}
	if len(klassen) == 0 && gezaehlt > 0 {
		// Zählen und Blatt lesen dieselbe Auswahl, das ist nicht zu erwarten. Keine
		// Mahnung ohne PDF festschreiben — der Rollback (defer) nimmt das Zählen zurück.
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("dateninkonsistenz: keine PDF-Daten trotz aktualisierter Mahnstufen"))
		return nil, false
	}
	if len(klassen) == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine Ausleihe mit abgelaufener Frist in der Auswahl"))
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

// BulkPrintRequest benennt die Ausleihen, für die in einem Zug gemahnt werden soll.
// Die Auswahl kommt aus der Oberfläche, nicht aus einer serverseitigen Abfrage — wer
// mahnt, soll vorher sehen, wen es trifft.
type BulkPrintRequest struct {
	AusleihIDs []string `json:"ausleih_ids" validate:"omitempty,dive,uuid_oder_leer"`
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
