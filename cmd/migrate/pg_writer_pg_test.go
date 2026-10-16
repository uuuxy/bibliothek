package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// quellzeile baut einen Datensatz, wie ihn ladeMySQLTitel liefern würde.
func quellzeile(id int, titel, isbn string, anzahl int) mysqlMedium {
	m := mysqlMedium{ID: id, Titel: titel, Anzahl: anzahl}
	if isbn != "" {
		m.ISBN = sql.NullString{String: isbn, Valid: true}
	}
	return m
}

// Gültige ISBN-13 (die Prüfziffer stimmt), sonst verwürfe KlaereISBN sie, und der Fall liefe an
// der Kollision vorbei.
const (
	isbnA = "9783161484100"
	isbnB = "9780306406157"
)

func legeTitelAn(t *testing.T, pool *pgxpool.Pool, titel, isbn string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2) RETURNING id`,
		titel, nullbarerText(isbn)).Scan(&id)
	if err != nil {
		t.Fatalf("Vorbedingung: Titel %q konnte nicht angelegt werden: %v", titel, err)
	}
	return id
}

func legeExemplarAn(t *testing.T, pool *pgxpool.Pool, titelID, barcode string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2)`,
		titelID, barcode); err != nil {
		t.Fatalf("Vorbedingung: Exemplar %q konnte nicht angelegt werden: %v", barcode, err)
	}
}

