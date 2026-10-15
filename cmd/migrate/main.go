// Command migrate überträgt einmalig einen Bestand aus MySQL nach PostgreSQL.
//
// Aufruf:
//
//	MYSQL_DSN="user:pass@tcp(host:3306)/olddb?parseTime=true" \
//	PG_DSN="postgres://user:pass@host:5432/newdb" \
//	go run ./cmd/migrate [--dry-run] [--batch 500]
//
// Pflichtangaben in der Umgebung:
//
//	MYSQL_DSN   – DSN von MySQL im Format von go-sql-driver/mysql
//	PG_DSN      – Verbindung zu PostgreSQL (URL oder DSN)
//
// Schalter:
//
//	--dry-run   – nur prüfen und protokollieren, nichts nach PostgreSQL schreiben
//	--batch N   – Titel je Transaktion (Vorgabe: 200)
//
// Rückgabewerte, die Kurzfassung des Abschlussberichts:
//
//	0 – vollständig übernommen. Warnungen können im Protokoll stehen, aber jeder
//	    Quelldatensatz ist in PostgreSQL angekommen.
//	1 – abgebrochen. Ab dem gemeldeten Punkt wurde nichts mehr geschrieben.
//	2 – abgeschlossen, aber unvollständig: einzelne Datensätze fehlen. Welche,
//	    steht als FEHLER-Zeile in migration_errors.log.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"bibliothek/internal/uebernahme"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	trocken      = flag.Bool("dry-run", false, "validate only, do not write to PostgreSQL")
	batchGroesse = flag.Int("batch", 200, "titles per INSERT transaction")
)

const (
	rueckgabeOK             = 0
	rueckgabeAbgebrochen    = 1
	rueckgabeUnvollstaendig = 2
)

// main hält nichts als den Rückgabewert. Alle Arbeit steckt in fuehreAus, damit die Kette der
// defer wirklich läuft: Das Fehlerprotokoll wird über einen Puffer von 64 KB geschrieben, und ein
// os.Exit mitten im Ablauf ließe den Puffer ungeleert, genau in dem Lauf, für den man das
// Protokoll braucht.
func main() {
	os.Exit(fuehreAus())
}

// fuehreAus ist der ganze Ablauf: Umgebung lesen, Quelle laden, prüfen oder übernehmen,
// berichten.
func fuehreAus() int {
	flag.Parse()

	mysqlDSN, pgDSN, err := leseUmgebung()
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: %v", err)
		return rueckgabeAbgebrochen
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fehlerprotokoll, err := newFehlerprotokoll()
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: Fehlerprotokoll konnte nicht geöffnet werden: %v", err)
		return rueckgabeAbgebrochen
	}
	defer fehlerprotokoll.Schliesse()

	mysqlDB, err := verbindeMySQL(ctx, mysqlDSN)
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: %v", err)
		return rueckgabeAbgebrochen
	}
	defer func() { _ = mysqlDB.Close() }() //nolint:errcheck

	log.Println("Lese Quelldaten aus MySQL …")
	medien, err := ladeMySQLTitel(mysqlDB)
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: mysql read: %v", err)
		return rueckgabeAbgebrochen
	}
	// #nosec G706
	log.Printf("Gelesen: %d Titel aus MySQL", len(medien))

	if *trocken {
		fuehreTrockenlaufAus(medien, fehlerprotokoll)
		return rueckgabeOK
	}

	pool, err := verbindePostgres(ctx, pgDSN)
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: %v", err)
		return rueckgabeAbgebrochen
	}
	defer pool.Close()

	bericht := fuehreUebernahmeDurch(ctx, pool, medien, fehlerprotokoll)
	bericht.Warnungen, bericht.Fehler = fehlerprotokoll.Warnungen(), fehlerprotokoll.FehlerAnzahl()
	druckeZusammenfassung(bericht)
	return bericht.rueckgabewert()
}

// abschlussbericht ist die vollständige Bilanz eines Laufs, einschließlich dessen, was nicht
// geklappt hat. Behauptung (was der Code gezählt hat) und Befund (was in der
// Datenbank steht) werden getrennt geführt und am Ende gegeneinander geprüft.
type abschlussbericht struct {
	QuellTitel     int
	Titel          int // gemeldet: erfolgreich geschrieben
	Exemplare      int
	Uebersprungen  int
	Warnungen      int
	Fehler         int
	LetzterBarcode int

	IstTitel     int // tatsächlicher Zeilenzuwachs in PostgreSQL
	IstExemplare int
	AbgleichOK   bool

	Abbruch error // gesetzt, wenn die Übernahme vorzeitig endete
}

// rueckgabewert fasst den Bericht in einem Rückgabewert zusammen.
func (b abschlussbericht) rueckgabewert() int {
	switch {
	case b.Abbruch != nil:
		return rueckgabeAbgebrochen
	case b.Uebersprungen > 0 || b.Fehler > 0 || !b.AbgleichOK:
		return rueckgabeUnvollstaendig
	default:
		return rueckgabeOK
	}
}

