// cmd/migrate/main.go
//
// One-shot MySQL → PostgreSQL migration tool for the Bibliothek system.
//
// Usage:
//
//	MYSQL_DSN="user:pass@tcp(host:3306)/olddb?parseTime=true" \
//	PG_DSN="postgres://user:pass@host:5432/newdb" \
//	go run ./cmd/migrate [--dry-run] [--batch 500]
//
// Required environment variables:
//
//	MYSQL_DSN   – MySQL DSN in go-sql-driver/mysql format
//	PG_DSN      – PostgreSQL connection string (URL or DSN)
//
// Optional flags:
//
//	--dry-run   – Validate and log only; make no writes to PostgreSQL
//	--batch N   – Number of titles to insert per transaction (default: 200)
//
// Rückgabewerte — sie sind die Kurzfassung des Abschlussberichts:
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

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dryRun    = flag.Bool("dry-run", false, "validate only, do not write to PostgreSQL")
	batchSize = flag.Int("batch", 200, "titles per INSERT transaction")
)

const (
	exitOK             = 0
	exitAbgebrochen    = 1
	exitUnvollstaendig = 2
)

// main hält nichts als den Rückgabewert. Alle Arbeit steckt in run, damit die defer-Kette
// wirklich läuft: das Fehlerprotokoll wird über einen 64-KB-Puffer geschrieben, und ein
// os.Exit mitten im Ablauf ließ diesen Puffer ungeleert. Der Aufruf stand ausgerechnet in
// printSummary und griff nur dann, wenn es Fehler gab — das Protokoll brach also genau in
// dem Lauf ab, für den man es braucht.
func main() {
	os.Exit(run())
}

func run() int {
	flag.Parse()

	mysqlDSN, pgDSN, err := parseEnv()
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: %v", err)
		return exitAbgebrochen
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	el, err := newErrLogger()
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: Fehlerprotokoll konnte nicht geöffnet werden: %v", err)
		return exitAbgebrochen
	}
	defer el.close()

	mysqlDB, err := connectMySQL(ctx, mysqlDSN)
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: %v", err)
		return exitAbgebrochen
	}
	defer func() { _ = mysqlDB.Close() }() //nolint:errcheck

	log.Println("Lese Quelldaten aus MySQL …")
	titles, err := readMySQLTitles(mysqlDB)
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: mysql read: %v", err)
		return exitAbgebrochen
	}
	// #nosec G706
	log.Printf("Gelesen: %d Titel aus MySQL", len(titles))

	if *dryRun {
		doDryRun(titles, el)
		return exitOK
	}

	pgPool, err := connectPostgres(ctx, pgDSN)
	if err != nil {
		// #nosec G706
		log.Printf("FEHLER: %v", err)
		return exitAbgebrochen
	}
	defer pgPool.Close()

	bericht := fuehreUebernahmeDurch(ctx, pgPool, titles, el)
	bericht.Warnungen, bericht.Fehler = el.warnings, el.failures
	printSummary(bericht)
	return bericht.exitCode()
}

// abschlussbericht ist die vollständige Bilanz eines Laufs — einschließlich dessen, was
// NICHT geklappt hat. Behauptung (was der Code gezählt hat) und Befund (was in der
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

func (b abschlussbericht) exitCode() int {
	switch {
	case b.Abbruch != nil:
		return exitAbgebrochen
	case b.Uebersprungen > 0 || b.Fehler > 0 || !b.AbgleichOK:
		return exitUnvollstaendig
	default:
		return exitOK
	}
}

func parseEnv() (mysqlDSN, pgDSN string, err error) {
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

func connectMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	log.Println("Verbinde mit MySQL …")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close() //nolint:errcheck
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	log.Println("MySQL Verbindung OK")
	return db, nil
}

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
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

// doDryRun prüft die Quelldaten, ohne zu schreiben — inklusive der Freitextlängen, denn
// überlange Felder sind neben doppelten ISBNs der zweite Grund, aus dem eine Übernahme
// Datensätze abwertet. Wer den Trockenlauf macht, will beides vorher wissen.
func doDryRun(titles []mysqlMedium, el *errLogger) {
	log.Println("DRY-RUN: Validiere Datensätze ohne Schreibzugriff auf PostgreSQL …")
	seenISBNs := make(map[string]int)
	for _, m := range titles {
		klaerISBN(m, seenISBNs, el)
		kuerzeFelder(el, m)
	}
	// #nosec G706
	log.Printf("DRY-RUN abgeschlossen: %d Anmerkungen → %s", el.warnings, errorLogPath)
}

