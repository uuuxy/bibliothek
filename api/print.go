package api

import (
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pdf"
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// PrintRechnungHandler generates the invoice for lost books of a student.
func PrintRechnungHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// extract schueler_id from the URL path: /api/print/rechnung/{id}
		idStr := r.PathValue("schueler_id")
		schuelerID, err := uuid.Parse(idStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		var s pdf.Schueler
		err = dbPool.QueryRow(ctx, `
			SELECT vorname, nachname
			FROM schueler WHERE id = $1
		`, schuelerID).Scan(&s.Vorname, &s.Nachname)
		s.Strasse = ""
		s.Hausnummer = ""
		s.PLZ = ""
		s.Ort = ""
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return
		}

		query := `
			SELECT t.titel, e.barcode_id, a.ausgeliehen_am, sf.betrag
			FROM schadensfaelle sf
			JOIN buecher_exemplare e ON sf.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			JOIN ausleihen a ON sf.ausleihe_id = a.id
			WHERE sf.schueler_id = $1 AND sf.ist_bezahlt = false
		`
		rows, err := dbPool.Query(ctx, query, schuelerID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		var items []pdf.RechnungItem
		for rows.Next() {
			var item pdf.RechnungItem
			if err := rows.Scan(&item.Titel, &item.Barcode, &item.Ausleihdatum, &item.Ersatzpreis); err != nil {
				continue
			}
			items = append(items, item)
		}

		if len(items) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("no open damage records found for student"))
			return
		}

		pdfBytes, err := pdf.GenerateRechnung(s, items)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `inline; filename="Rechnung.pdf"`)
		w.Header().Set("Content-Length", fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, "Rechnung.pdf", time.Now(), bytes.NewReader(pdfBytes))
	}
}

// PrintMahnungHandler generates the overdue notice PDF for all students in a class.
func PrintMahnungHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		klasse := r.PathValue("klasse")
		if klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("klasse is required"))
			return
		}

		query := `
			SELECT s.id, s.vorname, s.nachname, s.klasse, t.titel, e.barcode_id, a.ausgeliehen_am
			FROM schueler s
			JOIN ausleihen a ON s.id = a.schueler_id
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE LOWER(s.klasse) = LOWER($1)
			  AND a.rueckgabe_am IS NULL
			  AND a.rueckgabe_frist < CURRENT_DATE
			ORDER BY s.nachname, s.vorname, a.ausgeliehen_am
		`
		rows, err := dbPool.Query(ctx, query, klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		schuelerMap := make(map[string]*pdf.MahnungSchueler)
		var schuelerOrder []string

		for rows.Next() {
			var sID uuid.UUID
			var vorname, nachname, klasseStr, titel, barcode string
			var ausleihdatum time.Time

			if err := rows.Scan(&sID, &vorname, &nachname, &klasseStr, &titel, &barcode, &ausleihdatum); err != nil {
				continue
			}

			key := sID.String()
			if _, exists := schuelerMap[key]; !exists {
				schuelerMap[key] = &pdf.MahnungSchueler{
					Vorname:  vorname,
					Nachname: nachname,
					Klasse:   klasseStr,
					Buecher:  []pdf.MahnungBuch{},
				}
				schuelerOrder = append(schuelerOrder, key)
			}

			schuelerMap[key].Buecher = append(schuelerMap[key].Buecher, pdf.MahnungBuch{
				Titel:       titel,
				Barcode:     barcode,
				FaelligSeit: ausleihdatum, // The prompt requested "FaelligSeit" but we can use ausleihdatum or calculate frist. We use ausleihdatum as requested ("inkl. Barcode und Rückgabedatum/Ausleihdatum").
			})
		}

		if len(schuelerOrder) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine überfälligen Ausleihen für Klasse %s gefunden", klasse))
			return
		}

		var schuelerListe []pdf.MahnungSchueler
		for _, key := range schuelerOrder {
			schuelerListe = append(schuelerListe, *schuelerMap[key])
		}

		pdfBytes, err := pdf.GenerateMahnliste(schuelerListe)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		filename := fmt.Sprintf("Mahnliste_Klasse_%s.pdf", klasse)

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}

// PrintKontoauszugHandler generates a simple receipt of all currently borrowed books for a student.
func PrintKontoauszugHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		idStr := r.PathValue("schueler_id")
		schuelerID, err := uuid.Parse(idStr)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		var s pdf.KontoauszugSchueler
		err = dbPool.QueryRow(ctx, `
			SELECT vorname, nachname, klasse
			FROM schueler WHERE id = $1
		`, schuelerID).Scan(&s.Vorname, &s.Nachname, &s.Klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, err)
			return
		}

		query := `
			SELECT t.titel, e.barcode_id, a.ausgeliehen_am, a.rueckgabe_frist
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			JOIN buecher_titel t ON e.titel_id = t.id
			WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
			ORDER BY a.rueckgabe_frist ASC
		`
		rows, err := dbPool.Query(ctx, query, schuelerID)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		var buecher []pdf.KontoauszugBuch
		for rows.Next() {
			var b pdf.KontoauszugBuch
			if err := rows.Scan(&b.Titel, &b.Barcode, &b.Ausleihdatum, &b.Rueckgabedatum); err != nil {
				continue
			}
			buecher = append(buecher, b)
		}

		if len(buecher) == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine aktiven Ausleihen gefunden"))
			return
		}

		pdfBytes, err := pdf.GenerateKontoauszug(s, buecher)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		filename := fmt.Sprintf("Kontoauszug_%s_%s.pdf", s.Vorname, s.Nachname)

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprint(len(pdfBytes)))

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(pdfBytes))
	}
}
