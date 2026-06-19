package api

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(b *testing.B) *pgxpool.Pool {
	ctx := context.Background()
	// Using the fallback PG on 5432
	dsn := "postgres://postgres:postgrespassword@localhost:5432/bibliothek?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		b.Skipf("Skipping benchmark, could not connect to database: %v", err)
	}

	// Make sure schueler and ausleihen exist
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schueler (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			barcode_id VARCHAR(100) UNIQUE NOT NULL,
			vorname VARCHAR(100) NOT NULL,
			nachname VARCHAR(100) NOT NULL,
			klasse VARCHAR(20) NOT NULL,
			geburtsdatum DATE DEFAULT NULL,
			abgaenger_jahr INTEGER NOT NULL,
			ist_gesperrt BOOLEAN NOT NULL DEFAULT false,
			lusd_id VARCHAR(64) UNIQUE,
			ist_abgaenger BOOLEAN NOT NULL DEFAULT false,
			erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS ausleihen (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			schueler_id UUID NOT NULL,
			buch_id UUID NOT NULL,
			ausgeliehen_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			rueckgabe_am TIMESTAMP WITH TIME ZONE
		);
	`)
	if err != nil {
		b.Fatalf("Failed to create tables: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE schueler CASCADE;")
	if err != nil {
		b.Fatalf("Failed to truncate schueler: %v", err)
	}
	_, err = pool.Exec(ctx, "TRUNCATE ausleihen CASCADE;")
	if err != nil {
		b.Fatalf("Failed to truncate ausleihen: %v", err)
	}

	return pool
}

func BenchmarkComputeLusdChanges(b *testing.B) {
	pool := setupTestDB(b)
	defer pool.Close()

	ctx := context.Background()

	// Create mock server
	s := &Server{
		DB: &db.Database{Pool: pool},
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Clean up previous run
		_, err := pool.Exec(ctx, "TRUNCATE schueler CASCADE; TRUNCATE ausleihen CASCADE;")
		if err != nil {
			b.Fatalf("Failed to truncate tables: %v", err)
		}

		// Insert 1000 existing students
		var insertRows [][]any
		for j := 0; j < 1000; j++ {
			barcode := fmt.Sprintf("B-OLD-%d-%d", i, j)
			lusdID := fmt.Sprintf("LUSD-%d-%d", i, j)
			insertRows = append(insertRows, []any{barcode, "OldFirst", "OldLast", "5a", 2030, lusdID, nil})
		}
		_, err = pool.CopyFrom(
			ctx,
			pgx.Identifier{"schueler"},
			[]string{"barcode_id", "vorname", "nachname", "klasse", "abgaenger_jahr", "lusd_id", "geburtsdatum"},
			pgx.CopyFromRows(insertRows),
		)
		if err != nil {
			b.Fatalf("Failed to insert initial students: %v", err)
		}

		// Create CSV records:
		// 800 students class changes (j=0..799)
		// 200 students absent (j=800..999) - Graduates
		// 200 new students (j=1000..1199)
		var records []lusdRecord
		for j := 0; j < 800; j++ {
			records = append(records, lusdRecord{
				LusdID:   fmt.Sprintf("LUSD-%d-%d", i, j),
				Vorname:  "OldFirst",
				Nachname: "OldLast",
				Klasse:   "6b", // changed from 5a
			})
		}
		for j := 1000; j < 1200; j++ {
			records = append(records, lusdRecord{
				LusdID:   fmt.Sprintf("LUSD-%d-%d", i, j),
				Vorname:  "NewFirst",
				Nachname: "NewLast",
				Klasse:   "5a",
			})
		}

		b.StartTimer()

		res, err := s.computeLusdChanges(ctx, records, true)
		if err != nil {
			b.Fatalf("computeLusdChanges failed: %v", err)
		}

		b.StopTimer()

		if res.ClassChanges != 800 || res.Graduates != 200 || res.NewStudents != 200 {
			b.Fatalf("Unexpected stats: %+v", res)
		}
	}
}
