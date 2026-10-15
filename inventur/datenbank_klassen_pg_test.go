package inventur

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestKlassennamen_RegulaererAusdruck(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL nicht gesetzt — DB-Integrationstest übersprungen")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("pool.Begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	_, err = tx.Exec(ctx, `
		CREATE TEMP TABLE class_books (
			class_name TEXT NOT NULL,
			book_id UUID NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("tx.Exec: %v", err)
	}

	titelID := "123e4567-e89b-12d3-a456-426614174000"
	bookID2 := "123e4567-e89b-12d3-a456-426614174001"

	_, err = tx.Exec(ctx, `
		INSERT INTO class_books (class_name, book_id) VALUES
		('5 A', $1),
		('5A', $1),
		(' 6 B ', $2),
		('7', $1),
		('07', $1),
		('8', $2),
		('10A', $1)
	`, titelID, bookID2)
	if err != nil {
		t.Fatalf("tx.Exec: %v", err)
	}

	// Genau die Abfragen der Normalisierung aller Klassen.

	// 1. Delete dup spaces
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE class_name LIKE '% %'
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = REPLACE(cb1.class_name, ' ', '')
			AND cb2.book_id = cb1.book_id
		)
	`)
	if err != nil {
		t.Fatalf("tx.Exec: %v", err)
	}

	// 2. Remove spaces
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = REPLACE(class_name, ' ', '')
		WHERE class_name LIKE '% %'
	`)
	if err != nil {
		t.Fatalf("tx.Exec: %v", err)
	}

	// 3. Delete dup leading zero
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE (class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$')
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = '0' || cb1.class_name
			AND cb2.book_id = cb1.book_id
		)
	`)
	if err != nil {
		t.Fatalf("tx.Exec: %v", err)
	}

	// 4. Update leading zero
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = '0' || class_name
		WHERE class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$'
	`)
	if err != nil {
		t.Fatalf("tx.Exec: %v", err)
	}

	// Prüfen.
	zeilen, err := tx.Query(ctx, "SELECT class_name, book_id FROM class_books ORDER BY class_name, book_id")
	if err != nil {
		t.Fatalf("tx.Query: %v", err)
	}
	defer zeilen.Close()

	type ergebnis struct {
		ClassName string
		BookID    string
	}
	var ergebnisse []ergebnis
	for zeilen.Next() {
		var r ergebnis
		err := zeilen.Scan(&r.ClassName, &r.BookID)
		if err != nil {
			t.Fatalf("zeilen.Scan: %v", err)
		}
		ergebnisse = append(ergebnisse, r)
	}
	if err := zeilen.Err(); err != nil {
		t.Fatalf("zeilen.Err: %v", err)
	}

	expected := []ergebnis{
		{"05A", titelID},
		{"06B", bookID2},
		{"07", titelID},
		{"08", bookID2},
		{"10A", titelID},
	}

	if diff := cmp.Diff(expected, ergebnisse); diff != "" {
		t.Errorf("ergebnisse weicht ab (-erwartet +ist):\n%s", diff)
	}
}
