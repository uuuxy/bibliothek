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
	dbConnStr := flag.String("db", os.Getenv("DATABASE_URL"), "PostgreSQL Connection String (Default: $DATABASE_URL)")
	separator := flag.String("sep", ";", "Trennzeichen für die CSV (z.B. ',' oder ';')")
	colTitle := flag.Int("col-title", 0, "Index der Spalte mit dem Buchtitel (0-basiert)")
	colIsbn := flag.Int("col-isbn", 1, "Index der Spalte mit der ISBN (0-basiert)")
	hasHeader := flag.Bool("header", true, "Gibt an, ob die erste Zeile eine Kopfzeile ist und übersprungen werden soll")

	flag.Parse()

	if *dbConnStr == "" {
		log.Fatal("Kein Connection-String: setze DATABASE_URL oder nutze das -db Flag")
	}

	// 2. Datenbankverbindung herstellen
	db := oeffneDatenbank(*dbConnStr)
	defer db.Close()

	// 3. CSV-Datei öffnen + Reader konfigurieren (inkl. Header-Skip)
	file, reader := oeffneCSVReader(*csvFilePath, *separator, *hasHeader)
	defer file.Close()

	// 4. Pre-compile SQL Statement für maximale Performance
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
	defer stmt.Close()

	log.Println("Starte ISBN-Import... (dies kann einen Moment dauern)")

	// Metriken / Logging Counter
	var countErfolgreich int
	var countNichtGefunden int
	var countFehler int

	// 5. Zeile für Zeile verarbeiten
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

		switch verarbeiteISBNZeile(stmt, record, *colTitle, *colIsbn) {
		case "ok":
			countErfolgreich++
		case "notfound":
			countNichtGefunden++
		case "fehler":
			countFehler++
		}
	}

	// 6. Abschluss-Bericht
	fmt.Println("\n========================================")
	fmt.Println("         IMPORT ABGESCHLOSSEN           ")
	fmt.Println("========================================")
	fmt.Printf("Erfolgreich gemerged          : %d\n", countErfolgreich)
	fmt.Printf("Nicht gefunden / bereits voll : %d\n", countNichtGefunden)
	fmt.Printf("Fehlerhafte CSV-Zeilen        : %d\n", countFehler)
	fmt.Println("========================================")
	fmt.Println("HINWEIS: Es wurden gemäß Vorgabe KEINE neuen Datensätze per INSERT angelegt.")
}

// oeffneDatenbank öffnet die DB-Verbindung und prüft sie (log.Fatal bei Fehler).
func oeffneDatenbank(connStr string) *sql.DB {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Keine Verbindung zur DB. Prüfe den Connection-String: %v", err)
	}
	log.Println("Erfolgreich mit der Datenbank verbunden.")
	return db
}

// oeffneCSVReader öffnet die CSV-Datei, konfiguriert den Reader und überspringt bei Bedarf
// die Kopfzeile. Der Aufrufer muss die zurückgegebene Datei schließen.
func oeffneCSVReader(path, separator string, hasHeader bool) (*os.File, *csv.Reader) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der CSV-Datei '%s': %v", path, err)
	}

	reader := csv.NewReader(file)
	if len(separator) > 0 {
		reader.Comma = rune(separator[0])
	}
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Erlaubt Zeilen mit unterschiedlicher Spaltenanzahl

	if hasHeader {
		if _, err := reader.Read(); err != nil && err != io.EOF {
			log.Fatalf("Fehler beim Lesen des CSV-Headers: %v", err)
		}
	}
	return file, reader
}

// verarbeiteISBNZeile führt das Titel→ISBN-Merge-Update für eine CSV-Zeile aus und liefert
// den Ausgang: "ok", "notfound", "fehler" oder "" (leere Werte, keine Aktion).
func verarbeiteISBNZeile(stmt *sql.Stmt, record []string, colTitle, colIsbn int) string {
	// Sicherheitsprüfung: Sind die angegebenen Spaltenindizes überhaupt in dieser Zeile vorhanden?
	maxRequiredIndex := colTitle
	if colIsbn > maxRequiredIndex {
		maxRequiredIndex = colIsbn
	}
	if len(record) <= maxRequiredIndex {
		return "fehler"
	}

	titel := strings.TrimSpace(record[colTitle])
	isbn := strings.TrimSpace(record[colIsbn])
	if titel == "" || isbn == "" {
		return "" // Keine Aktion bei leeren Werten
	}

	var updatedID string
	err := stmt.QueryRowContext(context.Background(), isbn, titel).Scan(&updatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			// sql.ErrNoRows bedeutet: UPDATE hat 0 Zeilen verändert (Titel unbekannt oder
			// bereits mit ISBN belegt).
			return "notfound"
		}
		log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
		return "fehler"
	}
	return "ok"
}