func zaehleTitel(t *testing.T, pool *pgxpool.Pool, bedingung string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM buecher_titel WHERE `+bedingung, args...).Scan(&n); err != nil {
		t.Fatalf("Zählung fehlgeschlagen: %v", err)
	}
	return n
}

// Der Kern: Ein einziger kollidierender Datensatz in der Mitte eines Batches darf die anderen
// nicht mitreißen.
//
// Ohne Savepoint versetzte der 23505 auf buecher_titel.isbn die Transaktion in den Zustand des
// Abbruchs, alle folgenden INSERTs scheiterten mit 25P02, und der COMMIT würde zum ROLLBACK:
// 0 von 5 Titeln in der Datenbank, bei einem Fehlerprotokoll, das vier weitere Zeilen als
// „übersprungen" meldet.
func TestBatchUeberlebtEinzelnenZeilenfehler(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	fehlerprotokoll, protokoll := testFehlerprotokoll(t)

	// Ein früherer, abgebrochener Lauf hat diese ISBN bereits hinterlassen.
	legeTitelAn(t, pool, "Aus einem früheren Lauf", isbnA)

	teil := []mysqlMedium{
		quellzeile(1, "Erster Titel", "", 2),
		quellzeile(2, "Zweiter Titel", "", 1),
		quellzeile(3, "Kollidierender Titel", isbnA, 3), // 23505 auf isbn
		quellzeile(4, "Vierter Titel", "", 1),
		quellzeile(5, "Fünfter Titel", "", 2),
	}

	nummer := 0
	ergebnis, err := legeBatchAn(context.Background(), pool, teil, map[string]string{}, fehlerprotokoll, &nummer)
	if err != nil {
		t.Fatalf("Ein Datensatzfehler darf den Batch nicht abbrechen, meldete aber: %v", err)
	}

	if ergebnis.Titel != 4 {
		t.Errorf("4 übernommene Titel erwartet, gemeldet: %d", ergebnis.Titel)
	}
	if ergebnis.Uebersprungen != 1 {
		t.Errorf("1 übersprungener Titel erwartet, gemeldet: %d", ergebnis.Uebersprungen)
	}
	if ergebnis.Exemplare != 6 { // 2+1+1+2, die 3 des kollidierenden Titels fehlen
		t.Errorf("6 übernommene Exemplare erwartet, gemeldet: %d", ergebnis.Exemplare)
	}

	// Die Gegenprobe an der Datenbank: Die Meldung oben ist wertlos, wenn der COMMIT nicht
	// durchkam.
	if n := zaehleTitel(t, pool, `titel <> 'Aus einem früheren Lauf'`); n != 4 {
		t.Errorf("4 Titel sollten committet in der Datenbank stehen, gefunden: %d", n)
	}
	if n := zaehleTitel(t, pool, `titel = 'Kollidierender Titel'`); n != 0 {
		t.Errorf("der kollidierende Titel darf nicht in der Datenbank stehen, gefunden: %d", n)
	}

	if fehlerprotokoll.FehlerAnzahl() != 1 || fehlerprotokoll.Warnungen() != 0 {
		t.Errorf("Fehler/Warnungen = %d/%d, erwartet 1/0", fehlerprotokoll.FehlerAnzahl(), fehlerprotokoll.Warnungen())
	}

	// Das Protokoll muss die Ursache benennen, nicht nur den Ausfall; dafür gibt es das
	// Werkzeug.
	text := protokoll()
	for _, erwartet := range []string{"FEHLER", "mysql_id=3", isbnA, "23505"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("Fehlerprotokoll nennt %q nicht:\n%s", erwartet, text)
		}
	}
}

// Sichert die Wahl der atomaren Einheit ab: Die Stückzahl der Quelle wird vollständig zu
// Exemplaren; ein Titel mit Anzahl 3 und nur einem Exemplar wäre ein unbemerkter Fehler im
// Bestand.
func TestTitelKommtGanzOderGarNicht(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	fehlerprotokoll, protokoll := testFehlerprotokoll(t)

	// B-00002 ist schon vergeben; das zweite von drei Exemplaren kollidiert.
	fremd := legeTitelAn(t, pool, "Fremdbestand", "")
	legeExemplarAn(t, pool, fremd, "B-00002")

	teil := []mysqlMedium{quellzeile(7, "Dreibändiges Werk", "", 3)}

	nummer := 0
	ergebnis, err := legeBatchAn(context.Background(), pool, teil, map[string]string{}, fehlerprotokoll, &nummer)
	if err != nil {
		t.Fatalf("Barcode-Kollision ist ein Datensatzfehler, kein Abbruch: %v", err)
	}
	if ergebnis.Titel != 0 || ergebnis.Exemplare != 0 || ergebnis.Uebersprungen != 1 {
		t.Errorf("Titel muss vollständig zurückgenommen werden, gemeldet: %+v", ergebnis)
	}
	if n := zaehleTitel(t, pool, `titel = 'Dreibändiges Werk'`); n != 0 {
		t.Errorf("kein halb befüllter Titel erlaubt, gefunden: %d", n)
	}

	// Ohne den Barcode im Text findet niemand die Kollision in einer Quelle mit 80.000 Zeilen.
	if text := protokoll(); !strings.Contains(text, "B-00002") {
		t.Errorf("Fehlerprotokoll nennt den kollidierenden Barcode nicht:\n%s", text)
	}
}

// Die Erkennung der Dubletten steht in einer Map im Arbeitsspeicher, und der Savepoint kann sie
// nicht zurückrollen. Ohne das ausdrückliche delete verlöre der nächste echte Titel seine ISBN an
// einen Datensatz, der gar nicht in der Datenbank steht.
func TestISBNReservierungWirdBeiRuecknahmeFrei(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	fehlerprotokoll, _ := testFehlerprotokoll(t)

	fremd := legeTitelAn(t, pool, "Fremdbestand", "")
	legeExemplarAn(t, pool, fremd, "B-00001") // bringt den ERSTEN Datensatz zu Fall

	teil := []mysqlMedium{
		quellzeile(11, "Scheitert am Barcode", isbnB, 1),
		quellzeile(12, "Kommt durch", isbnB, 1),
	}

	nummer := 0
	bekannt := map[string]string{}
	ergebnis, err := legeBatchAn(context.Background(), pool, teil, bekannt, fehlerprotokoll, &nummer)
	if err != nil {
		t.Fatalf("unerwarteter Abbruch: %v", err)
	}
	if ergebnis.Titel != 1 {
		t.Fatalf("1 übernommener Titel erwartet, gemeldet: %d", ergebnis.Titel)
	}

	var isbn sql.NullString
	if err := pool.QueryRow(context.Background(),
		`SELECT isbn FROM buecher_titel WHERE titel = 'Kommt durch'`).Scan(&isbn); err != nil {
		t.Fatalf("übernommener Titel nicht gefunden: %v", err)
	}
	if !isbn.Valid || isbn.String != isbnB {
		t.Errorf("ISBN %q erwartet, gespeichert: %v — die Reservierung des zurückgenommenen "+
			"Titels wurde nicht freigegeben", isbnB, isbn)
	}
	if _, belegt := bekannt[isbnB]; !belegt {
		t.Error("die ISBN sollte nun vom erfolgreichen Titel belegt sein")
	}
}

// Ein abgebrochener Kontext betrifft nicht diesen einen Datensatz, sondern alle. Dann muss die
// Übernahme enden, statt 79.700 Zeilen als „übersprungen" zu protokollieren, die nie versucht
// wurden.
func TestFatalerFehlerBrichtAbStattZuLuegen(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	fehlerprotokoll, _ := testFehlerprotokoll(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	teil := []mysqlMedium{
		quellzeile(21, "Eins", "", 1),
		quellzeile(22, "Zwei", "", 1),
	}
	nummer := 0
	ergebnis, err := legeBatchAn(ctx, pool, teil, map[string]string{}, fehlerprotokoll, &nummer)
	if err == nil {
		t.Fatal("abgebrochener Kontext muss einen Fehler liefern, kein stilles Überspringen")
	}
	if ergebnis.Titel != 0 || ergebnis.Exemplare != 0 {
		t.Errorf("bei Abbruch darf nichts als übernommen gemeldet werden, gemeldet: %+v", ergebnis)
	}
	if fehlerprotokoll.FehlerAnzahl() != 0 {
		t.Errorf("ein Abbruch ist kein Datensatzfehler und darf keine FEHLER-Zeilen erzeugen, "+
			"gezählt: %d", fehlerprotokoll.FehlerAnzahl())
	}
}

// varchar(255) ist neben doppelten ISBNs der häufigste Grund, aus dem eine Übernahme über
// Datensätze stolpert. Postgres nennt bei SQLSTATE 22001 nicht einmal die Spalte; deshalb wird
// vorher gekürzt und die Kürzung mit dem vollen ursprünglichen Wert protokolliert.
func TestUeberlangeFelderWerdenGekuerztStattVerloren(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	fehlerprotokoll, protokoll := testFehlerprotokoll(t)

	langerTitel := strings.Repeat("Ä", 300) // Umlaute: die Kürzung muss zeichenweise sein
	teil := []mysqlMedium{quellzeile(31, langerTitel, "", 1)}

	nummer := 0
	ergebnis, err := legeBatchAn(context.Background(), pool, teil, map[string]string{}, fehlerprotokoll, &nummer)
	if err != nil {
		t.Fatalf("unerwarteter Abbruch: %v", err)
	}
	if ergebnis.Titel != 1 {
		t.Fatalf("der Titel soll gekürzt ankommen, nicht verloren gehen, gemeldet: %d", ergebnis.Titel)
	}

	var gespeichert string
	if err := pool.QueryRow(context.Background(),
		`SELECT titel FROM buecher_titel LIMIT 1`).Scan(&gespeichert); err != nil {
		t.Fatalf("Titel nicht gefunden: %v", err)
	}
	if r := []rune(gespeichert); len(r) != maxTitelSpalte {
		t.Errorf("auf %d Zeichen gekürzt erwartet, gespeichert: %d Zeichen", maxTitelSpalte, len(r))
	}
	if !strings.HasPrefix(gespeichert, "ÄÄÄ") {
		t.Errorf("die Kürzung hat die Umlaute zerlegt: %q", gespeichert[:12])
	}
	if fehlerprotokoll.Warnungen() != 1 || fehlerprotokoll.FehlerAnzahl() != 0 {
		t.Errorf("Kürzung ist eine WARNUNG, kein FEHLER; gezählt: %d/%d", fehlerprotokoll.Warnungen(), fehlerprotokoll.FehlerAnzahl())
	}
	if text := protokoll(); !strings.Contains(text, "WARNUNG") || !strings.Contains(text, "titel war 300 Zeichen") {
		t.Errorf("Protokoll benennt die Kürzung nicht nachvollziehbar:\n%s", text)
	}
}

// barcode_id ist VARCHAR. Mit MAX(barcode_id) gälte die Ordnung der Zeichen, in der 'B-99999'
// größer ist als 'B-100000'. Jenseits von 100.000 Exemplaren spränge der Zähler damit zurück und
// vergäbe reihenweise Nummern, die es längst gibt; bei rund 80.000 Titeln ist die Schwelle
// erreichbar.
func TestHoechsteBarcodeNummerZaehltNumerisch(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)

	titelID := legeTitelAn(t, pool, "Bestandsträger", "")
	legeExemplarAn(t, pool, titelID, "B-99999")
	legeExemplarAn(t, pool, titelID, "B-100000")

	nummer, err := hoechsteBarcodeNummer(context.Background(), pool)
	if err != nil {
		t.Fatalf("highestBarcodeSeq: %v", err)
	}
	if nummer != 100000 {
		t.Errorf("hoechsteBarcodeNummer(…) = %d, erwartet 100000 — die Nummern werden wieder als Text sortiert", nummer)
	}
}

// Ein von Hand vergebener Barcode darf den Zähler nicht unbemerkt auf 0 zurücksetzen (der
// reguläre Ausdruck griffe sonst auf dem Treffer von MAX nicht), denn danach kollidierte jede
// Nummer.
func TestHoechsteBarcodeNummerIgnoriertFremdformate(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)

	titelID := legeTitelAn(t, pool, "Bestandsträger", "")
	legeExemplarAn(t, pool, titelID, "B-00042")
	legeExemplarAn(t, pool, titelID, "B-2024/A")

	nummer, err := hoechsteBarcodeNummer(context.Background(), pool)
	if err != nil {
		t.Fatalf("highestBarcodeSeq: %v", err)
	}
	if nummer != 42 {
		t.Errorf("hoechsteBarcodeNummer(…) = %d, erwartet 42", nummer)
	}
}

// Die leere Datenbank ist der gewöhnliche Fall beim ersten Lauf.
func TestHoechsteBarcodeNummerOhneBestand(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)

	nummer, err := hoechsteBarcodeNummer(context.Background(), pool)
	if err != nil {
		t.Fatalf("highestBarcodeSeq: %v", err)
	}
	if nummer != 0 {
		t.Errorf("hoechsteBarcodeNummer(…) = %d, erwartet 0", nummer)
	}
}
