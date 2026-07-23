package api

// graduates.go — Handler for listing graduating students with unreturned books.
// Used by the administration view to generate Laufzettel PDFs for outgoing classes.

import (
	"bytes"
	"context"
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

// queryGraduatesBasic liefert eine Zeile je Abgänger mit offenen Ausleihen — inkl. der
// Anzahl offener und davon überfälliger Bücher. Genau diese Zahl ist in der Abgänger-
// Ansicht die handlungsrelevante Information (was muss noch zurück?), nicht die Ausweis-
// nummer. COUNT + GROUP BY ersetzt das frühere DISTINCT (eine Zeile je Schüler bleibt).
func (s *Server) queryGraduatesBasic(ctx context.Context) ([]any, error) {
	query := `
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
		       COUNT(a.id)                                        AS offene_buecher,
		       COUNT(a.id) FILTER (WHERE a.rueckgabe_frist < now()) AS ueberfaellig
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id
		WHERE s.deleted_at IS NULL
		  AND s.ist_abgaenger = true
		  AND a.rueckgabe_am IS NULL
		GROUP BY s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt
		ORDER BY s.klasse, s.nachname
	`
	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []any{}
	for rows.Next() {
		var id, barcode, vorname, nachname, klasse string
		var abgaengerJahr int
		var gesperrt bool
		var offeneBuecher, ueberfaellig int
		if err := rows.Scan(&id, &barcode, &vorname, &nachname, &klasse, &abgaengerJahr, &gesperrt,
			&offeneBuecher, &ueberfaellig); err != nil {
			return nil, err
		}
		students = append(students, map[string]any{
			"id":             id,
			"barcode_id":     barcode,
			"vorname":        vorname,
			"nachname":       nachname,
			"klasse":         klasse,
			"abgaenger_jahr": abgaengerJahr,
			"ist_gesperrt":   gesperrt,
			"offene_buecher": offeneBuecher,
			"ueberfaellig":   ueberfaellig,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return students, nil
}

// queryGraduatesDetail liefert die Abgänger mit ihren offenen Ausleihen, gruppiert je
// Schüler in stabiler Reihenfolge (Klasse, Nachname, Titel) — für den Laufzettel-View.
func (s *Server) queryGraduatesDetail(ctx context.Context) ([]*GraduateDetail, error) {
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
		return nil, err
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
			return nil, err
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
		return nil, err
	}

	result := make([]*GraduateDetail, 0, len(studOrder))
	for _, id := range studOrder {
		result = append(result, studMap[id])
	}
	return result, nil
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
			students, err := s.queryGraduatesBasic(ctx)
			if err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			RespondJSON(w, http.StatusOK, students)
			return
		}

		// Detail mode: one row per loan, assembled into per-student objects
		result, err := s.queryGraduatesDetail(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}

// queryLaufzettelStudents lädt die Abgänger MIT noch offenen Ausleihen für die
// Laufzettel-PDF. Ein Laufzettel listet die zurückzugebenden Bücher — ein Abgänger ohne
// offene Bücher braucht keinen. Deshalb INNER JOIN auf ausleihen (früher LEFT JOIN):
// sonst erschien jeder Abgänger, und beim Massendruck von 150 Abgängern kamen 140 komplett
// leere Laufzettel aus dem Drucker (massive Papierverschwendung).
//
// Filter ist ist_abgaenger — dieselbe Definition wie die übrige Abgänger-Ansicht.
// Früher stand hier eine hartkodierte, case-sensitive Klassenliste ('9h','10r','13').
// Die versetzte/importierte Klassen wie '09h', '9H' oder '10a' schlicht ignoriert:
// Betroffene Abgänger fehlten auf dem PDF und verließen die Schule mit ihren Büchern.
func (s *Server) queryLaufzettelStudents(ctx context.Context) ([]pdf.LaufzettelStudent, error) {
	detailQuery := `
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
		       t.titel,
		       coalesce(t.autor, '') AS autor,
		       coalesce(t.cover_url, '') AS cover_url,
		       e.barcode_id AS ex_barcode,
		       coalesce(to_char(a.rueckgabe_frist, 'DD.MM.YYYY'), '') AS frist
		FROM schueler s
		JOIN ausleihen a ON s.id = a.schueler_id AND a.rueckgabe_am IS NULL
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE s.deleted_at IS NULL AND s.ist_abgaenger = true
		ORDER BY s.klasse, s.nachname, t.titel
	`
	rows, err := s.DB.Pool.Query(ctx, detailQuery)
	if err != nil {
		return nil, err
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
			return nil, err
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
		return nil, err
	}

	result := make([]pdf.LaufzettelStudent, 0, len(studOrder))
	for _, id := range studOrder {
		result = append(result, *studMap[id])
	}
	return result, nil
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

		result, err := s.queryLaufzettelStudents(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
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

		w.Header().Set(headerContentType, contentTypePDF)
		w.Header().Set(headerContentDisposition, `attachment; filename="Laufzettel.pdf"`)
		w.Header().Set(headerContentLength, fmt.Sprint(len(pdfBytes)))
		http.ServeContent(w, r, "Laufzettel.pdf", time.Now(), bytes.NewReader(pdfBytes))
	}
}
