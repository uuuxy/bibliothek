package repository

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"bibliothek/pkg/isbnutil"
)

// SchreibeTitel speichert viele Titel in einem gepipelineten Batch statt
// je Titel eine eigene Datenbank-Rundreise (der frühere N+1 ließ den MAB2-Import
// mit ~15.000 Titeln gegen eine nicht-lokale DB in Client-Timeouts laufen).
//
// Jeder Titel wird zuerst über die ISBN, dann über den Titel gegen den Bestand
// gematcht. Der Titel-Fallback gilt auch für Datensätze mit ISBN: die Bestands-CSV
// legt Titel ohne ISBN an — würde das Katalogisat-XML danach nur über die ISBN
// matchen, entstünde für jeden dieser Titel eine Dublette. Beim Titel-Match wird
// die ISBN nachgetragen (eine bereits gesetzte ISBN bleibt unangetastet). Die
// Signatur wird beim Update nie mit einem leeren Wert überschrieben (das
// Rücken-Etikett gewinnt).
//
// Zurückgegeben wird die Zahl der verarbeiteten (eingefügten oder
// aktualisierten) Titel.
func (a *pgKatalogAbfragen) SchreibeTitel(ctx context.Context, titel []Titel) (int, error) {
	if len(titel) == 0 {
		return 0, nil
	}

	// Alles-oder-nichts: entweder der komplette Import landet, oder gar nichts
	// (kein halb importierter Katalog bei einem Fehler in der Mitte).
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck // No-op nach erfolgreichem Commit.

	// 1. Bestand einmalig vorladen.
	bestand, err := LadeTitelBestand(ctx, tx)
	if err != nil {
		return 0, err
	}
	titel = mitISBNInNormalform(titel)

	// autor/verlag/erscheinungsjahr sind wie signatur/isbn per COALESCE geschützt:
	// eine Quelle ohne diese Angabe (z. B. MAB-Exporte ohne Autor-Feld) darf einen
	// bereits bekannten, besseren Wert nicht stillschweigend leeren.
	//
	// ist_lernmittel wird per OR nur gesetzt, nie gelöscht: Ein Buch, das jemand in der
	// Maske als Lernmittel markiert hat, verliert das nicht durch einen Export, der es
	// nicht kennt. Das Fach füllt nur eine Leerstelle. Die Jahrgangsspanne
	// dagegen folgt der Quelle, sobald sie beide Werte nennt: Litteras Signatur ist für
	// den Altbestand die gepflegte Quelle. Nennt die Quelle keine, bleibt am Titel, was
	// dort steht.
	//
	// Ausgenommen ist ein Mehrjahresband: Den Schalter gibt es nur in
	// der Titel-Verwaltung, und wer ihn setzt, setzt die Spanne mit — sie ist dann
	// gepflegt, und an ihrem „bis" hängt die Frist. Die Signatur nennt oft nur einen
	// Jahrgang („LMF Bio 7"); übernommen, verletzte sie chk_mehrjahresband_spanne, und der
	// ganze Import fiele.
	const sqlAendern = `
		UPDATE buecher_titel SET
			titel = $2,
			autor = COALESCE(NULLIF($3, ''), autor),
			verlag = COALESCE(NULLIF($4, ''), verlag),
			erscheinungsjahr = COALESCE(NULLIF($5, 0), erscheinungsjahr),
			signatur = COALESCE(NULLIF($6, ''), signatur),
			isbn = COALESCE(isbn, NULLIF($7, '')),
			ist_lernmittel = ist_lernmittel OR $8,
			subject = COALESCE(subject, NULLIF($9, '')),
			jahrgang_von = CASE WHEN $10 > 0 AND $11 > 0 AND NOT mehrjahresband THEN $10 ELSE jahrgang_von END,
			jahrgang_bis = CASE WHEN $10 > 0 AND $11 > 0 AND NOT mehrjahresband THEN $11 ELSE jahrgang_bis END,
			aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	// subject ist FK auf die Systematik: unbekannte Fächer vor dem
	// SendBatch in derselben Transaktion registrieren (danach ist die Verbindung bis
	// br.Close() belegt) und jede Zeile auf die kanonische Schreibweise ziehen.
	kanonisch, err := StelleFaecherSicher(ctx, tx, faecherDerTitel(titel))
	if err != nil {
		return 0, err
	}

	// 2. In-Batch-Dedup (letzter Datensatz gewinnt): verhindert, dass derselbe
	//    Titel bzw. dieselbe ISBN innerhalb einer Datei doppelt geschrieben wird.
	c := &titelSchreibLauf{
		bestand:       bestand,
		geseheneISBN:  make(map[string]bool),
		geseheneTitel: make(map[string]bool),
	}

	batch := &pgx.Batch{}
	eingereiht := 0
	for _, t := range titel {
		t.Fach = kanonisch[t.Fach]
		if reiheTitelEin(batch, t, c, sqlAendern) {
			eingereiht++
		}
	}

	if eingereiht == 0 {
		return 0, nil
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < eingereiht; i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close() //nolint:errcheck
			return 0, err
		}
	}
	if err := br.Close(); err != nil {
		return 0, err
	}
	if err := ergaenzeSchlagworteAusImport(ctx, tx, titel); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return eingereiht, nil
}

// mitISBNInNormalform bringt die ISBN jedes Datensatzes in die Form, in der die Datenbank
// speichert: Die Zuordnung vergleicht Zeichen für Zeichen. Zehnstellig aus der Datei träfe sie
// den Titel mit der dreizehnstelligen sonst nicht, und der INSERT scheiterte am UNIQUE-Index.
// Auf einer Kopie; die Liste des Aufrufers bleibt, wie sie ist.
func mitISBNInNormalform(titel []Titel) []Titel {
	titel = slices.Clone(titel)
	for i := range titel {
		titel[i].ISBN = isbnutil.Normalform(titel[i].ISBN)
	}
	return titel
}

// faecherDerTitel sammelt die Fächer, die die Titel nennen.
func faecherDerTitel(titel []Titel) []string {
	faecher := make([]string, 0, len(titel))
	for _, t := range titel {
		if t.Fach != "" {
			faecher = append(faecher, t.Fach)
		}
	}
	return faecher
}

// ergaenzeSchlagworteAusImport schreibt die Schlagworte des Katalogisats an die Titel, die noch
// keine tragen — über SetzeSchlagworte, den Pfad des
// Buchformulars, aufbereitet wie in der Übernahme aus der Sicherung (SchlagworteAusFremddaten).
// Wer schon Schlagworte trägt, behält seine: Ein erneuter Import überschreibt keine Pflege —
// dieselbe Regel wie beim Fach, das nur Leerstellen füllt. Nennen zwei Datensätze denselben
// Titel, gilt der erste, der Schlagworte trägt.
//
// Die Titel werden nach dem Batch über dieselbe Zuordnung gefunden wie im Upsert
// (reiheTitelEin): zuerst die ISBN, sonst der Titel. Neu angelegte stehen dann schon in der
// Tabelle, der Batch gibt ihre Kennungen nicht zurück. Gefragt wird für jeden Datensatz der
// Datei, auch für die, die der Upsert als Dublette übergangen hat: Im Katalogisat vom Juni 2026
// steht mancher Titel zweimal, der erste Eintrag ohne Schlagworte, der zweite mit. Nur mit den
// eingereihten Datensätzen brächte erst ein zweiter Lauf deren Schlagworte (gemessen:
// 253 Zuordnungen mehr); so bringt sie der erste. Ein zweiter Lauf derselben Datei
// ergänzt danach höchstens Titel, die der Upsert selbst erst beim zweiten Mal zuordnet (am
// Katalogisat vom Juni 2026: einer), und ersetzt nie vorhandene Schlagworte.
func ergaenzeSchlagworteAusImport(ctx context.Context, tx pgx.Tx, titel []Titel) error {
	if !slices.ContainsFunc(titel, func(t Titel) bool { return len(t.Schlagworte) > 0 }) {
		return nil
	}
	bestand, err := LadeTitelBestand(ctx, tx)
	if err != nil {
		return err
	}
	ohne, err := ladeTitelOhneSchlagworte(ctx, tx)
	if err != nil {
		return err
	}
	for _, t := range titel {
		woerter, _ := SchlagworteAusFremddaten(t.Schlagworte)
		if len(woerter) == 0 {
			continue
		}
		id, _ := bestand.TitelZu(t.ISBN, t.Titel)
		if !ohne[id] {
			continue // trägt schon Schlagworte, oder der Titel ist nicht zu finden
		}
		if _, err := SetzeSchlagworte(ctx, tx, id, woerter); err != nil {
			return fmt.Errorf("schlagworte für %q: %w", t.Titel, err)
		}
		ohne[id] = false
	}
	return nil
}

// ladeTitelOhneSchlagworte liefert die Kennungen der Titel, die kein Schlagwort tragen.
func ladeTitelOhneSchlagworte(ctx context.Context, tx pgx.Tx) (map[string]bool, error) {
	rows, err := tx.Query(ctx, `
		SELECT t.id::text FROM buecher_titel t
		WHERE NOT EXISTS (SELECT 1 FROM titel_schlagworte ts WHERE ts.titel_id = t.id)`)
	if err != nil {
		return nil, fmt.Errorf("titel ohne schlagworte lesen: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("titel ohne schlagworte lesen: %w", err)
	}
	ohne := make(map[string]bool, len(ids))
	for _, id := range ids {
		ohne[id] = true
	}
	return ohne, nil
}

// titelAusImportAnlegen legt einen Titel an, wie ein Import ihn liefert. Lernmittel, Fach und
// Jahrgang liest der Import aus Litteras Signatur „LMF Bio 7", den Schlagwörtern und der
// Zielgruppe (pkg/lmf); ohne Angabe bleibt der Jahrgang leer, das heißt unbekannt.
const titelAusImportAnlegen = `
		INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject, signatur,
		                           ist_lernmittel, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), NULLIF($6, ''), NULLIF($7, ''),
		        $8, NULLIF($9, 0), NULLIF($10, 0))
		RETURNING id
	`

// reiheTitelAnlageEin reiht das Anlegen eines Titels in einen Sammelauftrag ein. Katalog-Import
// und Bestands-Import legen neue Titel nur hier an; das Fach muss vorher in der Systematik
// stehen (StelleFaecherSicher).
func reiheTitelAnlageEin(batch *pgx.Batch, t Titel) {
	batch.Queue(titelAusImportAnlegen, t.Titel, t.Autor, t.Verlag, t.ISBN, t.Erscheinungsjahr, t.Fach, t.Signatur,
		t.IstLernmittel, t.JahrgangVon, t.JahrgangBis)
}

// titelSchreibLauf bündelt den Bestand und was der Lauf in dieser Datei schon eingereiht hat.
type titelSchreibLauf struct {
	bestand                     TitelBestand
	geseheneISBN, geseheneTitel map[string]bool
}

// reiheTitelEin reiht einen Titel als UPDATE (bekannt) oder INSERT (neu) in den Batch ein.
// Ob er bekannt ist, sagt der Bestand (TitelBestand.TitelZu). Innerhalb der Datei wird nach
// ISBN und Titel dedupliziert. Liefert false, wenn der Titel übersprungen wurde (leer oder
// Dublette).
func reiheTitelEin(batch *pgx.Batch, t Titel, c *titelSchreibLauf, sqlAendern string) bool {
	if t.Titel == "" {
		return false
	}
	if t.ISBN != "" && c.geseheneISBN[t.ISBN] {
		return false
	}

	titelSchluessel := NormalisiereTitelSchluessel(t.Titel)
	id, bekannt := c.bestand.TitelZu(t.ISBN, t.Titel)
	// Unbekannter Titel, aber bereits in dieser Datei eingereiht: kein zweiter
	// INSERT — sonst legt eine Datei mit ISBN- und ISBN-losen Varianten desselben
	// Titels zwei Zeilen an.
	if !bekannt && c.geseheneTitel[titelSchluessel] {
		return false
	}
	c.geseheneTitel[titelSchluessel] = true
	if t.ISBN != "" {
		c.geseheneISBN[t.ISBN] = true
	}

	if bekannt {
		batch.Queue(sqlAendern, id, t.Titel, t.Autor, t.Verlag, t.Erscheinungsjahr, t.Signatur, t.ISBN,
			t.IstLernmittel, t.Fach, t.JahrgangVon, t.JahrgangBis)
	} else {
		reiheTitelAnlageEin(batch, t)
	}
	return true
}