// leseUmgebung liest die beiden Verbindungen aus der Umgebung.
func leseUmgebung() (mysqlDSN, pgDSN string, err error) {
	mysqlDSN = os.Getenv("MYSQL_DSN")
	if mysqlDSN == "" {
		return "", "", fmt.Errorf("MYSQL_DSN environment variable is required")
	}
	pgDSN = os.Getenv("PG_DSN")
	if pgDSN == "" {
		return "", "", fmt.Errorf("PG_DSN environment variable is required")
	}
	return mysqlDSN, pgDSN, nil
}

// verbindeMySQL öffnet die Quelle und prüft die Verbindung.
func verbindeMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	log.Println("Verbinde mit MySQL …")
	verbindung, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	verbindung.SetMaxOpenConns(4)
	verbindung.SetConnMaxLifetime(5 * time.Minute)
	if err := verbindung.PingContext(ctx); err != nil {
		_ = verbindung.Close() //nolint:errcheck
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	log.Println("MySQL Verbindung OK")
	return verbindung, nil
}

// verbindePostgres öffnet das Ziel und prüft die Verbindung.
func verbindePostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	log.Println("Verbinde mit PostgreSQL …")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx pool new: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg ping: %w", err)
	}
	log.Println("PostgreSQL Verbindung OK")
	return pool, nil
}

// fuehreTrockenlaufAus prüft die Quelldaten, ohne zu schreiben, einschließlich der Länge der
// Freitexte, denn überlange Felder sind neben doppelten ISBNs der zweite Grund, aus dem eine
// Übernahme Datensätze abwertet. Wer den Trockenlauf macht, will beides vorher wissen.
func fuehreTrockenlaufAus(medien []mysqlMedium, fehlerprotokoll *uebernahme.Protokoll) {
	log.Println("DRY-RUN: Validiere Datensätze ohne Schreibzugriff auf PostgreSQL …")
	bekannteISBNs := make(map[string]string)
	for _, m := range medien {
		uebernahme.KlaereISBN(fehlerprotokoll, quellID(m), isbnRoh(m), bekannteISBNs)
		kuerzeFelder(fehlerprotokoll, m)
	}
	// #nosec G706
	log.Printf("DRY-RUN abgeschlossen: %d Anmerkungen → %s", fehlerprotokoll.Warnungen(), fehlerprotokollPfad)
}

// fuehreUebernahmeDurch klammert den eigentlichen Import zwischen zwei Zählungen des
// tatsächlichen Bestands. Der Abgleich am Ende ist die Notbremse gegen eine Zahl im
// Abschlussbericht, die niemand gegen die Datenbank gehalten hat.
func fuehreUebernahmeDurch(
	ctx context.Context, pool *pgxpool.Pool, medien []mysqlMedium, fehlerprotokoll *uebernahme.Protokoll,
) abschlussbericht {
	bericht := abschlussbericht{QuellTitel: len(medien)}

	vorherTitel, vorherExemplare, err := zaehleBestand(ctx, pool)
	if err != nil {
		bericht.Abbruch = fmt.Errorf("konnte den Bestand vor der Übernahme nicht zählen: %w", err)
		return bericht
	}

	importiere(ctx, pool, medien, fehlerprotokoll, &bericht)

	nachherTitel, nachherExemplare, err := zaehleBestand(ctx, pool)
	if err != nil {
		// Der Import selbst ist gelaufen; nur der Nachweis fehlt. Das ist kein Abbruch, aber auch
		// kein „alles gut"; deshalb bleibt AbgleichOK false.
		// #nosec G706
		log.Printf("WARNUNG: Bestand nach der Übernahme nicht zählbar, Abgleich entfällt: %v", err)
		return bericht
	}
	bericht.IstTitel = nachherTitel - vorherTitel
	bericht.IstExemplare = nachherExemplare - vorherExemplare
	bericht.AbgleichOK = bericht.IstTitel == bericht.Titel && bericht.IstExemplare == bericht.Exemplare
	return bericht
}

