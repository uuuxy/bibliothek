package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func deutschSignaturID(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var id int
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM signatures WHERE name = 'Deutsch'`).Scan(&id); err != nil {
		t.Fatalf("Deutsch-Signatur-ID: %v", err)
	}
	return id
}

func erfassungsZahl(t *testing.T, pool *pgxpool.Pool, sessionID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM inventur_erfassungen WHERE session_id = $1`, sessionID).Scan(&n); err != nil {
		t.Fatalf("Erfassungszahl: %v", err)
	}
	return n
}

func ausgesonderteZahl(t *testing.T, pool *pgxpool.Pool, exemplarIDs []string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM buecher_exemplare WHERE id = ANY($1) AND ist_ausgesondert = true`,
		exemplarIDs).Scan(&n); err != nil {
		t.Fatalf("Ausgesonderte-Zahl: %v", err)
	}
	return n
}

func leiheAus(t *testing.T, pool *pgxpool.Pool, exemplarID string) {
	t.Helper()
	ctx := context.Background()

	var schuelerID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('S-INV', 'Test', 'Schueler', '7a', 2030)
		 ON CONFLICT (barcode_id) DO UPDATE SET vorname = EXCLUDED.vorname
		 RETURNING id`).Scan(&schuelerID); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		 VALUES ($1, $2, CURRENT_DATE + 14)`, exemplarID, schuelerID); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
}