// fuehreUebernahmeDurch klammert den eigentlichen Import zwischen zwei Zählungen des
// tatsächlichen Bestands. Der Abgleich am Ende ist die Notbremse gegen genau die
// Fehlerklasse, die diese Datei nötig gemacht hat: eine Zahl im Abschlussbericht, die
// niemand je gegen die Datenbank gehalten hat.
func fuehreUebernahmeDurch(
	ctx context.Context, pgPool *pgxpool.Pool, titles []mysqlMedium, el *errLogger,
) abschlussbericht {
	bericht := abschlussbericht{QuellTitel: len(titles)}

	vorherTitel, vorherExemplare, err := zaehleBestand(ctx, pgPool)
	if err != nil {
		bericht.Abbruch = fmt.Errorf("konnte den Bestand vor der Übernahme nicht zählen: %w", err)
		return bericht
	}

	doImport(ctx, pgPool, titles, el, &bericht)

	nachherTitel, nachherExemplare, err := zaehleBestand(ctx, pgPool)
	if err != nil {
		// Der Import selbst ist gelaufen; nur der Nachweis fehlt. Das ist kein Abbruch,
		// aber auch kein „alles gut" — deshalb bleibt AbgleichOK false.
		// #nosec G706
		log.Printf("WARNUNG: Bestand nach der Übernahme nicht zählbar, Abgleich entfällt: %v", err)
		return bericht
	}
	bericht.IstTitel = nachherTitel - vorherTitel
	bericht.IstExemplare = nachherExemplare - vorherExemplare
	bericht.AbgleichOK = bericht.IstTitel == bericht.Titel && bericht.IstExemplare == bericht.Exemplare
	return bericht
}

// zaehleBestand liest den tatsächlichen Zeilenstand.
//
// Voraussetzung des Abgleichs: während der Übernahme schreibt niemand sonst in die
// Datenbank. Für ein Werkzeug, das einmal pro Schuljahr von Hand gestartet wird, ist das
// die richtige Annahme.
func zaehleBestand(ctx context.Context, pool *pgxpool.Pool) (titel, exemplare int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM buecher_titel),
		       (SELECT count(*) FROM buecher_exemplare)
	`).Scan(&titel, &exemplare)
	return titel, exemplare, err
}

// doImport schreibt die Titel batchweise. Meldet ein Batch einen Fehler, ist die
// Übernahme zu Ende: der Fehler betrifft dann nicht mehr einen einzelnen Datensatz,
// sondern die Verbindung oder die Transaktion selbst. Weiterzulaufen hieße, Tausende
// Zeilen „übersprungen" zu protokollieren, die in Wahrheit nie versucht wurden.
func doImport(
	ctx context.Context, pgPool *pgxpool.Pool, titles []mysqlMedium, el *errLogger, bericht *abschlussbericht,
) {
	barcodeSeq, err := highestBarcodeSeq(ctx, pgPool)
	if err != nil {
		bericht.Abbruch = fmt.Errorf("barcode seq lookup: %w", err)
		return
	}
	bericht.LetzterBarcode = barcodeSeq
	// #nosec G706
	log.Printf("Höchster vorhandener Barcode: B-%05d → Nächster: B-%05d", barcodeSeq, barcodeSeq+1)

	seenISBNs := make(map[string]int)
	totalBatches := int(math.Ceil(float64(len(titles)) / float64(*batchSize)))

	for i := 0; i < len(titles); i += *batchSize {
		end := min(i+*batchSize, len(titles))
		batch := titles[i:end]
		batchNum := (i / *batchSize) + 1

		// #nosec G706
		log.Printf("Batch %d/%d: importiere %d Titel …", batchNum, totalBatches, len(batch))
		res, err := insertBatch(ctx, pgPool, batch, seenISBNs, el, &barcodeSeq)
		bericht.LetzterBarcode = barcodeSeq
		if err != nil {
			bericht.Abbruch = fmt.Errorf("abgebrochen in Batch %d/%d (Quellzeilen ab mysql_id=%d): %w",
				batchNum, totalBatches, batch[0].ID, err)
			return
		}
		bericht.Titel += res.Titel
		bericht.Exemplare += res.Exemplare
		bericht.Uebersprungen += res.Uebersprungen
		// #nosec G706
		log.Printf("  → %d Titel, %d Exemplare eingetragen, %d übersprungen",
			res.Titel, res.Exemplare, res.Uebersprungen)
	}
}

func printSummary(b abschlussbericht) {
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
	log.Printf("  Warnungen (übernommen):   %d → %s", b.Warnungen, errorLogPath)
	// #nosec G706
	log.Printf("  Fehler (NICHT übernommen):%d → %s", b.Fehler, errorLogPath)
	// #nosec G706
	log.Printf("  Letzter Barcode:          B-%05d", b.LetzterBarcode)
	// #nosec G706
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	printAbgleich(b)

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
			"Details je Datensatz: grep FEHLER %s", b.Uebersprungen, errorLogPath)
	}
	if b.Warnungen > 0 {
		// #nosec G706
		log.Printf("ℹ  %d Datensätze wurden abgewertet übernommen (ISBN verworfen oder Feld gekürzt): "+
			"grep WARNUNG %s", b.Warnungen, errorLogPath)
	}
}

// printAbgleich meldet das Ergebnis der Gegenprobe. Eine Abweichung heißt: der Bericht
// oben ist falsch — und zwar zugunsten des Werkzeugs. Das muss lauter stehen als die
// Zahlen selbst.
func printAbgleich(b abschlussbericht) {
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
