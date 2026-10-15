//go:build ignore

// import_isbns trägt ISBNs aus einer CSV-Datei bei Titeln nach, die noch keine haben. Es legt
// keinen Datensatz an; ein Titel ohne Treffer bleibt, wie er ist.
//
// Aufruf: go run scripts/import_isbns.go -file isbns.csv [-db postgres://…] [-sep ";"]
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
	// 1. Einstellungen über die Schalter der Kommandozeile.
	csvPfad := flag.String("file", "isbns.csv", "Pfad zur CSV-Datei")
	datenbank := flag.String("db", os.Getenv("DATABASE_URL"), "PostgreSQL Connection String (Default: $DATABASE_URL)")
	trenner := flag.String("sep", ";", "Trennzeichen für die CSV (z.B. ',' oder ';')")
	spalteTitel := flag.Int("col-title", 0, "Index der Spalte mit dem Buchtitel (0-basiert)")
	spalteISBN := flag.Int("col-isbn", 1, "Index der Spalte mit der ISBN (0-basiert)")
	mitKopfzeile := flag.Bool("header", true, "Gibt an, ob die erste Zeile eine Kopfzeile ist und übersprungen werden soll")

	flag.Parse()

	if *datenbank == "" {
		log.Fatal("Kein Connection-String: setze DATABASE_URL oder nutze das -db Flag")
	}

	// 2. Mit der Datenbank verbinden.
	verbindung := oeffneDatenbank(*datenbank)
	defer verbindung.Close()

	// 3. Die CSV-Datei öffnen und den Leser einstellen, mit Überspringen der Kopfzeile.
	datei, leser := oeffneCSV(*csvPfad, *trenner, *mitKopfzeile)
	defer datei.Close()

	// 4. Die Anweisung einmal vorbereiten. UPDATE nur, wenn der Titel passt (ILIKE, ohne Rücksicht
	// auf Groß- und Kleinschreibung) und das Feld isbn leer oder NULL ist. Der Titel der Datei
	// geht in die Form, in der die Datenbank Titel speichert.
	abfrage := `
		UPDATE buecher_titel
		SET isbn = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE titel ILIKE titeltext_normalform($2)
		  AND (isbn IS NULL OR trim(isbn) = '')
		RETURNING id;
	`
	anweisung, err := verbindung.PrepareContext(context.Background(), abfrage)
	if err != nil {
		log.Fatalf("Fehler beim Vorbereiten des SQL-Statements: %v", err)
	}
	defer anweisung.Close()

	log.Println("Starte ISBN-Import... (dies kann einen Moment dauern)")

	// Zähler für den Bericht.
	var erfolgreich int
	var nichtGefunden int
	var fehlerhaft int

	// 5. Zeile für Zeile verarbeiten.
	for {
		zeile, err := leser.Read()
		if err == io.EOF {
			break // Ende der Datei
		}
		if err != nil {
			log.Printf("Fehler beim Lesen einer CSV-Zeile: %v", err)
			fehlerhaft++
			continue
		}

		switch verarbeiteISBNZeile(anweisung, zeile, *spalteTitel, *spalteISBN) {
		case "ok":
			erfolgreich++
		case "notfound":
			nichtGefunden++
		case "fehler":
			fehlerhaft++
		}
	}

	// 6. Bericht.
	fmt.Println("\n========================================")
	fmt.Println("         IMPORT ABGESCHLOSSEN           ")
	fmt.Println("========================================")
	fmt.Printf("Erfolgreich gemerged          : %d\n", erfolgreich)
	fmt.Printf("Nicht gefunden / bereits voll : %d\n", nichtGefunden)
	fmt.Printf("Fehlerhafte CSV-Zeilen        : %d\n", fehlerhaft)
	fmt.Println("========================================")
	fmt.Println("HINWEIS: Es wurden gemäß Vorgabe KEINE neuen Datensätze per INSERT angelegt.")
}

// oeffneDatenbank öffnet die Verbindung zur Datenbank und prüft sie (log.Fatal bei Fehler).
func oeffneDatenbank(verbindungsText string) *sql.DB {
	verbindung, err := sql.Open("pgx", verbindungsText)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der DB: %v", err)
	}
	if err := verbindung.Ping(); err != nil {
		log.Fatalf("Keine Verbindung zur DB. Prüfe den Connection-String: %v", err)
	}
	log.Println("Erfolgreich mit der Datenbank verbunden.")
	return verbindung
}

// oeffneCSV öffnet die CSV-Datei, stellt den Leser ein und überspringt bei Bedarf die
// Kopfzeile. Der Aufrufer schließt die zurückgegebene Datei.
func oeffneCSV(pfad, trenner string, mitKopfzeile bool) (*os.File, *csv.Reader) {
	datei, err := os.Open(pfad)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der CSV-Datei '%s': %v", pfad, err)
	}

	leser := csv.NewReader(datei)
	if len(trenner) > 0 {
		leser.Comma = rune(trenner[0])
	}
	leser.LazyQuotes = true
	leser.TrimLeadingSpace = true
	leser.FieldsPerRecord = -1 // Zeilen mit verschiedener Zahl von Spalten erlauben

	if mitKopfzeile {
		if _, err := leser.Read(); err != nil && err != io.EOF {
			log.Fatalf("Fehler beim Lesen des CSV-Headers: %v", err)
		}
	}
	return datei, leser
}

// verarbeiteISBNZeile trägt die ISBN einer CSV-Zeile beim passenden Titel nach und liefert den
// Ausgang: "ok", "notfound", "fehler" oder "" (leere Werte, nichts zu tun).
func verarbeiteISBNZeile(anweisung *sql.Stmt, zeile []string, spalteTitel, spalteISBN int) string {
	// Stehen die angegebenen Spalten überhaupt in dieser Zeile?
	hoechsteSpalte := spalteTitel
	if spalteISBN > hoechsteSpalte {
		hoechsteSpalte = spalteISBN
	}
	if len(zeile) <= hoechsteSpalte {
		return "fehler"
	}

	titel := strings.TrimSpace(zeile[spalteTitel])
	isbn := strings.TrimSpace(zeile[spalteISBN])
	if titel == "" || isbn == "" {
		return "" // leere Werte: nichts zu tun
	}

	var kennung string
	err := anweisung.QueryRowContext(context.Background(), isbn, titel).Scan(&kennung)
	if err != nil {
		if err == sql.ErrNoRows {
			// sql.ErrNoRows heißt: UPDATE hat 0 Zeilen geändert (Titel unbekannt oder schon mit
			// ISBN).
			return "notfound"
		}
		log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
		return "fehler"
	}
	return "ok"
}
