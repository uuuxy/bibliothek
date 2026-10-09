package service

import (
	"bibliothek/db"
	"bibliothek/pkg/isbnutil"
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

// titelLookup ordnet eine Zeile der Datei einem Titel zu: der Bestand aus
// repository.LadeTitelBestand, ergänzt um die Titel, die der Lauf selbst anlegt.
type titelLookup struct {
	isbnToID  map[string]string
	titelToID map[string]string
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

// sammleNeueTitel identifiziert (erster Pass) die Titel, die neu angelegt werden
// müssen, weil sie sich weder über ISBN noch über den Titel matchen lassen.
func sammleNeueTitel(rows [][]string, headerMap map[string]int, lookup titelLookup) (map[string]*importNewTitle, []string) {
	newTitlesMap := make(map[string]*importNewTitle) // key: isbn or titel
	var newTitlesOrder []string

	for _, row := range rows[1:] {
		cacheKey, t, ok := baueNeuTitelAusZeile(row, headerMap, lookup)
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
// matchTitelID liefert die bekannte Titel-ID über ISBN (bevorzugt) oder Titel; "" wenn
// noch unbekannt. Beide Seiten in der Normalform (isbnutil.Normalform, Migration 133) —
// die Datenbank speichert sie so, der Bestand kann noch anders geschrieben sein.
func matchTitelID(isbn, titel string, lookup titelLookup) string {
	if n := isbnutil.Normalform(isbn); n != "" && lookup.isbnToID[n] != "" {
		return lookup.isbnToID[n]
	}
	return lookup.titelToID[repository.NormalisiereTitelKey(titel)]
}

// baueNeuTitelAusZeile prüft eine Zeile und liefert (falls es ein noch unbekannter Titel
// ist) den Cache-Key und den anzulegenden Titel. ok=false bedeutet: Zeile überspringen
// (leer oder bereits über ISBN/Titel gematcht).
func baueNeuTitelAusZeile(row []string, headerMap map[string]int, lookup titelLookup) (cacheKey string, t *importNewTitle, ok bool) {
	z := titelZeilenFelder(row, headerMap)
	barcode := spaltenWert(row, headerMap, "barcode")
	if z.titel == "" || barcode == "" {
		return "", nil, false
	}

	// Die Form, in der die Datenbank speichert: Nennt die Datei dasselbe Buch zehn- und
	// dreizehnstellig, ist es ein Titel. Als zwei Schlüssel ergäbe es zwei INSERTs, und der
	// zweite scheiterte am UNIQUE-Index.
	isbn := isbnutil.Normalform(isbnutil.CleanISBN(spaltenWert(row, headerMap, "isbn")))

	if matchTitelID(isbn, z.titel, lookup) != "" {
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

// fachDerZeile: das aus der Lernmittelsignatur gelesene Fach, sonst die Kategorie —
// aber nur, wenn die als Ganzes ein Fach benennt. Bis 03.09.2026 landete die Spalte
// ungeprüft in subject; die aus dem Littera-PDF gewonnene Bestands-CSV trägt dort
// Standorttexte („Buch Pg/Kaf 078829 1. Aufl."), und jeder davon wurde ein eigenes
// „Fach" in der Systematik (1.677 Zeilen auf dem Test-Server, scripts/
// repair_fach_kategorie.sql räumt sie ab).
func fachDerZeile(t *importNewTitle) string {
	if t.Fach != "" {
		return t.Fach
	}
	return lmf.FachExakt(t.Kategorie)
}

// alsTitel füllt den Titel, wie die Datenbankschicht ihn anlegt. Das Fach kommt aus der
// Lernmittelsignatur („LMF Ma 6" → Mathematik), sonst aus der Kategorie (fachDerZeile).
func (t *importNewTitle) alsTitel() repository.BookTitle {
	return repository.BookTitle{
		Titel:            t.Titel,
		Autor:            t.Autor,
		Verlag:           t.Verlag,
		ISBN:             t.ISBN,
		Erscheinungsjahr: t.Jahr,
		Fach:             fachDerZeile(t),
		Signatur:         t.Signatur,
		IstLernmittel:    t.IstLernmittel,
		JahrgangVon:      t.JahrgangVon,
		JahrgangBis:      t.JahrgangBis,
	}
}

// fuegeNeueTitelEin legt die neuen Titel an und ergänzt die ID-Maps um die neu vergebenen
// Titel-IDs.
func fuegeNeueTitelEin(ctx context.Context, tx pgx.Tx, newTitlesMap map[string]*importNewTitle, newTitlesOrder []string, lookup titelLookup) (int, error) {
	neu := make([]repository.BookTitle, 0, len(newTitlesOrder))
	for _, key := range newTitlesOrder {
		neu = append(neu, newTitlesMap[key].alsTitel())
	}
	ids, err := repository.LegeImportTitelAn(ctx, tx, neu)
	if err != nil {
		return 0, err
	}
	for i, key := range newTitlesOrder {
		t := newTitlesMap[key]
		if t.ISBN != "" {
			lookup.isbnToID[t.ISBN] = ids[i]
		}
		lookup.titelToID[repository.NormalisiereTitelKey(t.Titel)] = ids[i]
	}
	return len(ids), nil
}

// sammleExemplare sammelt (zweiter Pass) alle einzufügenden Exemplare, jetzt mit
// den vollständigen Titel-IDs aus Pass 1.
func sammleExemplare(rows [][]string, headerMap map[string]int, lookup titelLookup) []repository.ImportExemplar {
	var copiesToInsert []repository.ImportExemplar

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

		isbn := isbnutil.CleanISBN(spaltenWert(row, headerMap, "isbn"))
		titelID := matchTitelID(isbn, titel, lookup)

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
			copiesToInsert = append(copiesToInsert, repository.ImportExemplar{
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
func sammleSignaturUpdates(rows [][]string, headerMap map[string]int, lookup titelLookup) map[string]string {
	if _, ok := headerMap["signatur"]; !ok {
		return nil
	}

	updates := make(map[string]string)
	for _, row := range rows[1:] {
		z := titelZeilenFelder(row, headerMap)
		if z.titel == "" || z.signatur == "" {
			continue
		}
		isbn := isbnutil.CleanISBN(spaltenWert(row, headerMap, "isbn"))
		if id := matchTitelID(isbn, z.titel, lookup); id != "" {
			updates[id] = z.signatur
		}
	}
	return updates
}

// importSignaturen bringt die gesammelten Signaturen in die Form der Datenbankschicht. Trägt
// eine Signatur Litteras LMF-Kennung, markiert der Import den Titel zugleich als Lernmittel.
func importSignaturen(updates map[string]string) []repository.ImportSignatur {
	signaturen := make([]repository.ImportSignatur, 0, len(updates))
	for id, signatur := range updates {
		signaturen = append(signaturen, repository.ImportSignatur{
			TitelID: id, Signatur: signatur, Lernmittel: lmf.HatKennung(signatur),
		})
	}
	return signaturen
}

// ImportDynamic verarbeitet die in rows übergebenen Daten (aus CSV oder XLSX).
// Die Spalten werden über die headerMap dynamisch zugeordnet.
func (s *ImportService) ImportDynamic(ctx context.Context, rows [][]string, headerMap map[string]int) (int, int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer db.SafeRollback(ctx, tx)

	isbnToID, titelToID, err := repository.LadeTitelBestand(ctx, tx)
	if err != nil {
		return 0, 0, err
	}
	lookup := titelLookup{isbnToID: isbnToID, titelToID: titelToID}

	newTitlesMap, newTitlesOrder := sammleNeueTitel(rows, headerMap, lookup)

	newTitlesCount, err := fuegeNeueTitelEin(ctx, tx, newTitlesMap, newTitlesOrder, lookup)
	if err != nil {
		return 0, 0, err
	}

	signaturen := importSignaturen(sammleSignaturUpdates(rows, headerMap, lookup))
	if err := repository.SetzeImportSignaturen(ctx, tx, signaturen); err != nil {
		return 0, 0, err
	}

	copiesToInsert := sammleExemplare(rows, headerMap, lookup)

	importedCopiesCount, skippedCount, err := repository.LegeImportExemplareAn(ctx, tx, copiesToInsert)
	if err != nil {
		return 0, 0, err
	}
	if skippedCount > 0 {
		log.Printf("Warnung: %d Exemplare wurden übersprungen (bereits vorhanden)", skippedCount)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}

	return newTitlesCount, importedCopiesCount, nil
}
