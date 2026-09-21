// littera-import importiert ein Littera-MAB2-Katalogisat (XML) über denselben
// Service-Pfad wie der API-Endpunkt POST /api/import/littera: Titel werden über
// ISBN oder Titel gegen den Bestand gematcht (keine Dubletten bei Re-Imports),
// Signaturen landen in der echten Spalte buecher_titel.signatur, LMF-Bestand
// wird per "LMF-"-Präfix geflaggt.
//
// Aufruf: go run ./cmd/littera-import -file katalogisat.xml [-db postgres://…]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "littera-import: %v\n", err)
		os.Exit(1)
	}
}

// run ist der ganze Ablauf, mit Argumenten, Umgebung und Ausgabe als Parameter — so
// prüft ihn der Test ohne echte Umgebungsvariablen (wie encrypt-backup). Bis zum
// 21.09.2026 stand alles in main und hatte keinen Test (OFFEN.md 5.10); das Werkzeug
// läuft einmal gegen den echten Bestand.
//
// Die Datei wird VOR der Datenbankverbindung geöffnet: Ein Tippfehler im Pfad soll
// scheitern, bevor irgendetwas eine Verbindung aufbaut.
func run(args []string, getenv func(string) string, stdout io.Writer) error {
	flags := flag.NewFlagSet("littera-import", flag.ContinueOnError)
	flags.SetOutput(stdout)
	xmlFile := flags.String("file", "", "Pfad zur Katalogisat-XML-Datei")
	dbConn := flags.String("db", getenv("DATABASE_URL"), "Datenbank-URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *xmlFile == "" {
		return errors.New("bitte XML-Datei mit -file angeben")
	}
	if *dbConn == "" {
		return errors.New("keine Datenbank: -db angeben oder DATABASE_URL setzen")
	}

	file, err := os.Open(*xmlFile) // #nosec G304 -- Pfad kommt vom Aufrufer auf der Kommandozeile
	if err != nil {
		return fmt.Errorf("XML-Datei konnte nicht geöffnet werden: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Der Import läuft als EIN gepipelineter Batch in einer Transaktion —
	// gegen eine nicht-lokale DB braucht das bei ~15.000 Titeln Minuten, nicht Sekunden.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dbConn)
	if err != nil {
		return fmt.Errorf("datenbankverbindung fehlgeschlagen: %w", err)
	}
	defer pool.Close()

	importSvc := service.NewImportService(repository.NewBookRepository(pool), pool)
	count, err := importSvc.ParseLitteraXML(ctx, file)
	if err != nil {
		return fmt.Errorf("import fehlgeschlagen: %w", err)
	}

	logger.Info("Import abgeschlossen", "verarbeitete_titel", count)
	return nil
}