// zaehleBestand liest den tatsächlichen Stand der Zeilen.
//
// Voraussetzung des Abgleichs: Während der Übernahme
// Datenbank. Für ein Werkzeug, das einmal pro Schuljahr von Hand gestartet wird, ist das
// die richtige Annahme.
func zaehleBestand(ctx context.Context, pool *pgxpool.Pool) (titel, exemplare int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM buecher_titel),
		       (SELECT count(*) FROM buecher_exemplare)
	`).Scan(&titel, &exemplare)
	return titel, exemplare, err
}

// importiere schreibt die Titel in Batches. Meldet ein Batch einen Fehler, ist die Übernahme
// zu Ende: Der Fehler betrifft dann nicht mehr einen einzelnen Datensatz, sondern die
// Verbindung oder die Transaktion selbst. Weiterzulaufen hieße, Tausende Zeilen als
// „übersprungen" zu protokollieren, die in Wahrheit nie versucht wurden.
func importiere(
	ctx context.Context, pool *pgxpool.Pool, medien []mysqlMedium, fehlerprotokoll *uebernahme.Protokoll, bericht *abschlussbericht,
) {
	barcodeNummer, err := hoechsteBarcodeNummer(ctx, pool)
	if err != nil {
		bericht.Abbruch = fmt.Errorf("barcode seq lookup: %w", err)
		return
	}
	bericht.LetzterBarcode = barcodeNummer
	// #nosec G706
	log.Printf("Höchster vorhandener Barcode: B-%05d → Nächster: B-%05d", barcodeNummer, barcodeNummer+1)

	bekannteISBNs := make(map[string]string)
	batchAnzahl := int(math.Ceil(float64(len(medien)) / float64(*batchGroesse)))

	for i := 0; i < len(medien); i += *batchGroesse {
		ende := min(i+*batchGroesse, len(medien))
		teil := medien[i:ende]
		batchNummer := (i / *batchGroesse) + 1

		// #nosec G706
		log.Printf("Batch %d/%d: importiere %d Titel …", batchNummer, batchAnzahl, len(teil))
		ergebnis, err := legeBatchAn(ctx, pool, teil, bekannteISBNs, fehlerprotokoll, &barcodeNummer)
		bericht.LetzterBarcode = barcodeNummer
		if err != nil {
			bericht.Abbruch = fmt.Errorf("abgebrochen in Batch %d/%d (Quellzeilen ab mysql_id=%d): %w",
				batchNummer, batchAnzahl, teil[0].ID, err)
			return
		}
		bericht.Titel += ergebnis.Titel
		bericht.Exemplare += ergebnis.Exemplare
		bericht.Uebersprungen += ergebnis.Uebersprungen
		// #nosec G706
		log.Printf("  → %d Titel, %d Exemplare eingetragen, %d übersprungen",
			ergebnis.Titel, ergebnis.Exemplare, ergebnis.Uebersprungen)
	}
}

// druckeZusammenfassung gibt den Abschlussbericht ins Log.
func druckeZusammenfassung(b abschlussbericht) {
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if b.Abbruch != nil {
		// #nosec G706
		log.Printf("Migration ABGEBROCHEN:")
	} else {
		// #nosec G706
		log.Printf("Migration abgeschlossen:")
	}
	// #nosec G706
	log.Printf("  Quell-Titel (MySQL):      %d", b.QuellTitel)
	// #nosec G706
	log.Printf("  Importierte Titel:        %d", b.Titel)
	// #nosec G706
	log.Printf("  Importierte Exemplare:    %d", b.Exemplare)
	// #nosec G706
	log.Printf("  Übersprungene Titel:      %d", b.Uebersprungen)
	// #nosec G706
	log.Printf("  Warnungen (übernommen):   %d → %s", b.Warnungen, fehlerprotokollPfad)
	// #nosec G706
	log.Printf("  Fehler (NICHT übernommen):%d → %s", b.Fehler, fehlerprotokollPfad)
	// #nosec G706
	log.Printf("  Letzter Barcode:          B-%05d", b.LetzterBarcode)
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	druckeAbgleich(b)

	if b.Abbruch != nil {
		// #nosec G706
		log.Printf("⚠  Die Übernahme endete vorzeitig: %v", b.Abbruch)
		// #nosec G706
		log.Printf("⚠  Alles ab diesem Punkt wurde NICHT geschrieben. Ursache beheben und erneut starten.")
		return
	}
	if b.Uebersprungen > 0 {
		// #nosec G706
		log.Printf("⚠  %d Titel wurden übersprungen und fehlen in der neuen Datenbank. "+
			"Details je Datensatz: grep FEHLER %s", b.Uebersprungen, fehlerprotokollPfad)
	}
	if b.Warnungen > 0 {
		// #nosec G706
		log.Printf("ℹ  %d Datensätze wurden abgewertet übernommen (ISBN verworfen oder Feld gekürzt): "+
			"grep WARNUNG %s", b.Warnungen, fehlerprotokollPfad)
	}
}

// druckeAbgleich meldet das Ergebnis der Gegenprobe. Eine Abweichung heißt: Der Bericht oben
// ist falsch, und zwar zugunsten des Werkzeugs. Das muss lauter stehen als die Zahlen selbst.
func druckeAbgleich(b abschlussbericht) {
	if b.Abbruch != nil {
		return
	}
	if b.AbgleichOK {
		// #nosec G706
		log.Printf("✓  Abgleich mit der Datenbank: %d Titel / %d Exemplare tatsächlich neu angelegt.",
			b.IstTitel, b.IstExemplare)
		return
	}
	// #nosec G706
	log.Printf("⚠  ABGLEICH FEHLGESCHLAGEN: gemeldet %d Titel / %d Exemplare, "+
		"tatsächlich neu in der Datenbank %d Titel / %d Exemplare.",
		b.Titel, b.Exemplare, b.IstTitel, b.IstExemplare)
	// #nosec G706
	log.Printf("⚠  Den Zahlen oben ist nicht zu trauen. Bestand prüfen, bevor die alte Datenbank abgeschaltet wird.")
}
