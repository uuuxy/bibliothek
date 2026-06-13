package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Feld struct {
	MAB   string `xml:"MAB,attr"`
	Value string `xml:",chardata"`
}

type Katalogisat struct {
	Felder []Feld `xml:"Feld"`
}

func extractValue(felder []Feld, mabCode string) string {
	for _, f := range felder {
		// Whitespaces trimmen, da LITTERA "540 " oder "310 " ausgibt
		if strings.TrimSpace(f.MAB) == mabCode {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}

func cleanTitle(titel string) string {
	// LITTERA nutzt ¬ für Sortier-Ignorierungen (z.B. ¬Die¬ Republik)
	return strings.ReplaceAll(titel, "¬", "")
}

func main() {
	xmlFile := flag.String("file", "", "Pfad zur XML-Datei")
	dbConn := flag.String("db", os.Getenv("DATABASE_URL"), "Datenbank-URL")
	flag.Parse()

	// 0. Setup strukturiertes JSON-Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if *xmlFile == "" {
		slog.Error("Bitte XML-Datei mit -file angeben")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	defer file.Close()

	decoder := xml.NewDecoder(file)
	var updatedCount int
	var processedCount int

	for {
		t, err := decoder.Token()
		if err != nil {
			break // EOF oder Fehler
		}

		switch se := t.(type) {
		case xml.StartElement:
			if strings.EqualFold(se.Name.Local, "katalogisat") {
				var k Katalogisat
				if err := decoder.DecodeElement(&k, &se); err != nil {
					slog.Warn("Fehler beim Dekodieren", "error", err)
					continue
				}

				processedCount++
				titel := cleanTitle(extractValue(k.Felder, "310"))
				isbn := extractValue(k.Felder, "540")

				if titel != "" && isbn != "" {
					// Update nur auf buecher_titel (in schema.sql heißt die Tabelle so, nicht buecher)
					res, err := pool.Exec(context.Background(),
						`UPDATE buecher_titel SET isbn = $1 WHERE titel ILIKE $2 AND (isbn IS NULL OR TRIM(isbn) = '')`,
						isbn, titel)
					if err == nil {
						if res.RowsAffected() > 0 {
							updatedCount += int(res.RowsAffected())
						}
					} else {
						slog.Error("DB Update Fehler", "titel", titel, "error", err)
					}
				}

				if processedCount%1000 == 0 {
					slog.Info("Zwischenstand", "verarbeitet", processedCount, "aktualisiert", updatedCount)
				}
			}
		}
	}

	fmt.Printf("Import abgeschlossen. %d Katalogisate verarbeitet, %d ISBNs wurden aktualisiert.\n", processedCount, updatedCount)
}
