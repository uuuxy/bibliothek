//go:build ignore

package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// 1. Konfiguration über Command-Line Flags
	csvFilePath := flag.String("file", "isbns.csv", "Pfad zur CSV-Datei")
	dbConnStr := flag.String("db", "postgres://postgres:geheim@localhost:5434/bibliothek?sslmode=disable", "PostgreSQL Connection String")
	separator := flag.String("sep", ";", "Trennzeichen für die CSV (z.B. ',' oder ';')")
	colTitle := flag.Int("col-title", 0, "Index der Spalte mit dem Buchtitel (0-basiert)")
	colIsbn := flag.Int("col-isbn", 1, "Index der Spalte mit der ISBN (0-basiert)")
	hasHeader := flag.Bool("header", true, "Gibt an, ob die erste Zeile eine Kopfzeile ist und übersprungen werden soll")

	flag.Parse()

	// 2. Datenbankverbindung herstellen
	db, err := sql.Open("pgx", *dbConnStr)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der DB: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Fehler beim Schließen der Datenbankverbindung: %v", err)
		}
	}()

	if err := db.Ping(); err != nil {
		log.Fatalf("Keine Verbindung zur DB. Prüfe den Connection-String: %v", err)
	}
	log.Println("Erfolgreich mit der Datenbank verbunden.")

	// 3. CSV-Datei öffnen
	file, err := os.Open(*csvFilePath)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der CSV-Datei '%s': %v", *csvFilePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Fehler beim Schließen der Datei: %v", err)
		}
	}()

	// 4. CSV-Reader konfigurieren
	reader := csv.NewReader(file)
	if len(*separator) > 0 {
		reader.Comma = rune((*separator)[0])
	}
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Erlaubt Zeilen mit unterschiedlicher Spaltenanzahl

	// Falls ein Header existiert, die erste Zeile verwerfen
	if *hasHeader {
		if _, err := reader.Read(); err != nil && err != io.EOF {
			log.Fatalf("Fehler beim Lesen des CSV-Headers: %v", err)
		}
	}

	// Metriken / Logging Counter
	var countErfolgreich int
	var countNichtGefunden int
	var countFehler int

	// 5. Pre-compile SQL Statement für maximale Performance
	// UPDATE nur wenn Titel (via ILIKE case-insensitive) matcht UND das ISBN-Feld leer/NULL ist.
	updateQuery := `
		UPDATE buecher_titel 
		SET isbn = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE titel ILIKE $2 
		  AND (isbn IS NULL OR trim(isbn) = '')
		RETURNING id;
	`
	stmt, err := db.PrepareContext(context.Background(), updateQuery)
	if err != nil {
		log.Fatalf("Fehler beim Vorbereiten des SQL-Statements: %v", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Printf("Fehler beim Schließen des Statements: %v", err)
		}
	}()

	log.Println("Starte ISBN-Import... (dies kann einen Moment dauern)")

	// 6. Zeile für Zeile verarbeiten
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // Ende der Datei erreicht
		}
		if err != nil {
			log.Printf("Fehler beim Lesen einer CSV-Zeile: %v", err)
			countFehler++
			continue
		}

		// Sicherheitsprüfung: Sind die angegebenen Spaltenindizes überhaupt in dieser Zeile vorhanden?
		maxRequiredIndex := *colTitle
		if *colIsbn > maxRequiredIndex {
			maxRequiredIndex = *colIsbn
		}

		if len(record) <= maxRequiredIndex {
			// Zeile hat zu wenige Spalten, um Titel und ISBN sicher auszulesen
			countFehler++
			continue
		}

		// Werte extrahieren und bereinigen
		titel := strings.TrimSpace(record[*colTitle])
		isbn := strings.TrimSpace(record[*colIsbn])

		if titel == "" || isbn == "" {
			continue // Keine Aktion bei leeren Werten
		}

		// Update ausführen
		var updatedId string
		err = stmt.QueryRowContext(context.Background(), isbn, titel).Scan(&updatedId)

		if err != nil {
			if err == sql.ErrNoRows {
				// sql.ErrNoRows bedeutet: UPDATE hat 0 Zeilen verändert.
				// Entweder existiert der Titel nicht, oder er hat bereits eine ISBN.
				countNichtGefunden++
			} else {
				log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
				countFehler++
			}
		} else {
			countErfolgreich++
		}
	}

	// 7. Abschluss-Bericht
	fmt.Println("\n========================================")
	fmt.Println("         IMPORT ABGESCHLOSSEN           ")
	fmt.Println("========================================")
	fmt.Printf("Erfolgreich gemerged          : %d\n", countErfolgreich)
	fmt.Printf("Nicht gefunden / bereits voll : %d\n", countNichtGefunden)
	fmt.Printf("Fehlerhafte CSV-Zeilen        : %d\n", countFehler)
	fmt.Println("========================================")
	fmt.Println("HINWEIS: Es wurden gemäß Vorgabe KEINE neuen Datensätze per INSERT angelegt.")
}
