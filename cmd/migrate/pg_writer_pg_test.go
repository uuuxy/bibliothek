package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// quellzeile baut einen Datensatz, wie ihn readMySQLTitles liefern würde.
func quellzeile(id int, titel, isbn string, anzahl int) mysqlMedium {
	m := mysqlMedium{ID: id, Titel: titel, Anzahl: anzahl}
	if isbn != "" {
		m.ISBN = sql.NullString{String: isbn, Valid: true}
	}
	return m
}

// Gültige ISBN-13 (Prüfziffer stimmt) — sonst würde klaerISBN sie verwerfen und der
// Testfall liefe an der Kollision vorbei.
const (
	isbnA = "9783161484100"
	isbnB = "9780306406157"
)

func legeTitelAn(t *testing.T, pool *pgxpool.Pool, titel, isbn string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2) RETURNING id`,
		titel, nullStr(isbn)).Scan(&id)
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

// TestBatchUeberlebtEinzelnenZeilenfehler ist der Kern: ein einziger kollidierender
// Datensatz in der Mitte eines Batches darf die anderen nicht mitreißen.
//
// Vor dem Savepoint war das anders: der 23505 auf buecher_titel.isbn versetzte die
// Transaktion in den Abbruchzustand, alle folgenden INSERTs scheiterten mit 25P02, und
// der COMMIT wurde zum ROLLBACK. Ergebnis: 0 von 5 Titeln in der Datenbank — bei einem
// Fehlerprotokoll, das brav vier weitere „übersprungen"-Zeilen meldete.
func TestBatchUeberlebtEinzelnenZeilenfehler(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	el, protokoll := testLogger(t)

	// Ein früherer, abgebrochener Lauf hat diese ISBN bereits hinterlassen.
	legeTitelAn(t, pool, "Aus einem früheren Lauf", isbnA)

	batch := []mysqlMedium{
		quellzeile(1, "Erster Titel", "", 2),
		quellzeile(2, "Zweiter Titel", "", 1),
		quellzeile(3, "Kollidierender Titel", isbnA, 3), // 23505 auf isbn
		quellzeile(4, "Vierter Titel", "", 1),
		quellzeile(5, "Fünfter Titel", "", 2),
	}

	seq := 0
	res, err := insertBatch(context.Background(), pool, batch, map[string]string{}, el, &seq)
	if err != nil {
		t.Fatalf("Ein Datensatzfehler darf den Batch nicht abbrechen, meldete aber: %v", err)
	}

	if res.Titel != 4 {
		t.Errorf("4 übernommene Titel erwartet, gemeldet: %d", res.Titel)
	}
	if res.Uebersprungen != 1 {
		t.Errorf("1 übersprungener Titel erwartet, gemeldet: %d", res.Uebersprungen)
	}
	if res.Exemplare != 6 { // 2+1+1+2, die 3 des kollidierenden Titels fehlen
		t.Errorf("6 übernommene Exemplare erwartet, gemeldet: %d", res.Exemplare)
	}

	// Die Gegenprobe an der Datenbank: die Meldung oben ist wertlos, wenn der COMMIT
	// nicht durchkam. Genau das war der alte Zustand.
	if n := zaehleTitel(t, pool, `titel <> 'Aus einem früheren Lauf'`); n != 4 {
		t.Errorf("4 Titel sollten committet in der Datenbank stehen, gefunden: %d", n)
	}
	if n := zaehleTitel(t, pool, `titel = 'Kollidierender Titel'`); n != 0 {
		t.Errorf("der kollidierende Titel darf nicht in der Datenbank stehen, gefunden: %d", n)
	}

	if el.FehlerAnzahl() != 1 || el.Warnungen() != 0 {
		t.Errorf("genau 1 FEHLER und 0 WARNUNGEN erwartet, gezählt: %d/%d", el.FehlerAnzahl(), el.Warnungen())
	}

	// Das Protokoll muss die Ursache benennen, nicht nur den Ausfall — dafür existiert
	// dieses Werkzeug.
	text := protokoll()
	for _, erwartet := range []string{"FEHLER", "mysql_id=3", isbnA, "23505"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("Fehlerprotokoll nennt %q nicht:\n%s", erwartet, text)
		}
	}
}

// TestTitelKommtGanzOderGarNicht sichert die Wahl der atomaren Einheit ab.
// buecher_titel.stock trägt die Stückzahl aus der Quelle; ein Titel mit stock=3 und nur
// einem Exemplar wäre ein stiller Bestandsfehler, den niemand je bemerkt.
func TestTitelKommtGanzOderGarNicht(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	el, protokoll := testLogger(t)

	// B-00002 ist bereits vergeben — das zweite von drei Exemplaren kollidiert.
	fremd := legeTitelAn(t, pool, "Fremdbestand", "")
	legeExemplarAn(t, pool, fremd, "B-00002")

	batch := []mysqlMedium{quellzeile(7, "Dreibändiges Werk", "", 3)}

	seq := 0
	res, err := insertBatch(context.Background(), pool, batch, map[string]string{}, el, &seq)
	if err != nil {
		t.Fatalf("Barcode-Kollision ist ein Datensatzfehler, kein Abbruch: %v", err)
	}
	if res.Titel != 0 || res.Exemplare != 0 || res.Uebersprungen != 1 {
		t.Errorf("Titel muss vollständig zurückgenommen werden, gemeldet: %+v", res)
	}
	if n := zaehleTitel(t, pool, `titel = 'Dreibändiges Werk'`); n != 0 {
		t.Errorf("kein halb befüllter Titel erlaubt, gefunden: %d", n)
	}

	// Ohne den Barcode im Text findet niemand die Kollision in einer 80.000-Zeilen-Quelle.
	if text := protokoll(); !strings.Contains(text, "B-00002") {
		t.Errorf("Fehlerprotokoll nennt den kollidierenden Barcode nicht:\n%s", text)
	}
}

// TestISBNReservierungWirdBeiRuecknahmeFrei: die Dublettenerkennung lebt in einer Map im
// Arbeitsspeicher, der Savepoint kann sie nicht zurückrollen. Ohne das ausdrückliche
// delete verlöre der nächste echte Titel seine ISBN an einen Datensatz, der gar nicht in
// der Datenbank steht.
func TestISBNReservierungWirdBeiRuecknahmeFrei(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	el, _ := testLogger(t)

	fremd := legeTitelAn(t, pool, "Fremdbestand", "")
	legeExemplarAn(t, pool, fremd, "B-00001") // bringt den ERSTEN Datensatz zu Fall

	batch := []mysqlMedium{
		quellzeile(11, "Scheitert am Barcode", isbnB, 1),
		quellzeile(12, "Kommt durch", isbnB, 1),
	}

	seq := 0
	seen := map[string]string{}
	res, err := insertBatch(context.Background(), pool, batch, seen, el, &seq)
	if err != nil {
		t.Fatalf("unerwarteter Abbruch: %v", err)
	}
	if res.Titel != 1 {
		t.Fatalf("1 übernommener Titel erwartet, gemeldet: %d", res.Titel)
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
	if _, belegt := seen[isbnB]; !belegt {
		t.Error("die ISBN sollte nun vom erfolgreichen Titel belegt sein")
	}
}

// TestFatalerFehlerBrichtAbStattZuLuegen: ein abgebrochener Kontext betrifft nicht diesen
// einen Datensatz, sondern alle. Dann muss die Übernahme enden — nicht 79.700 Zeilen als
// „übersprungen" protokollieren, die nie versucht wurden.
func TestFatalerFehlerBrichtAbStattZuLuegen(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	el, _ := testLogger(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	batch := []mysqlMedium{
		quellzeile(21, "Eins", "", 1),
		quellzeile(22, "Zwei", "", 1),
	}
	seq := 0
	res, err := insertBatch(ctx, pool, batch, map[string]string{}, el, &seq)
	if err == nil {
		t.Fatal("abgebrochener Kontext muss einen Fehler liefern, kein stilles Überspringen")
	}
	if res.Titel != 0 || res.Exemplare != 0 {
		t.Errorf("bei Abbruch darf nichts als übernommen gemeldet werden, gemeldet: %+v", res)
	}
	if el.FehlerAnzahl() != 0 {
		t.Errorf("ein Abbruch ist kein Datensatzfehler und darf keine FEHLER-Zeilen erzeugen, "+
			"gezählt: %d", el.FehlerAnzahl())
	}
}

// TestUeberlangeFelderWerdenGekuerztStattVerloren: varchar(255) ist neben doppelten ISBNs
// der häufigste Grund, aus dem eine Littera-Übernahme über Datensätze stolpert. Postgres
// nennt bei SQLSTATE 22001 nicht einmal die Spalte — deshalb wird vorher gekürzt und die
// Kürzung mit vollem Originalwert protokolliert.
func TestUeberlangeFelderWerdenGekuerztStattVerloren(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)
	el, protokoll := testLogger(t)

	langerTitel := strings.Repeat("Ä", 300) // Umlaute: die Kürzung muss zeichenweise sein
	batch := []mysqlMedium{quellzeile(31, langerTitel, "", 1)}

	seq := 0
	res, err := insertBatch(context.Background(), pool, batch, map[string]string{}, el, &seq)
	if err != nil {
		t.Fatalf("unerwarteter Abbruch: %v", err)
	}
	if res.Titel != 1 {
		t.Fatalf("der Titel soll gekürzt ankommen, nicht verloren gehen, gemeldet: %d", res.Titel)
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
	if el.Warnungen() != 1 || el.FehlerAnzahl() != 0 {
		t.Errorf("Kürzung ist eine WARNUNG, kein FEHLER; gezählt: %d/%d", el.Warnungen(), el.FehlerAnzahl())
	}
	if text := protokoll(); !strings.Contains(text, "WARNUNG") || !strings.Contains(text, "titel war 300 Zeichen") {
		t.Errorf("Protokoll benennt die Kürzung nicht nachvollziehbar:\n%s", text)
	}
}

// TestHighestBarcodeSeqZaehltNumerisch: barcode_id ist VARCHAR. Mit MAX(barcode_id) galt
// die Zeichenfolge-Ordnung, in der 'B-99999' größer ist als 'B-100000'. Jenseits von
// 100.000 Exemplaren sprang der Zähler damit zurück und vergab reihenweise Nummern, die
// es längst gab — bei rund 80.000 Titeln eine erreichbare Schwelle.
func TestHighestBarcodeSeqZaehltNumerisch(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)

	titelID := legeTitelAn(t, pool, "Bestandsträger", "")
	legeExemplarAn(t, pool, titelID, "B-99999")
	legeExemplarAn(t, pool, titelID, "B-100000")

	seq, err := highestBarcodeSeq(context.Background(), pool)
	if err != nil {
		t.Fatalf("highestBarcodeSeq: %v", err)
	}
	if seq != 100000 {
		t.Errorf("100000 erwartet, geliefert: %d — die Nummern werden wieder als Text sortiert", seq)
	}
}

// TestHighestBarcodeSeqIgnoriertFremdformate: ein handvergebener Barcode setzte den
// Zähler bisher stillschweigend auf 0 zurück (die Regex griff auf dem MAX-Treffer nicht),
// womit anschließend jede Nummer kollidierte.
func TestHighestBarcodeSeqIgnoriertFremdformate(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)

	titelID := legeTitelAn(t, pool, "Bestandsträger", "")
	legeExemplarAn(t, pool, titelID, "B-00042")
	legeExemplarAn(t, pool, titelID, "B-2024/A")

	seq, err := highestBarcodeSeq(context.Background(), pool)
	if err != nil {
		t.Fatalf("highestBarcodeSeq: %v", err)
	}
	if seq != 42 {
		t.Errorf("42 erwartet, geliefert: %d", seq)
	}
}

// TestHighestBarcodeSeqOhneBestand: die leere Datenbank ist der Normalfall beim ersten Lauf.
func TestHighestBarcodeSeqOhneBestand(t *testing.T) {
	pool := pgTestPool(t)
	leereBestand(t, pool)

	seq, err := highestBarcodeSeq(context.Background(), pool)
	if err != nil {
		t.Fatalf("highestBarcodeSeq: %v", err)
	}
	if seq != 0 {
		t.Errorf("0 erwartet, geliefert: %d", seq)
	}
}
