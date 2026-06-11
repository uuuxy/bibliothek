package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// LusdWidgetResult stores the response stats from the CSV import
type LusdWidgetResult struct {
	Updated  int `json:"updated"`
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

// PostSchuelerImportLusdHandler parses a LUSD CSV and upserts students
func (s *Server) PostSchuelerImportLusdHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Parse multipart file (max 10 MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Fehler beim Parsen der Formulardaten", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Keine Datei hochgeladen", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 2. Setup CSV Parser
		reader := csv.NewReader(file)
		reader.Comma = ';' // Default delimiter for LUSD in Germany
		reader.LazyQuotes = true

		header, err := reader.Read()
		if err != nil {
			http.Error(w, "Fehler beim Lesen der Kopfzeile", http.StatusBadRequest)
			return
		}

		// Normalize headers
		headers := make(map[string]int)
		for i, h := range header {
			headers[strings.ToLower(strings.TrimSpace(h))] = i
		}

		// Helper to safely get column index by partial name
		getCol := func(keywords ...string) int {
			for _, kw := range keywords {
				for h, idx := range headers {
					if strings.Contains(h, kw) {
						return idx
					}
				}
			}
			return -1
		}

		colID := getCol("id", "lusd")
		colVorname := getCol("vorname")
		colNachname := getCol("name", "nachname")
		colKlasse := getCol("klasse", "gruppe", "jahrgang")
		colStrasse := getCol("strasse")
		colHausnr := getCol("haus", "nr")
		colPlz := getCol("plz", "post")
		colOrt := getCol("ort")

		if colID == -1 || colVorname == -1 || colNachname == -1 {
			http.Error(w, "Pflichtspalten (ID, Vorname, Nachname) fehlen in CSV", http.StatusBadRequest)
			return
		}

		var result LusdWidgetResult
		tx, err := s.DB.Pool.Begin(ctx)
		if err != nil {
			http.Error(w, "Datenbankfehler", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		// 3. Process Rows and Upsert
		for {
			row, err := reader.Read()
			if err != nil {
				break // EOF or parsing error (ignore broken rows)
			}

			id := strings.TrimSpace(row[colID])
			vorname := strings.TrimSpace(row[colVorname])
			nachname := strings.TrimSpace(row[colNachname])

			if id == "" || vorname == "" || nachname == "" {
				result.Skipped++
				continue
			}

			var klasse, strasse, hausnr, plz, ort *string

			if colKlasse != -1 && len(row) > colKlasse {
				v := strings.TrimSpace(row[colKlasse])
				klasse = &v
			}
			if colStrasse != -1 && len(row) > colStrasse {
				v := strings.TrimSpace(row[colStrasse])
				strasse = &v
			}
			if colHausnr != -1 && len(row) > colHausnr {
				v := strings.TrimSpace(row[colHausnr])
				hausnr = &v
			}
			if colPlz != -1 && len(row) > colPlz {
				v := strings.TrimSpace(row[colPlz])
				plz = &v
			}
			if colOrt != -1 && len(row) > colOrt {
				v := strings.TrimSpace(row[colOrt])
				ort = &v
			}

			barcode := fmt.Sprintf("S-%05d%04d", time.Now().Unix()%100000, time.Now().Nanosecond()%10000)
			abgang := time.Now().Year() + 5

			// Upsert-Logik
			// Wir verwenden zuerst einen direkten ON CONFLICT. Wenn lusd_id keinen UNIQUE Index hat,
			// fangen wir den Fehler ab und machen es in 2 Schritten (SELECT -> UPDATE / INSERT).
			upsertQuery := `
				INSERT INTO schueler (lusd_id, barcode_id, vorname, nachname, klasse, strasse, hausnummer, plz, ort, abgaenger_jahr)
				VALUES ($1, $2, $3, $4, COALESCE($5, 'Unbekannt'), $6, $7, $8, $9, $10)
				ON CONFLICT (lusd_id) DO UPDATE SET
					vorname = EXCLUDED.vorname,
					nachname = EXCLUDED.nachname,
					klasse = EXCLUDED.klasse,
					strasse = EXCLUDED.strasse,
					hausnummer = EXCLUDED.hausnummer,
					plz = EXCLUDED.plz,
					ort = EXCLUDED.ort,
					aktualisiert_am = NOW()
				RETURNING (xmax = 0) AS inserted;
			`

			var wasInserted bool
			err = tx.QueryRow(ctx, upsertQuery, id, barcode, vorname, nachname, klasse, strasse, hausnr, plz, ort, abgang).Scan(&wasInserted)
			
			if err != nil {
				// Fallback: Manuelles Upsert für Systeme ohne UNIQUE Constraint auf lusd_id
				var dbID string
				chkErr := tx.QueryRow(ctx, "SELECT id FROM schueler WHERE lusd_id = $1 LIMIT 1", id).Scan(&dbID)
				
				if chkErr == nil {
					// Datensatz existiert -> UPDATE
					updQuery := `UPDATE schueler SET vorname=$1, nachname=$2, klasse=COALESCE($3, klasse), strasse=$4, hausnummer=$5, plz=$6, ort=$7, aktualisiert_am=NOW() WHERE id=$8`
					_, e2 := tx.Exec(ctx, updQuery, vorname, nachname, klasse, strasse, hausnr, plz, ort, dbID)
					if e2 == nil {
						result.Updated++
						continue
					}
				} else {
					// Datensatz fehlt -> INSERT
					insQuery := `INSERT INTO schueler (lusd_id, barcode_id, vorname, nachname, klasse, strasse, hausnummer, plz, ort, abgaenger_jahr) VALUES ($1, $2, $3, $4, COALESCE($5, 'Unbekannt'), $6, $7, $8, $9, $10)`
					_, e2 := tx.Exec(ctx, insQuery, id, barcode, vorname, nachname, klasse, strasse, hausnr, plz, ort, abgang)
					if e2 == nil {
						result.Inserted++
						continue
					}
				}
				
				result.Skipped++ // Wenn beides fehlschlägt
				continue
			}

			if wasInserted {
				result.Inserted++
			} else {
				result.Updated++
			}
		}

		if err := tx.Commit(ctx); err != nil {
			http.Error(w, "Fehler beim Speichern der Transaktion", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
