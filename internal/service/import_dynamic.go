package service

import (
	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/pkg/closeutil"
	"bibliothek/pkg/lmf"
	"bibliothek/repository"
	"context"
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
	// Lernmittel-Feld und seine Ableitungen (Migration 093), siehe titelZeilenFelder.
	IstLernmittel bool
	Fach          string
	JahrgangVon   int
	JahrgangBis   int
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

// zeilenFelder sind die normalisierten Titel-Felder einer Import-Zeile.
type zeilenFelder struct {
	titel, signatur, kategorie string
	// lernmittel: LMF-Token in Kategorie („Buch LMF Ma 6/Gri") oder Signatur („LMF Bio
	// 7"). Titel und Signatur bleiben unverändert (Migration 093); nur aus der Kategorie
	// wird das Token entfernt, weil sie als Fach-Rückfall in subject landet.
	lernmittel bool
	// fach/jahrgang aus dem LMF-Teil („LMF Ma 6" → Mathematik, 6), sonst leer/0.
	fach     string
	von, bis int
}

// titelZeilenFelder liefert die normalisierten Titel-Felder einer Zeile. BEIDE
// Import-Pässe (Titel sammeln, Exemplare sammeln) müssen diese Funktion verwenden,
// sonst verfehlt das Titel-Matching die gerade angelegten Titel.
func titelZeilenFelder(row []string, headerMap map[string]int) zeilenFelder {
	z := zeilenFelder{
		titel:     bereinigeImportTitel(spaltenWert(row, headerMap, "titel")),
		signatur:  spaltenWert(row, headerMap, "signatur"),
		kategorie: spaltenWert(row, headerMap, "kategorie"),
	}
	if !hatLMFKennung(z.kategorie) && !hatLMFKennung(z.signatur) {
		return z
	}
	z.lernmittel = true
	teil, ok := lmf.Zerlege(z.signatur)
	if !ok {
		teil, _ = zerlegeLMFTeil(z.kategorie)
	}
	z.fach, z.von, z.bis = teil.Fach, teil.JahrgangVon, teil.JahrgangBis
	z.kategorie = entferneLMFToken(z.kategorie)
	return z
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
	z := titelZeilenFelder(row, headerMap)
	barcode := spaltenWert(row, headerMap, "barcode")
	if z.titel == "" || barcode == "" {
		return "", nil, false
	}

	isbn := cleanISBN(spaltenWert(row, headerMap, "isbn"))

	if matchTitelID(isbn, z.titel, isbnToID, titelToID) != "" {
		return "", nil, false // schon vorhanden
	}

	// Needs new title
	cacheKey = isbn
	if cacheKey == "" {
		cacheKey = repository.NormalisiereTitelKey(z.titel)
	}

	var jahr int
	if j, err := strconv.Atoi(spaltenWert(row, headerMap, "jahr")); err == nil {
		jahr = j
	}
	return cacheKey, &importNewTitle{
		Titel:         z.titel,
		Autor:         spaltenWert(row, headerMap, "autor"),
		Verlag:        spaltenWert(row, headerMap, "verlag"),
		ISBN:          isbn,
		Jahr:          jahr,
		Kategorie:     z.kategorie,
		Signatur:      z.signatur,
		IstLernmittel: z.lernmittel,
		Fach:          z.fach,
		JahrgangVon:   z.von,
		JahrgangBis:   z.bis,
	}, true
}

// fachDerZeile: das aus der Lernmittelsignatur gelesene Fach, sonst die Kategorie.
func fachDerZeile(t *importNewTitle) string {
	if t.Fach != "" {
		return t.Fach
	}
	return t.Kategorie
}

// fuegeNeueTitelEin legt die neuen Titel per Batch an und ergänzt die
// ID-Maps um die neu vergebenen Titel-IDs.
func fuegeNeueTitelEin(ctx context.Context, tx pgx.Tx, newTitlesMap map[string]*importNewTitle, newTitlesOrder []string, isbnToID, titelToID map[string]string) (int, error) {
	if len(newTitlesOrder) == 0 {
		return 0, nil
	}

	// subject ist FK auf die Systematik (Migration 078): unbekannte Fächer VOR dem
	// SendBatch in derselben Transaktion registrieren (danach ist die Verbindung bis
	// br.Close() belegt) und jede Zeile auf die kanonische Schreibweise ziehen.
	// Fach: aus der Lernmittelsignatur („LMF Ma 6" → Mathematik), sonst die
	// Kategorie-Spalte wie bisher.
	faecher := make([]string, 0, len(newTitlesOrder))
	for _, key := range newTitlesOrder {
		faecher = append(faecher, fachDerZeile(newTitlesMap[key]))
	}
	kanonisch, err := inventur.StelleFaecherSicher(ctx, tx, faecher)
	if err != nil {
		return 0, err
	}

	batch := &pgx.Batch{}
	qInsertTitel := `
		INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject, signatur,
		                           ist_lernmittel, grade_level, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), NULLIF($6, ''), NULLIF($7, ''),
		        $8, NULLIF($9, 0)::smallint, COALESCE(NULLIF($10, 0), 5), COALESCE(NULLIF($11, 0), 10))
		RETURNING id
	`
	for _, key := range newTitlesOrder {
		t := newTitlesMap[key]
		stufe := 0
		if t.JahrgangVon > 0 && t.JahrgangVon == t.JahrgangBis {
			stufe = t.JahrgangVon
		}
		batch.Queue(qInsertTitel, t.Titel, t.Autor, t.Verlag, t.ISBN, t.Jahr, kanonisch[fachDerZeile(t)], t.Signatur,
			t.IstLernmittel, stufe, t.JahrgangVon, t.JahrgangBis)
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
		titel := titelZeilenFelder(row, headerMap).titel

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
		z := titelZeilenFelder(row, headerMap)
		if z.titel == "" || z.signatur == "" {
			continue
		}
		isbn := cleanISBN(spaltenWert(row, headerMap, "isbn"))
		if id := matchTitelID(isbn, z.titel, isbnToID, titelToID); id != "" {
			updates[id] = z.signatur
		}
	}
	return updates
}

// schreibeSignaturUpdates setzt die gesammelten Signaturen per Batch. Nur
// nicht-leere Werte sind im Map enthalten — die Konvention „das Rücken-Etikett
// gewinnt, leer überschreibt nie" bleibt damit gewahrt. Trägt die Signatur Litteras
// LMF-Kennung, wird der Bestandstitel zugleich als Lernmittel markiert (nur gesetzt,
// nie gelöscht — Migration 093).
func schreibeSignaturUpdates(ctx context.Context, tx pgx.Tx, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for id, signatur := range updates {
		batch.Queue(
			"UPDATE buecher_titel SET signatur = $2, ist_lernmittel = ist_lernmittel OR $3, aktualisiert_am = CURRENT_TIMESTAMP WHERE id = $1",
			id, signatur, lmf.HatKennung(signatur),
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
	// etikett_gedruckt = true: Der Sammelimport uebernimmt BESTAND — Buecher, die
	// physisch laengst ein Littera-Etikett tragen. Mit dem Default false zaehlte
	// das Druck-Center nach dem Prod-Import 30.658 "offene Etiketten" und stand
	// dauerhaft auf 999+ (ein Waechter, der immer schreit, wird abgeschaltet).
	// Neuzugaenge aus dem Bestellwesen behalten false — dort entsteht das Etikett
	// wirklich erst im Haus.
	qInsertExemplar := `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, ist_ausleihbar, zustand_notiz, etikett_gedruckt)
		VALUES ($1, $2, CURRENT_DATE, $3, NULLIF($4, ''), true)
		ON CONFLICT (barcode_id) DO NOTHING
	`
	for _, c := range copiesToInsert {
		batchCopies.Queue(qInsertExemplar, c.TitelID, c.Barcode, c.IstAusleihbar, c.ZustandNotiz)
	}

	bcr := tx.SendBatch(ctx, batchCopies)
	importedCopiesCount := 0
	skippedCount := 0
	for i := 0; i < len(copiesToInsert); i++ {
		ct, err := bcr.Exec()
		if err == nil {
			if ct.RowsAffected() == 1 {
				importedCopiesCount++
			} else {
				// ON CONFLICT DO NOTHING (0 rows affected)
				skippedCount++
			}
		} else {
			log.Printf("❌ Fehler beim Insert von Barcode '%s' (Titel-ID: %s): %v", copiesToInsert[i].Barcode, copiesToInsert[i].TitelID, err)
		}
	}

	if skippedCount > 0 {
		log.Printf("Warnung: %d Exemplare wurden übersprungen (bereits vorhanden)", skippedCount)
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
