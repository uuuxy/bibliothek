package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// erzeugeUndCommitBulkMahnung zählt die Mahnung, liest in derselben Transaktion die Briefe
// und schreibt erst fest, wenn das PDF steht. Läge das Lesen vor dem Zählen, könnte ein
// inzwischen zurückgegebenes Buch auf dem Brief stehen, ohne gezählt zu sein. ok=false: Die
// Fehlerantwort ist geschrieben, das Zählen nimmt der Rollback zurück.
func (s *Server) erzeugeUndCommitBulkMahnung(ctx context.Context, w http.ResponseWriter, ausleihIDs []string) ([]byte, bool) {
	// Vorlage und Absender vor der Transaktion: Sie hält eine Verbindung und Zeilensperren.
	vorlage := s.ladeMahnbriefVorlage(ctx)

	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	defer db.SafeRollback(ctx, tx)

	// Nur dieser Druck zählt die Mahnung, der Mail-Versand nicht (docs/invarianten.md §1).
	// Die Auswahl kommt aus der Oberfläche und kann älter sein als eine Verlängerung oder
	// Rückgabe.
	mahnRepo := repository.NewMahnwesenRepository(s.DB.Pool)
	gezaehlt, err := mahnRepo.ZaehleMahnungTx(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim update der mahnstufen: %w", err))
		return nil, false
	}

	// Der Brief hängt nicht am Zählen: Ein Buch steigt höchstens einmal am Tag, gedruckt
	// wird es nach einem Papierstau auch ein zweites Mal.
	briefe, err := mahnRepo.MahnbriefeTx(ctx, tx, ausleihIDs)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("fehler beim abrufen der daten für pdf: %w", err))
		return nil, false
	}
	if len(briefe) == 0 && gezaehlt > 0 {
		// Zählen und Lesen folgen derselben Bedingung, das ist nicht zu erwarten. Keine
		// Mahnung ohne Brief festschreiben.
		apierrors.SendHTTPError(w, http.StatusInternalServerError, fmt.Errorf("dateninkonsistenz: keine PDF-Daten trotz aktualisierter Mahnstufen"))
		return nil, false
	}
	if len(briefe) == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine Ausleihe mit abgelaufener Frist in der Auswahl"))
		return nil, false
	}

	pdfBytes, err := erzeugeMahnbriefe(briefe, vorlage)
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

// BulkPrintRequest benennt die Ausleihen, für die in einem Zug gemahnt werden soll.
// Die Auswahl kommt aus der Oberfläche, nicht aus einer serverseitigen Abfrage — wer
// mahnt, soll vorher sehen, wen es trifft.
type BulkPrintRequest struct {
	AusleihIDs []string `json:"ausleih_ids" validate:"omitempty,dive,required,uuid_oder_leer"`
}

// BulkPrintMahnungenHandler druckt die Mahnbriefe zu den gewählten Ausleihen und zählt die
// Mahnung, beides in einer Transaktion.
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

		pdfBytes, ok := s.erzeugeUndCommitBulkMahnung(r.Context(), w, req.AusleihIDs)
		if !ok {
			return
		}

		filename := fmt.Sprintf("mahnbriefe_%s.pdf", schulzeit.Jetzt().Format(dateFormatISO))

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
