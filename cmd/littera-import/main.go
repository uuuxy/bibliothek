// Command littera-import übernimmt ein Katalogisat aus Littera (MAB2 als XML) über denselben Weg
// im Dienst wie die Tür POST /api/import/littera: Titel werden über ISBN oder Titel dem Bestand
// zugeordnet (keine Dubletten bei wiederholtem Import), Signaturen landen in der Spalte
// buecher_titel.signatur, und den Bestand der Lernmittelfreiheit erkennt der Import an der
// Kennung LMF und setzt ist_lernmittel.
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
	if err := fuehreAus(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "littera-import: %v\n", err)
		os.Exit(1)
	}
}

// fuehreAus ist der ganze Ablauf, mit Argumenten, Umgebung und Ausgabe als Parameter; so prüft
// ihn der Test ohne echte Umgebungsvariablen (wie encrypt-backup). Das Werkzeug läuft einmal
// gegen den echten Bestand.
//
// Die Datei wird vor der Verbindung zur Datenbank geöffnet: Ein Tippfehler im Pfad soll
// scheitern, bevor irgendetwas eine Verbindung aufbaut.
func fuehreAus(argumente []string, umgebung func(string) string, stdout io.Writer) error {
	schalter := flag.NewFlagSet("littera-import", flag.ContinueOnError)
	schalter.SetOutput(stdout)
	xmlDatei := schalter.String("file", "", "Pfad zur Katalogisat-XML-Datei")
	datenbank := schalter.String("db", umgebung("DATABASE_URL"), "Datenbank-URL")
	if err := schalter.Parse(argumente); err != nil {
		return err
	}
	if *xmlDatei == "" {
		return errors.New("bitte XML-Datei mit -file angeben")
	}
	if *datenbank == "" {
		return errors.New("keine Datenbank: -db angeben oder DATABASE_URL setzen")
	}

	datei, err := os.Open(*xmlDatei) // #nosec G304 -- Pfad kommt vom Aufrufer auf der Kommandozeile
	if err != nil {
		return fmt.Errorf("XML-Datei konnte nicht geöffnet werden: %w", err)
	}
	defer func() { _ = datei.Close() }() //nolint:errcheck

	logAusgabe := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logAusgabe)

	// Der Import läuft als ein Batch in einer Transaktion; gegen eine entfernte Datenbank braucht
	// das bei etwa 15.000 Titeln Minuten, nicht Sekunden.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, *datenbank)
	if err != nil {
		return fmt.Errorf("datenbankverbindung fehlgeschlagen: %w", err)
	}
	defer pool.Close()

	importDienst := service.NewImportDienst(repository.NewKatalogAbfragen(pool), pool)
	anzahl, err := importDienst.ImportiereLitteraXML(ctx, datei)
	if err != nil {
		return fmt.Errorf("import fehlgeschlagen: %w", err)
	}

	logAusgabe.Info("Import abgeschlossen", "verarbeitete_titel", anzahl)
	return nil
}
