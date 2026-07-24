package service

import (
	"bibliothek/db"
	"bibliothek/pkg/closeutil"
	"bibliothek/repository"
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
	Signatur  string
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

// bereinigeImportTitel entfernt Streu-Anführungszeichen an den Rändern, wie sie
// die aus dem Littera-PDF konvertierte Bestands-CSV enthält (`"Elemente Chemie 1`).
// Ohne diese Bereinigung matcht so eine Zeile nie den Katalogisat-Titel — der
// Import legt dann eine Dublette an.
func bereinigeImportTitel(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), `"`))
}

// titelZeilenFelder liefert die normalisierten Titel-Felder einer Zeile: Titel
// (bereinigt, ggf. LMF-geflaggt), Signatur und Kategorie (jeweils ohne LMF-Token).
// BEIDE Import-Pässe (Titel sammeln, Exemplare sammeln) müssen diese Funktion
// verwenden, sonst verfehlt das Titel-Matching die gerade angelegten Titel.
func titelZeilenFelder(row []string, headerMap map[string]int) (titel, signatur, kategorie string) {
	titel = bereinigeImportTitel(spaltenWert(row, headerMap, "titel"))
	signatur = spaltenWert(row, headerMap, "signatur")
	kategorie = spaltenWert(row, headerMap, "kategorie")

	// Lernmittelfreiheit: LMF-Token in Kategorie ("Buch LMF Ma 6/Gri") oder
	// Signatur ("LMF Bio 7") → Token entfernen und den Titel per
	// Projekt-Konvention "LMF-" flaggen (identisch zum XML-Pfad).
	if hatLMFKennung(kategorie) || hatLMFKennung(signatur) {
		kategorie = entferneLMFToken(kategorie)
		signatur = entferneLMFToken(signatur)
		titel = flaggeAlsSchulbuch(titel)
	}
	return titel, signatur, kategorie
}

// ladeVorhandeneTitel lädt die bestehenden Titel für schnelles ISBN-/Titel-Matching.
// Die Titel-Map ist über repository.NormalisiereTitelKey geschlüsselt — identisch
// zum XML-Pfad, damit Anführungszeichen-Varianten desselben Titels matchen.
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
			titelToID[repository.NormalisiereTitelKey(titel)] = id
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
// noch unbekannt. Der Titel-Lookup läuft über den normalisierten Schlüssel.
// cleanISBN entfernt effizient Bindestriche und Leerzeichen aus einer ISBN
func cleanISBN(val string) string {
	var count int
	for i := 0; i < len(val); i++ {
		if val[i] == '-' || val[i] == ' ' {
			count++
		}
	}
	if count == 0 {
		return val
	}
	b := make([]byte, len(val)-count)
	var j int
	for i := 0; i < len(val); i++ {
		if val[i] != '-' && val[i] != ' ' {
			b[j] = val[i]
			j++
		}
	}
	return string(b)
}

func matchTitelID(isbn, titel string, isbnToID, titelToID map[string]string) string {
	if isbn != "" && isbnToID[isbn] != "" {
		return isbnToID[isbn]
	}
	return titelToID[repository.NormalisiereTitelKey(titel)]
}

// baueNeuTitelAusZeile prüft eine Zeile und liefert (falls es ein noch unbekannter Titel
// ist) den Cache-Key und den anzulegenden Titel. ok=false bedeutet: Zeile überspringen
// (leer oder bereits über ISBN/Titel gematcht).
func baueNeuTitelAusZeile(row []string, headerMap map[string]int, isbnToID, titelToID map[string]string) (cacheKey string, t *importNewTitle, ok bool) {
	titel, signatur, kategorie := titelZeilenFelder(row, headerMap)
	barcode := spaltenWert(row, headerMap, "barcode")
	if titel == "" || barcode == "" {
		return "", nil, false
	}

	isbn := cleanISBN(spaltenWert(row, headerMap, "isbn"))

	if matchTitelID(isbn, titel, isbnToID, titelToID) != "" {
		return "", nil, false // schon vorhanden
	}

	// Needs new title
	cacheKey = isbn
	if cacheKey == "" {
		cacheKey = repository.NormalisiereTitelKey(titel)
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
		Signatur:  signatur,
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
		INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject, signatur)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), $6, NULLIF($7, ''))
		RETURNING id
	`
	for _, key := range newTitlesOrder {
		t := newTitlesMap[key]
		batch.Queue(qInsertTitel, t.Titel, t.Autor, t.Verlag, t.ISBN, t.Jahr, t.Kategorie, t.Signatur)
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
		titelToID[repository.NormalisiereTitelKey(t.Titel)] = insertedID
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
		// Identische Titel-Normalisierung wie Pass 1, sonst verfehlt das
		// Titel-Matching die gerade angelegten Titel.
		titel, _, _ := titelZeilenFelder(row, headerMap)

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

		isbn := cleanISBN(spaltenWert(row, headerMap, "isbn"))
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

// sammleSignaturUpdates sammelt je Titel-ID die Signatur aus der Datei (letzte
// nicht-leere gewinnt). Damit bekommen auch BESTEHENDE Titel ihre Signatur —
// der Insert-Pfad deckt nur neue Titel ab.
func sammleSignaturUpdates(rows [][]string, headerMap map[string]int, isbnToID, titelToID map[string]string) map[string]string {
	if _, ok := headerMap["signatur"]; !ok {
		return nil
	}

	updates := make(map[string]string)
	for _, row := range rows[1:] {
		titel, signatur, _ := titelZeilenFelder(row, headerMap)
		if titel == "" || signatur == "" {
			continue
		}
		isbn := cleanISBN(spaltenWert(row, headerMap, "isbn"))
		if id := matchTitelID(isbn, titel, isbnToID, titelToID); id != "" {
			updates[id] = signatur
		}
	}
	return updates
}

// schreibeSignaturUpdates setzt die gesammelten Signaturen per Batch. Nur
// nicht-leere Werte sind im Map enthalten — die Konvention „das Rücken-Etikett
// gewinnt, leer überschreibt nie" bleibt damit gewahrt.
func schreibeSignaturUpdates(ctx context.Context, tx pgx.Tx, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for id, signatur := range updates {
		batch.Queue(
			"UPDATE buecher_titel SET signatur = $2, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $1",
			id, signatur,
		)
	}
	br := tx.SendBatch(ctx, batch)
	for i := 0; i < len(updates); i++ {
		if _, err := br.Exec(); err != nil {
			closeutil.LogClose(br, "signatur update batch")
			return fmt.Errorf("failed to update signatur batch: %w", err)
		}
	}
	return br.Close()
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

	if err := schreibeSignaturUpdates(ctx, tx, sammleSignaturUpdates(rows, headerMap, isbnToID, titelToID)); err != nil {
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
