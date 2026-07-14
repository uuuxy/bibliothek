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
	"flag"
	"log/slog"
	"os"
	"time"

	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	xmlFile := flag.String("file", "", "Pfad zur Katalogisat-XML-Datei")
	dbConn := flag.String("db", os.Getenv("DATABASE_URL"), "Datenbank-URL")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if *xmlFile == "" {
		slog.Error("Bitte XML-Datei mit -file angeben")
		os.Exit(1)
	}

	// Der Import läuft als EIN gepipelineter Batch in einer Transaktion —
	// gegen eine nicht-lokale DB braucht das bei ~15.000 Titeln Minuten, nicht Sekunden.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dbConn)
	if err != nil {
		slog.Error("Datenbankverbindung fehlgeschlagen", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	file, err := os.Open(*xmlFile)
	if err != nil {
		slog.Error("Konnte XML-Datei nicht öffnen", "error", err)
		os.Exit(1)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck

	importSvc := service.NewImportService(repository.NewBookRepository(pool), pool)
	count, err := importSvc.ParseLitteraXML(ctx, file)
	if err != nil {
		slog.Error("Import fehlgeschlagen", "error", err)
		os.Exit(1)
	}

	slog.Info("Import abgeschlossen", "verarbeitete_titel", count)
}
