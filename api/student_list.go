package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
)

// ListStudentsHandler returns all students, optionally filtered by klasse.
// @Summary      List students
// @Description  Retrieves students, optionally filtered by a specific school class, along with loan counts.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        klasse  query     string  false  "School class to filter by"
// @Success      200     {array}   map[string]any
// @Failure      500     {object}  map[string]string
// @Router       /schueler [get]
func (s *Server) ListStudentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		klasse := r.URL.Query().Get("klasse")

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var rows pgx.Rows
		var err error
		if klasse != "" {
			rows, err = s.DB.Pool.Query(ctx, `
				SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, strasse, hausnummer, plz, ort, eltern_email,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL) as ausgeliehen_anzahl,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl
				FROM schueler 
				WHERE klasse = $1 
				ORDER BY nachname, vorname
			`, klasse)
		} else {
			rows, err = s.DB.Pool.Query(ctx, `
				SELECT id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, strasse, hausnummer, plz, ort, eltern_email,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL) as ausgeliehen_anzahl,
					(SELECT COUNT(*) FROM ausleihen a WHERE a.schueler_id = schueler.id AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl
				FROM schueler 
				ORDER BY klasse, nachname, vorname 
				LIMIT 500
			`)
		}

		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		students := []map[string]any{}
		for rows.Next() {
			var id, barcode, vorname, nachname, kl string
			var strasse, hausnr, plz, ort, email *string
			var abgaengerJahr int
			var gesperrt bool
			var ausgeliehenAnzahl, ueberfaelligAnzahl int
			if err := rows.Scan(&id, &barcode, &vorname, &nachname, &kl, &abgaengerJahr, &gesperrt, &strasse, &hausnr, &plz, &ort, &email, &ausgeliehenAnzahl, &ueberfaelligAnzahl); err == nil {
				fotoURL := ""
				if barcode != "" {
					filePath := filepath.Join("uploads", "fotos", fmt.Sprintf("%s.jpg", barcode))
					if _, err := os.Stat(filePath); err == nil {
						fotoURL = fmt.Sprintf("/uploads/fotos/%s.jpg", barcode)
					}
				}
				students = append(students, map[string]any{
					"id":                 id,
					"barcode_id":         barcode,
					"vorname":            vorname,
					"nachname":           nachname,
					"klasse":             kl,
					"abgaenger_jahr":     abgaengerJahr,
					"ist_gesperrt":       gesperrt,
					"strasse":            strasse,
					"hausnummer":         hausnr,
					"plz":                plz,
					"ort":                ort,
					"eltern_email":       email,
					"ausgeliehen_count":  ausgeliehenAnzahl,
					"ueberfaellig_count": ueberfaelligAnzahl,
					"foto_url":           fotoURL,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(students)
	}
}
