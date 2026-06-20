package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbUrl := "postgres://peterflasch@127.0.0.1:5432/bibliothek?sslmode=disable"

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}
	defer pool.Close()

	query := `
		SELECT 
			bt.titel, 
			coalesce(bt.autor, ''), 
			coalesce(bt.verlag, ''), 
			coalesce(bt.isbn, ''), 
			coalesce(bt.erscheinungsjahr, 0), 
			coalesce(bt.subject, ''), 
			coalesce(be.barcode_id, ''),
			coalesce(be.zustand_notiz, '')
		FROM buecher_titel bt
		LEFT JOIN buecher_exemplare be ON bt.id = be.titel_id AND be.ist_ausgesondert = false
		ORDER BY bt.titel, be.barcode_id;
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	file, err := os.Create("/Users/peterflasch/Desktop/Bibliothek_Bestand_Export.csv")
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	_, _ = file.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(file)
	writer.Comma = ';'

	if err := writer.Write([]string{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Barcode", "Zustand"}); err != nil {
		log.Fatalf("Failed to write headers: %v", err)
	}

	count := 0
	for rows.Next() {
		var titel, autor, verlag, isbn, subject, barcode, zustand string
		var jahr int

		if err := rows.Scan(&titel, &autor, &verlag, &isbn, &jahr, &subject, &barcode, &zustand); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		jahrStr := ""
		if jahr > 0 {
			jahrStr = strconv.Itoa(jahr)
		}

		if isbn != "" {
			isbn = "'" + isbn
		}

		if err := writer.Write([]string{titel, autor, verlag, isbn, jahrStr, subject, barcode, zustand}); err != nil {
			log.Fatalf("Failed to write row: %v", err)
		}
		count++
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Fatalf("Error flushing writer: %v", err)
	}

	fmt.Printf("Successfully exported %d rows to /Users/peterflasch/Desktop/Bibliothek_Bestand_Export.csv\n", count)
}
