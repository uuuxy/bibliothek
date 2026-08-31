package lmf

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// sqlErkennt wertet SQLBedingung in ECHTEM Postgres aus — der einzige Weg, die beiden
// Antworten (Go und SQL) wirklich zu vergleichen. Ohne TEST_DATABASE_URL übersprungen.
func sqlErkennt(t *testing.T, titel, signatur string) bool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL nicht gesetzt — SQL-Vergleich übersprungen")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("Verbindung: %v", err)
	}
	defer func() {
		if err := conn.Close(ctx); err != nil {
			t.Logf("Verbindung schließen: %v", err)
		}
	}()

	var treffer bool
	// Die Spalten der Bedingung als Parameter untergeschoben: derselbe Ausdruck, den
	// die Aufrufer auf ihre Tabellen anwenden.
	if err := conn.QueryRow(ctx,
		`SELECT `+SQLBedingung("t.titel", "t.signatur")+
			` FROM (SELECT $1::text AS titel, $2::text AS signatur) t`,
		titel, signatur).Scan(&treffer); err != nil {
		t.Fatalf("SQLBedingung auswerten: %v", err)
	}
	return treffer
}
