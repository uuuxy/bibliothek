package api

// graduates.go — Handler for listing graduating students with unreturned books.
// Used by the administration view to generate Laufzettel PDFs for outgoing classes.

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/pdf"
)

// AusleiheDetail holds book-loan info for one physical copy.
type AusleiheDetail struct {
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	CoverURL  string `json:"cover_url"`
	BarcodeID string `json:"barcode_id"`
	Frist     string `json:"frist"`
}

// GraduateDetail extends the basic graduate record with all open loans.
type GraduateDetail struct {
	ID            string           `json:"id"`
	BarcodeID     string           `json:"barcode_id"`
	Vorname       string           `json:"vorname"`
	Nachname      string           `json:"nachname"`
	Klasse        string           `json:"klasse"`
	AbgaengerJahr int              `json:"abgaenger_jahr"`
	IstGesperrt   bool             `json:"ist_gesperrt"`
	Ausleihen     []AusleiheDetail `json:"ausleihen"`
}

// GetGraduatesHandler lists graduating students with unreturned books.
// Pass ?details=true to include per-student loan details (for Laufzettel PDF).
// @Summary      Get list of graduating students
// @Description  Retrieves former/graduating students who still have unreturned books, optionally including loan details.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        details  query     bool  false  "True to include loan detail structures"
// @Success      200      {array}   GraduateDetail
// @Failure      500      {object}  map[string]string
// @Router       /abgaenger [get]
func (s *Server) GetGraduatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.URL.Query().Get("details") != "true" {
			// Basic list: one row per student
			query := `
				SELECT DISTINCT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt
				FROM schueler s
				JOIN ausleihen a ON s.id = a.schueler_id
				WHERE s.deleted_at IS NULL
				  AND s.ist_abgaenger = true
				  AND a.rueckgabe_am IS NULL
				ORDER BY s.klasse, s.nachname
			`
			rows, err := s.DB.Pool.Query(ctx, query)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			defer rows.Close()

			students := []any{}
			for rows.Next() {
				var id, barcode, vorname, nachname, klasse string
				var abgaengerJahr int
				var gesperrt bool
				if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse, &abgaengerJahr, &gesperrt); err != nil {
					apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				students = append(students, map[string]any{
					"id":             id,
					"barcode_id":     barcode,
					"vorname":        vorname,
					"nachname":       nachname,
					"klasse":         klasse,
					"abgaenger_jahr": abgaengerJahr,
					"ist_gesperrt":   gesperrt,
				})
			}
			if err := rows.Err(); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			RespondJSON(w, http.StatusOK, students)
			return
		}

		// Detail mode: one row per loan, assembled into per-student objects
		detailQuery := `
			SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
			       t.titel,
			       coalesce(t.autor, '') AS autor,
			       coalesce(t.cover_url, '') AS cover_url,
			       e.barcode_id AS ex_barcode,
			       coalesce(to_char(a.rueckgabe_frist, 'DD.MM.YYYY'), '') AS frist
			FROM schueler s
			JOIN ausleihen a ON s.id = a.schueler_id
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE s.deleted_at IS NULL
			  AND s.ist_abgaenger = true
			  AND a.rueckgabe_am IS NULL
			ORDER BY s.klasse, s.nachname, t.titel
		`
		rows, err := s.DB.Pool.Query(ctx, detailQuery)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		studMap := map[string]*GraduateDetail{}
		studOrder := make([]string, 0)
		for rows.Next() {
			var id, barcode, vorname, nachname, klasse string
			var abgaengerJahr int
			var gesperrt bool
			var titel, autor, coverURL, exBarcode, frist string
			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse,
				&abgaengerJahr, &gesperrt, &titel, &autor, &coverURL, &exBarcode, &frist); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			if _, ok := studMap[id]; !ok {
				studMap[id] = &GraduateDetail{
					ID:            id,
					BarcodeID:     barcode,
					Vorname:       vorname,
					Nachname:      nachname,
					Klasse:        klasse,
					AbgaengerJahr: abgaengerJahr,
					IstGesperrt:   gesperrt,
					Ausleihen:     []AusleiheDetail{},
				}
				studOrder = append(studOrder, id)
			}
			studMap[id].Ausleihen = append(studMap[id].Ausleihen, AusleiheDetail{
				Titel:     titel,
				Autor:     autor,
				CoverURL:  coverURL,
				BarcodeID: exBarcode,
				Frist:     frist,
			})
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		result := make([]*GraduateDetail, 0, len(studOrder))
		for _, id := range studOrder {
			result = append(result, studMap[id])
		}

		RespondJSON(w, http.StatusOK, result)
	}
}

// GetGraduatesPDFHandler generates the Laufzettel PDF for graduating students.
// @Summary      Get Laufzettel PDF
// @Description  Generates a printable PDF for former/graduating students with their unreturned books.
// @Tags         admin
// @Produce      application/pdf
// @Router       /abgaenger/pdf [get]
func (s *Server) GetGraduatesPDFHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		detailQuery := `
			SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
			       t.titel,
			       coalesce(t.autor, '') AS autor,
			       coalesce(t.cover_url, '') AS cover_url,
			       e.barcode_id AS ex_barcode,
			       coalesce(to_char(a.rueckgabe_frist, 'DD.MM.YYYY'), '') AS frist
			FROM schueler s
			LEFT JOIN ausleihen a ON s.id = a.schueler_id AND a.rueckgabe_am IS NULL
			LEFT JOIN buecher_exemplare e ON a.exemplar_id = e.id
			LEFT JOIN buecher_titel t ON e.titel_id = t.id
			WHERE s.deleted_at IS NULL AND s.klasse IN ('9h', '10r', '13')
			ORDER BY s.klasse, s.nachname, t.titel
		`
		rows, err := s.DB.Pool.Query(ctx, detailQuery)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		studMap := map[string]*pdf.LaufzettelStudent{}
		studOrder := make([]string, 0)
		for rows.Next() {
			var id, barcode, vorname, nachname, klasse string
			var abgaengerJahr int
			var gesperrt bool
			var titel, autor, coverURL, exBarcode, frist *string

			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse,
				&abgaengerJahr, &gesperrt, &titel, &autor, &coverURL, &exBarcode, &frist); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}

			if _, ok := studMap[id]; !ok {
				studMap[id] = &pdf.LaufzettelStudent{
					Vorname:   vorname,
					Nachname:  nachname,
					Klasse:    klasse,
					Ausleihen: []pdf.LaufzettelAusleihe{},
				}
				studOrder = append(studOrder, id)
			}

			if titel != nil {
				studMap[id].Ausleihen = append(studMap[id].Ausleihen, pdf.LaufzettelAusleihe{
					Titel:     *titel,
					BarcodeID: *exBarcode,
					Frist:     *frist,
				})
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		result := make([]pdf.LaufzettelStudent, 0, len(studOrder))
		for _, id := range studOrder {
			result = append(result, *studMap[id])
		}

		if len(result) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("no graduates found"))
			return
		}

		pdfBytes, err := pdf.GenerateLaufzettel(result)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="Laufzettel.pdf"`)
		w.Header().Set("Content-Length", fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, "Laufzettel.pdf", time.Now(), bytes.NewReader(pdfBytes))
	}
}
