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

	// 4. Die Anweisungen einmal vorbereiten. Gesucht wird der Titel gleich ohne Rücksicht auf
	// Groß- und Kleinschreibung, nicht per ILIKE: Darin wirkten % und _ eines Titels („100% Mathe“)
	// als Platzhalter und träfen fremde Titel. Der Titel der Datei geht in die Form, in der die
	// Datenbank Titel speichert. Gesetzt wird nur bei genau einem Titel ohne ISBN.
	suche, err := verbindung.PrepareContext(context.Background(), `
		SELECT id FROM buecher_titel
		WHERE lower(titel) = lower(titeltext_normalform($1))
		  AND (isbn IS NULL OR trim(isbn) = '')
		LIMIT 2`)
	if err != nil {
		log.Fatalf("Fehler beim Vorbereiten der Suche: %v", err)
	}
	defer suche.Close()
	setze, err := verbindung.PrepareContext(context.Background(), `
		UPDATE buecher_titel
		SET isbn = $1, aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2 AND (isbn IS NULL OR trim(isbn) = '')
		RETURNING id`)
	if err != nil {
		log.Fatalf("Fehler beim Vorbereiten des Updates: %v", err)
	}
	defer setze.Close()

	log.Println("Starte ISBN-Import... (dies kann einen Moment dauern)")

	// Zähler für den Bericht.
	var erfolgreich int
	var nichtGefunden int
	var mehrdeutig int
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

		switch verarbeiteISBNZeile(suche, setze, zeile, *spalteTitel, *spalteISBN) {
		case "ok":
			erfolgreich++
		case "notfound":
			nichtGefunden++
		case "mehrdeutig":
			mehrdeutig++
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
	fmt.Printf("Mehrere Titel, nichts gesetzt : %d\n", mehrdeutig)
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
// Ausgang: "ok", "notfound", "mehrdeutig", "fehler" oder "" (leere Werte, nichts zu tun).
// Mehrdeutig heißt: Mehrere Titel ohne ISBN tragen diesen Namen, etwa zwei Ausgaben eines
// Arbeitshefts; welcher die ISBN bekommt, entscheidet die Datei nicht, also keiner.
func verarbeiteISBNZeile(suche, setze *sql.Stmt, zeile []string, spalteTitel, spalteISBN int) string {
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

	kennung, ausgang := einzigerTitelOhneISBN(suche, titel)
	if ausgang != "" {
		return ausgang
	}
	err := setze.QueryRowContext(context.Background(), isbn, kennung).Scan(&kennung)
	if err == sql.ErrNoRows {
		return "notfound" // inzwischen von anderer Seite gesetzt
	}
	if err != nil {
		log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
		return "fehler"
	}
	return "ok"
}

// einzigerTitelOhneISBN liefert die Kennung des einen Titels ohne ISBN mit diesem Namen; sonst
// ist ausgang "notfound", "mehrdeutig" oder "fehler".
func einzigerTitelOhneISBN(suche *sql.Stmt, titel string) (kennung, ausgang string) {
	zeilen, err := suche.QueryContext(context.Background(), titel)
	if err != nil {
		log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
		return "", "fehler"
	}
	defer zeilen.Close()
	var kennungen []string
	for zeilen.Next() {
		var id string
		if err := zeilen.Scan(&id); err != nil {
			log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
			return "", "fehler"
		}
		kennungen = append(kennungen, id)
	}
	if err := zeilen.Err(); err != nil {
		log.Printf("Kritischer Datenbankfehler bei Titel '%s': %v", titel, err)
		return "", "fehler"
	}
	switch len(kennungen) {
	case 0:
		return "", "notfound"
	case 1:
		return kennungen[0], ""
	}
	log.Printf("Mehrere Titel ohne ISBN heißen '%s' – nichts gesetzt", titel)
	return "", "mehrdeutig"
}
