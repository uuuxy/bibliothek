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

type importNewTitle struct {
	Titel     string
	Autor     string
	Verlag    string
	ISBN      string
	Jahr      int
	Kategorie string
}

type importCopyData struct {
	TitelID       string
	Barcode       string
	IstAusleihbar bool
	ZustandNotiz  string
}

// spaltenWert liest den getrimmten Wert der über headerMap zugeordneten Spalte.
func spaltenWert(row []string, headerMap map[string]int, key string) string {
	if idx, ok := headerMap[key]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

// ladeVorhandeneTitel lädt die bestehenden Titel für schnelles ISBN-/Titel-Matching.
func ladeVorhandeneTitel(ctx context.Context, tx pgx.Tx) (map[string]string, map[string]string, error) {
	dbRows, err := tx.Query(ctx, "SELECT id, coalesce(isbn, ''), titel FROM buecher_titel")
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	dbRows.Close()
	return isbnToID, titelToID, nil
}

// sammleNeueTitel identifiziert (erster Pass) die Titel, die neu angelegt werden
// müssen, weil sie sich weder über ISBN noch über den Titel matchen lassen.
func sammleNeueTitel(rows [][]string, headerMap map[string]int, isbnToID, titelToID map[string]string) (map[string]*importNewTitle, []string) {
	newTitlesMap := make(map[string]*importNewTitle) // key: isbn or titel
	var newTitlesOrder []string

	for _, row := range rows[1:] {
		cacheKey, t, ok := baueNeuTitelAusZeile(row, headerMap, isbnToID, titelToID)
		if !ok {
			continue
		}
		if _, exists := newTitlesMap[cacheKey]; exists {
			continue
		}
		newTitlesMap[cacheKey] = t
		newTitlesOrder = append(newTitlesOrder, cacheKey)
	}
	return newTitlesMap, newTitlesOrder
}

// matchTitelID liefert die bekannte Titel-ID über ISBN (bevorzugt) oder Titel; "" wenn
// noch unbekannt.
func matchTitelID(isbn, titel string, isbnToID, titelToID map[string]string) string {
	if isbn != "" && isbnToID[isbn] != "" {
		return isbnToID[isbn]
	}
	if titelToID[titel] != "" {
		return titelToID[titel]
	}
	return ""
}

// baueNeuTitelAusZeile prüft eine Zeile und liefert (falls es ein noch unbekannter Titel
// ist) den Cache-Key und den anzulegenden Titel. ok=false bedeutet: Zeile überspringen
// (leer oder bereits über ISBN/Titel gematcht).
func baueNeuTitelAusZeile(row []string, headerMap map[string]int, isbnToID, titelToID map[string]string) (cacheKey string, t *importNewTitle, ok bool) {
	titel := spaltenWert(row, headerMap, "titel")
	barcode := spaltenWert(row, headerMap, "barcode")
	if titel == "" || barcode == "" {
		return "", nil, false
	}

	// Lernmittelfreiheit: LMF-Token in der Kategorie ("Buch LMF Ma 6/Gri")
	// → Token entfernen und den Titel per Projekt-Konvention "LMF-" flaggen.
	// Muss VOR dem Titel-Matching passieren, damit beide Pässe und die
	// Bestandsdaten denselben Schlüssel verwenden.
	kategorie := spaltenWert(row, headerMap, "kategorie")
	if hatLMFKennung(kategorie) {
		kategorie = entferneLMFToken(kategorie)
		titel = flaggeAlsSchulbuch(titel)
	}

	isbn := strings.ReplaceAll(strings.ReplaceAll(spaltenWert(row, headerMap, "isbn"), "-", ""), " ", "")

	if matchTitelID(isbn, titel, isbnToID, titelToID) != "" {
		return "", nil, false // schon vorhanden
	}

	// Needs new title
	cacheKey = isbn
	if cacheKey == "" {
		cacheKey = titel
	}

	var jahr int
	if j, err := strconv.Atoi(spaltenWert(row, headerMap, "jahr")); err == nil {
		jahr = j
	}
	return cacheKey, &importNewTitle{
		Titel:     titel,
		Autor:     spaltenWert(row, headerMap, "autor"),
		Verlag:    spaltenWert(row, headerMap, "verlag"),
		ISBN:      isbn,
		Jahr:      jahr,
		Kategorie: kategorie,
	}, true
}

// fuegeNeueTitelEin legt die neuen Titel per Batch an und ergänzt die
// ID-Maps um die neu vergebenen Titel-IDs.
func fuegeNeueTitelEin(ctx context.Context, tx pgx.Tx, newTitlesMap map[string]*importNewTitle, newTitlesOrder []string, isbnToID, titelToID map[string]string) (int, error) {
	if len(newTitlesOrder) == 0 {
		return 0, nil
	}

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
	newTitlesCount := 0
	for _, key := range newTitlesOrder {
		var insertedID string
		if err := br.QueryRow().Scan(&insertedID); err != nil {
			closeutil.LogClose(br, "title insert batch")
			return 0, fmt.Errorf("failed to insert title batch: %w", err)
		}
		t := newTitlesMap[key]
		if t.ISBN != "" {
			isbnToID[t.ISBN] = insertedID
		}
		titelToID[t.Titel] = insertedID
		newTitlesCount++
	}
	if err := br.Close(); err != nil {
		return 0, fmt.Errorf("failed to close title insert batch: %w", err)
	}
	return newTitlesCount, nil
}

// sammleExemplare sammelt (zweiter Pass) alle einzufügenden Exemplare, jetzt mit
// den vollständigen Titel-IDs aus Pass 1.
func sammleExemplare(rows [][]string, headerMap map[string]int, isbnToID, titelToID map[string]string) []importCopyData {
	var copiesToInsert []importCopyData

	for i, row := range rows[1:] {
		titel := spaltenWert(row, headerMap, "titel")

		// LMF-Flag identisch zu Pass 1 anwenden, sonst verfehlt das
		// Titel-Matching die gerade angelegten "LMF-…"-Titel.
		if hatLMFKennung(spaltenWert(row, headerMap, "kategorie")) {
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

		isbn := strings.ReplaceAll(strings.ReplaceAll(spaltenWert(row, headerMap, "isbn"), "-", ""), " ", "")
		titelID := matchTitelID(isbn, titel, isbnToID, titelToID)

		// Optionale Zustand-Spalte (nur in der Bestandsdatei vorhanden):
		// "verliehen" sperrt das Exemplar für neue Ausleihen, der Rohwert
		// landet als Zustandsnotiz. Fehlt die Spalte, ist das Exemplar
		// standardmäßig ausleihbar.
		istAusleihbar := true
		zustand := spaltenWert(row, headerMap, "zustand")
		if strings.EqualFold(zustand, "verliehen") {
			istAusleihbar = false
		}

		if titelID != "" {
			copiesToInsert = append(copiesToInsert, importCopyData{
				TitelID:       titelID,
				Barcode:       barcode,
				IstAusleihbar: istAusleihbar,
				ZustandNotiz:  zustand,
			})
		}
	}
	return copiesToInsert
}

// fuegeExemplareEin schreibt die Exemplare per Batch (ON CONFLICT DO NOTHING).
func fuegeExemplareEin(ctx context.Context, tx pgx.Tx, copiesToInsert []importCopyData) (int, error) {
	if len(copiesToInsert) == 0 {
		return 0, nil
	}

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
	importedCopiesCount := 0
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
		return 0, fmt.Errorf("failed to close copy insert batch: %w", err)
	}
	return importedCopiesCount, nil
}

// ImportDynamic verarbeitet die in rows übergebenen Daten (aus CSV oder XLSX).
// Die Spalten werden über die headerMap dynamisch zugeordnet.
func (s *ImportService) ImportDynamic(ctx context.Context, rows [][]string, headerMap map[string]int) (int, int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer db.SafeRollback(ctx, tx)

	isbnToID, titelToID, err := ladeVorhandeneTitel(ctx, tx)
	if err != nil {
		return 0, 0, err
	}

	newTitlesMap, newTitlesOrder := sammleNeueTitel(rows, headerMap, isbnToID, titelToID)

	newTitlesCount, err := fuegeNeueTitelEin(ctx, tx, newTitlesMap, newTitlesOrder, isbnToID, titelToID)
	if err != nil {
		return 0, 0, err
	}

	copiesToInsert := sammleExemplare(rows, headerMap, isbnToID, titelToID)

	importedCopiesCount, err := fuegeExemplareEin(ctx, tx, copiesToInsert)
	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}

	return newTitlesCount, importedCopiesCount, nil
}
