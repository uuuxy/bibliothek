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
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/charmap"
)

// Feld repräsentiert ein einzelnes XML-Feld aus LITTERA mit MAB-Code und Wert.
type Feld struct {
	MAB   string `xml:"MAB,attr"`
	Value string `xml:",chardata"`
}

// Katalogisat bündelt die Felder für einen Datensatz (ein Buch/Medium).
type Katalogisat struct {
	Felder []Feld `xml:"Feld"`
}

// extractValue sucht das erste Feld mit dem passenden MAB-Code und liefert dessen Wert (getrimmt).
func extractValue(felder []Feld, mabCode string) string {
	for _, f := range felder {
		// Whitespaces trimmen, da LITTERA "540 " oder "310 " ausgeben kann
		if strings.TrimSpace(f.MAB) == mabCode {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}

// cleanTitle entfernt Störzeichen wie das logische NOT (¬), das LITTERA oft zur Sortierung verwendet.
func cleanTitle(titel string) string {
	// LITTERA nutzt ¬ für Sortier-Ignorierungen (z.B. ¬Die¬ Republik)
	return strings.ReplaceAll(titel, "¬", "")
}

// parseYear extrahiert eine 4-stellige Jahreszahl aus einem beliebigen String.
func parseYear(y string) int {
	for i := 0; i <= len(y)-4; i++ {
		if y[i] >= '0' && y[i] <= '9' && y[i+1] >= '0' && y[i+1] <= '9' && y[i+2] >= '0' && y[i+2] <= '9' && y[i+3] >= '0' && y[i+3] <= '9' {
			var year int
			fmt.Sscanf(y[i:i+4], "%d", &year)
			return year
		}
	}
	return 0
}

// extractAllValues findet alle Werte für einen bestimmten MAB-Code (z.B. Schlagworte).
func extractAllValues(felder []Feld, mabCode string) []string {
	var vals []string
	for _, f := range felder {
		if strings.TrimSpace(f.MAB) == mabCode {
			val := strings.TrimSpace(f.Value)
			if val != "" {
				vals = append(vals, val)
			}
		}
	}
	return vals
}

// toUTF8 wandelt Strings (z.B. Windows-1252 codiert) sicher in UTF-8 um.
// Ungültige Zeichenfolgen werden entfernt, um DB-Fehler zu vermeiden.
func toUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	dec := charmap.Windows1252.NewDecoder()
	out, err := dec.String(s)
	if err == nil {
		return out
	}
	// Fallback: Kaputte Zeichen durch Entfernen ignorieren
	return strings.Map(func(r rune) rune {
		if r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
}

// limitString schneidet Strings ab, berücksichtigt dabei aber Multibyte-Runen,
// sodass keine "invalid byte sequence for encoding UTF8" entsteht.
func limitString(s string, limit int) string {
	runes := []rune(s)
	if len(runes) > limit {
		return string(runes[:limit-3]) + "..."
	}
	return s
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
	decoder.CharsetReader = charset.NewReaderLabel
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
				titel := cleanTitle(toUTF8(extractValue(k.Felder, "310")))
				isbn := toUTF8(extractValue(k.Felder, "540"))
				untertitel := toUTF8(extractValue(k.Felder, "335"))
				autor := toUTF8(extractValue(k.Felder, "100"))
				verlag := toUTF8(extractValue(k.Felder, "412"))
				erscheinungsjahr := parseYear(extractValue(k.Felder, "425"))
				beschreibung := toUTF8(extractValue(k.Felder, "750f"))
				signatur := toUTF8(extractValue(k.Felder, "700"))
				standort := toUTF8(extractValue(k.Felder, "108a"))

				subjects := extractAllValues(k.Felder, "710")
				var subject string
				if len(subjects) > 0 {
					for i, s := range subjects {
						subjects[i] = toUTF8(s)
					}
					subject = limitString(strings.Join(subjects, ", "), 100)
				}

				if titel != "" {
					query := `
						UPDATE buecher_titel 
						SET 
							isbn = COALESCE(NULLIF(isbn, ''), NULLIF($1, '')),
							untertitel = COALESCE(NULLIF(untertitel, ''), NULLIF($3, '')),
							autor = COALESCE(NULLIF(autor, ''), NULLIF($4, '')),
							verlag = COALESCE(NULLIF(verlag, ''), NULLIF($5, '')),
							erscheinungsjahr = COALESCE(NULLIF(erscheinungsjahr, 0), $6),
							beschreibung = COALESCE(NULLIF(beschreibung, ''), NULLIF($7, '')),
							subject = COALESCE(NULLIF(subject, ''), NULLIF($8, '')),
							erweiterte_eigenschaften = 
								CASE 
									WHEN $9::text <> '' AND $10::text <> '' THEN
										jsonb_set(jsonb_set(erweiterte_eigenschaften, '{signatur}', to_jsonb($9::text)), '{standort}', to_jsonb($10::text))
									WHEN $9::text <> '' THEN
										jsonb_set(erweiterte_eigenschaften, '{signatur}', to_jsonb($9::text))
									WHEN $10::text <> '' THEN
										jsonb_set(erweiterte_eigenschaften, '{standort}', to_jsonb($10::text))
									ELSE erweiterte_eigenschaften
								END
						WHERE titel ILIKE $2`

					res, err := pool.Exec(context.Background(), query,
						isbn, titel, untertitel, autor, verlag, erscheinungsjahr,
						beschreibung, subject, signatur, standort,
					)

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
