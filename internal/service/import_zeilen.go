package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"bibliothek/db"
	"bibliothek/pkg/isbnutil"
	"bibliothek/pkg/lmf"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

type importTitelNeu struct {
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

// spaltenWert liest den getrimmten Wert der über headerMap zugeordneten Spalte.
func spaltenWert(zeile []string, spalten map[string]int, schluessel string) string {
	if spalte, ok := spalten[schluessel]; ok && spalte < len(zeile) {
		return strings.TrimSpace(zeile[spalte])
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
func titelZeilenFelder(zeile []string, spalten map[string]int) zeilenFelder {
	z := zeilenFelder{
		titel:     bereinigeImportTitel(spaltenWert(zeile, spalten, "titel")),
		signatur:  spaltenWert(zeile, spalten, "signatur"),
		kategorie: spaltenWert(zeile, spalten, "kategorie"),
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
func sammleNeueTitel(zeilen [][]string, spalten map[string]int, bestand repository.TitelBestand) (map[string]*importTitelNeu, []string) {
	neueTitel := make(map[string]*importTitelNeu) // key: isbn or titel
	var reihenfolge []string

	for _, zeile := range zeilen[1:] {
		schluessel, t, ok := baueNeuTitelAusZeile(zeile, spalten, bestand)
		if !ok {
			continue
		}
		if _, da := neueTitel[schluessel]; da {
			continue
		}
		neueTitel[schluessel] = t
		reihenfolge = append(reihenfolge, schluessel)
	}
	return neueTitel, reihenfolge
}

// baueNeuTitelAusZeile prüft eine Zeile und liefert, falls der Bestand ihren Titel nicht
// führt, den Cache-Key und den anzulegenden Titel. ok=false bedeutet: Zeile überspringen (leer
// oder schon im Bestand).
func baueNeuTitelAusZeile(zeile []string, spalten map[string]int, bestand repository.TitelBestand) (schluessel string, t *importTitelNeu, ok bool) {
	z := titelZeilenFelder(zeile, spalten)
	barcode := spaltenWert(zeile, spalten, "barcode")
	if z.titel == "" || barcode == "" {
		return "", nil, false
	}

	// Die Form, in der die Datenbank speichert: Nennt die Datei dasselbe Buch zehn- und
	// dreizehnstellig, ist es ein Titel. Als zwei Schlüssel ergäbe es zwei INSERTs, und der
	// zweite scheiterte am UNIQUE-Index.
	isbn := isbnutil.Normalform(isbnutil.OhneTrenner(spaltenWert(zeile, spalten, "isbn")))

	if _, bekannt := bestand.TitelZu(isbn, z.titel); bekannt {
		return "", nil, false // schon vorhanden
	}

	// Needs new title
	schluessel = isbn
	if schluessel == "" {
		schluessel = repository.NormalisiereTitelSchluessel(z.titel)
	}

	var jahr int
	if j, err := strconv.Atoi(spaltenWert(zeile, spalten, "jahr")); err == nil {
		jahr = j
	}
	return schluessel, &importTitelNeu{
		Titel:         z.titel,
		Autor:         spaltenWert(zeile, spalten, "autor"),
		Verlag:        spaltenWert(zeile, spalten, "verlag"),
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
func fachDerZeile(t *importTitelNeu) string {
	if t.Fach != "" {
		return t.Fach
	}
	return lmf.FachExakt(t.Kategorie)
}

// alsTitel füllt den Titel, wie die Datenbankschicht ihn anlegt. Das Fach kommt aus der
// Lernmittelsignatur („LMF Ma 6" → Mathematik), sonst aus der Kategorie (fachDerZeile).
func (t *importTitelNeu) alsTitel() repository.Titel {
	return repository.Titel{
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

// legeNeueTitelAn legt die neuen Titel an und trägt sie im Bestand nach: Die Exemplare
// derselben Datei finden sie im zweiten Durchgang.
func legeNeueTitelAn(ctx context.Context, tx pgx.Tx, neueTitel map[string]*importTitelNeu, reihenfolge []string, bestand repository.TitelBestand) (int, error) {
	neu := make([]repository.Titel, 0, len(reihenfolge))
	for _, schluessel := range reihenfolge {
		neu = append(neu, neueTitel[schluessel].alsTitel())
	}
	ids, err := repository.LegeImportTitelAn(ctx, tx, neu)
	if err != nil {
		return 0, err
	}
	for i, schluessel := range reihenfolge {
		t := neueTitel[schluessel]
		bestand.Merke(t.ISBN, t.Titel, ids[i])
	}
	return len(ids), nil
}

// sammleExemplare sammelt (zweiter Pass) alle einzufügenden Exemplare, jetzt mit
// den vollständigen Titel-IDs aus Pass 1.
func sammleExemplare(zeilen [][]string, spalten map[string]int, bestand repository.TitelBestand) []repository.ImportExemplar {
	var exemplare []repository.ImportExemplar

	for i, zeile := range zeilen[1:] {
		// Identische Titel-Normalisierung wie Pass 1, sonst verfehlt das
		// Titel-Matching die gerade angelegten Titel.
		titel := titelZeilenFelder(zeile, spalten).titel

		// 1. String-Bereinigung & 2. Datentyp-Sicherheit
		barcodeRoh := ""
		if spalte, ok := spalten["barcode"]; ok && spalte < len(zeile) {
			barcodeRoh = zeile[spalte]
		}
		barcode := strings.TrimSpace(strings.Trim(barcodeRoh, "\uFEFF\u200B\x00\r\n\t"))

		if titel == "" {
			continue
		}

		// 3. Robustes Logging & Fehlerabfang
		if barcode == "" {
			id := fmt.Sprintf("Zeile %d", i+2)
			log.Printf("Warnung: Exemplar ID %s hat keinen Barcode", id)
			continue
		}

		isbn := isbnutil.OhneTrenner(spaltenWert(zeile, spalten, "isbn"))
		titelID, _ := bestand.TitelZu(isbn, titel)

		// Optionale Zustand-Spalte (nur in der Bestandsdatei vorhanden):
		// "verliehen" sperrt das Exemplar für neue Ausleihen, der Rohwert
		// landet als Zustandsnotiz. Fehlt die Spalte, ist das Exemplar
		// standardmäßig ausleihbar.
		istAusleihbar := true
		zustand := spaltenWert(zeile, spalten, "zustand")
		if strings.EqualFold(zustand, "verliehen") {
			istAusleihbar = false
		}

		if titelID != "" {
			exemplare = append(exemplare, repository.ImportExemplar{
				TitelID:       titelID,
				Barcode:       barcode,
				IstAusleihbar: istAusleihbar,
				ZustandNotiz:  zustand,
			})
		}
	}
	return exemplare
}

// sammleSignaturen sammelt je Titel-ID die Signatur aus der Datei (letzte
// nicht-leere gewinnt). Damit bekommen auch BESTEHENDE Titel ihre Signatur —
// der Insert-Pfad deckt nur neue Titel ab.
func sammleSignaturen(zeilen [][]string, spalten map[string]int, bestand repository.TitelBestand) map[string]string {
	if _, ok := spalten["signatur"]; !ok {
		return nil
	}

	nachTitel := make(map[string]string)
	for _, zeile := range zeilen[1:] {
		z := titelZeilenFelder(zeile, spalten)
		if z.titel == "" || z.signatur == "" {
			continue
		}
		isbn := isbnutil.OhneTrenner(spaltenWert(zeile, spalten, "isbn"))
		if id, bekannt := bestand.TitelZu(isbn, z.titel); bekannt {
			nachTitel[id] = z.signatur
		}
	}
	return nachTitel
}

// importSignaturen bringt die gesammelten Signaturen in die Form der Datenbankschicht. Trägt
// eine Signatur Litteras LMF-Kennung, markiert der Import den Titel zugleich als Lernmittel.
func importSignaturen(nachTitel map[string]string) []repository.ImportSignatur {
	signaturen := make([]repository.ImportSignatur, 0, len(nachTitel))
	for id, signatur := range nachTitel {
		signaturen = append(signaturen, repository.ImportSignatur{
			TitelID: id, Signatur: signatur, Lernmittel: lmf.HatKennung(signatur),
		})
	}
	return signaturen
}

// ImportiereZeilen verarbeitet die in rows übergebenen Daten (aus CSV oder XLSX).
// Die Spalten werden über die headerMap dynamisch zugeordnet.
func (d *ImportDienst) ImportiereZeilen(ctx context.Context, zeilen [][]string, spalten map[string]int) (int, int, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer db.SafeRollback(ctx, tx)

	bestand, err := repository.LadeTitelBestand(ctx, tx)
	if err != nil {
		return 0, 0, err
	}

	neueTitel, reihenfolge := sammleNeueTitel(zeilen, spalten, bestand)

	anzahlTitel, err := legeNeueTitelAn(ctx, tx, neueTitel, reihenfolge, bestand)
	if err != nil {
		return 0, 0, err
	}

	signaturen := importSignaturen(sammleSignaturen(zeilen, spalten, bestand))
	if err := repository.SetzeImportSignaturen(ctx, tx, signaturen); err != nil {
		return 0, 0, err
	}

	exemplare := sammleExemplare(zeilen, spalten, bestand)

	anzahlExemplare, uebersprungen, err := repository.LegeImportExemplareAn(ctx, tx, exemplare)
	if err != nil {
		return 0, 0, err
	}
	if uebersprungen > 0 {
		log.Printf("Warnung: %d Exemplare wurden übersprungen (bereits vorhanden)", uebersprungen)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}

	return anzahlTitel, anzahlExemplare, nil
}
