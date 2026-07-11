package service

import (
	"bibliothek/db"
	"bibliothek/pkg/closeutil"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ImportDynamic verarbeitet die in rows übergebenen Daten (aus CSV oder XLSX).
// Die Spalten werden über die headerMap dynamisch zugeordnet.
func (s *ImportService) ImportDynamic(ctx context.Context, rows [][]string, headerMap map[string]int) (int, int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer db.SafeRollback(ctx, tx)

	// Preload existing titles for fast mapping
	dbRows, err := tx.Query(ctx, "SELECT id, coalesce(isbn, ''), titel FROM buecher_titel")
	if err != nil {
		return 0, 0, err
	}
	isbnToID := make(map[string]string)
	titelToID := make(map[string]string)
	for dbRows.Next() {
		var id, isbn, titel string
		if err := dbRows.Scan(&id, &isbn, &titel); err == nil {
			if isbn != "" {
				isbnToID[isbn] = id
			}
			titelToID[titel] = id
		}
	}
	if err := dbRows.Err(); err != nil {
		dbRows.Close()
		return 0, 0, err
	}
	dbRows.Close()

	type NewTitle struct {
		Titel     string
		Autor     string
		Verlag    string
		ISBN      string
		Jahr      int
		Kategorie string
	}

	type CopyData struct {
		TitelID       string
		Barcode       string
		IstAusleihbar bool
		ZustandNotiz  string
	}

	newTitlesMap := make(map[string]*NewTitle) // key: isbn or titel
	var newTitlesOrder []string

	// First pass: identify titles that need to be created
	for _, row := range rows[1:] {
		getCol := func(key string) string {
			if idx, ok := headerMap[key]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		titel := getCol("titel")
		barcode := getCol("barcode")
		if titel == "" || barcode == "" {
			continue
		}

		// Lernmittelfreiheit: LMF-Token in der Kategorie ("Buch LMF Ma 6/Gri")
		// → Token entfernen und den Titel per Projekt-Konvention "LMF-" flaggen.
		// Muss VOR dem Titel-Matching passieren, damit beide Pässe und die
		// Bestandsdaten denselben Schlüssel verwenden.
		kategorie := getCol("kategorie")
		if hatLMFKennung(kategorie) {
			kategorie = entferneLMFToken(kategorie)
			titel = flaggeAlsSchulbuch(titel)
		}

		isbn := strings.ReplaceAll(strings.ReplaceAll(getCol("isbn"), "-", ""), " ", "")

		titelID := ""
		if isbn != "" && isbnToID[isbn] != "" {
			titelID = isbnToID[isbn]
		} else if titelToID[titel] != "" {
			titelID = titelToID[titel]
		}

		if titelID == "" {
			// Needs new title
			cacheKey := isbn
			if cacheKey == "" {
				cacheKey = titel
			}
			if _, exists := newTitlesMap[cacheKey]; !exists {
				var jahr int
				if j, err := strconv.Atoi(getCol("jahr")); err == nil {
					jahr = j
				}
				newTitlesMap[cacheKey] = &NewTitle{
					Titel:     titel,
					Autor:     getCol("autor"),
					Verlag:    getCol("verlag"),
					ISBN:      isbn,
					Jahr:      jahr,
					Kategorie: kategorie,
				}
				newTitlesOrder = append(newTitlesOrder, cacheKey)
			}
		}
	}

	var newTitlesCount int
	// Insert new titles using batch
	if len(newTitlesOrder) > 0 {
		batch := &pgx.Batch{}
		qInsertTitel := `
			INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject)
			VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), $6)
			RETURNING id
		`
		for _, key := range newTitlesOrder {
			t := newTitlesMap[key]
			batch.Queue(qInsertTitel, t.Titel, t.Autor, t.Verlag, t.ISBN, t.Jahr, t.Kategorie)
		}

		br := tx.SendBatch(ctx, batch)
		for _, key := range newTitlesOrder {
			var insertedID string
			err := br.QueryRow().Scan(&insertedID)
			if err != nil {
				closeutil.LogClose(br, "title insert batch")
				return 0, 0, fmt.Errorf("failed to insert title batch: %w", err)
			}
			t := newTitlesMap[key]
			if t.ISBN != "" {
				isbnToID[t.ISBN] = insertedID
			}
			titelToID[t.Titel] = insertedID
			newTitlesCount++
		}
		if err := br.Close(); err != nil {
			return 0, 0, fmt.Errorf("failed to close title insert batch: %w", err)
		}
	}

	// Second pass: Now all titles have IDs, collect all copies again
	var copiesToInsert []CopyData

	for i, row := range rows[1:] {
		getCol := func(key string) string {
			if idx, ok := headerMap[key]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}
		titel := getCol("titel")

		// LMF-Flag identisch zu Pass 1 anwenden, sonst verfehlt das
		// Titel-Matching die gerade angelegten "LMF-…"-Titel.
		if hatLMFKennung(getCol("kategorie")) {
			titel = flaggeAlsSchulbuch(titel)
		}

		// 1. String-Bereinigung & 2. Datentyp-Sicherheit
		barcodeRaw := ""
		if idx, ok := headerMap["barcode"]; ok && idx < len(row) {
			barcodeRaw = row[idx]
		}
		barcode := strings.TrimSpace(strings.Trim(barcodeRaw, "\uFEFF\u200B\x00\r\n\t"))

		if titel == "" {
			continue
		}

		// 3. Robustes Logging & Fehlerabfang
		if barcode == "" {
			id := fmt.Sprintf("Zeile %d", i+2)
			log.Printf("Warnung: Exemplar ID %s hat keinen Barcode", id)
			continue
		}

		isbn := strings.ReplaceAll(strings.ReplaceAll(getCol("isbn"), "-", ""), " ", "")

		titelID := ""
		if isbn != "" && isbnToID[isbn] != "" {
			titelID = isbnToID[isbn]
		} else if titelToID[titel] != "" {
			titelID = titelToID[titel]
		}

		// Optionale Zustand-Spalte (nur in der Bestandsdatei vorhanden):
		// "verliehen" sperrt das Exemplar für neue Ausleihen, der Rohwert
		// landet als Zustandsnotiz. Fehlt die Spalte, ist das Exemplar
		// standardmäßig ausleihbar.
		istAusleihbar := true
		zustand := getCol("zustand")
		if strings.EqualFold(zustand, "verliehen") {
			istAusleihbar = false
		}

		if titelID != "" {
			copiesToInsert = append(copiesToInsert, CopyData{
				TitelID:       titelID,
				Barcode:       barcode,
				IstAusleihbar: istAusleihbar,
				ZustandNotiz:  zustand,
			})
		}
	}

	var importedCopiesCount int
	// Insert copies using batch ON CONFLICT DO NOTHING
	if len(copiesToInsert) > 0 {
		batchCopies := &pgx.Batch{}
		qInsertExemplar := `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, ist_ausleihbar, zustand_notiz)
			VALUES ($1, $2, CURRENT_DATE, $3, NULLIF($4, ''))
			ON CONFLICT (barcode_id) DO NOTHING
			RETURNING id
		`
		for _, c := range copiesToInsert {
			batchCopies.Queue(qInsertExemplar, c.TitelID, c.Barcode, c.IstAusleihbar, c.ZustandNotiz)
		}

		bcr := tx.SendBatch(ctx, batchCopies)
		for i := 0; i < len(copiesToInsert); i++ {
			var id string
			err := bcr.QueryRow().Scan(&id)
			if err == nil {
				importedCopiesCount++
			} else if errors.Is(err, pgx.ErrNoRows) {
				// ON CONFLICT DO NOTHING liefert ErrNoRows zurück
				log.Printf("Warnung: Exemplar mit Barcode '%s' (Titel-ID: %s) wurde übersprungen (bereits vorhanden)", copiesToInsert[i].Barcode, copiesToInsert[i].TitelID)
			} else {
				log.Printf("❌ Fehler beim Insert von Barcode '%s' (Titel-ID: %s): %v", copiesToInsert[i].Barcode, copiesToInsert[i].TitelID, err)
			}
		}
		if err := bcr.Close(); err != nil {
			return 0, 0, fmt.Errorf("failed to close copy insert batch: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}

	return newTitlesCount, importedCopiesCount, nil
}
