package api

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// PrintErsatzEtikettHandler generates an A6 PDF label for a single given exemplar.
// It is used when a physical barcode label is damaged or lost and needs reprinting.
func (s *Server) PrintErsatzEtikettHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing exemplar ID parameter"))
			return
		}

		ctx := r.Context()

		var label BarcodeLabelDetail
		query := `
			SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, '')
			FROM buecher_exemplare e
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE e.id = $1
		`
		err := s.DB.Pool.QueryRow(ctx, query, id).Scan(&label.BarcodeID, &label.Titel, &label.Autor, &label.ISBN)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("exemplar nicht gefunden"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		pdfBytes, err := GenerateSingleLabelPDFA6(label)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		filename := fmt.Sprintf("Ersatz_Etikett_%s.pdf", label.BarcodeID)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		_, _ = w.Write(pdfBytes)
	}
}
