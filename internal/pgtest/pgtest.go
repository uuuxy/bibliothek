// Package pgtest ist die gemeinsame Infrastruktur der PG-Integrationstests
// (gated auf TEST_DATABASE_URL). Bis zum 31.08.2026 lag dieser Baustein als echte
// Kopie in fünf Paketen (api, repository, db, cmd/migrate, internal/littera) —
// Befund im Register, gemeldet auch von SonarQube.
//
// Wie internal/smtptest ist das ein normales Paket, das nur von Tests importiert
// wird: Testdateien (_test.go) lassen sich nicht paketübergreifend importieren.
// Die deadcode-Baseline führt die Funktionen deshalb mit Begründung.
//
// Warum echtes Postgres und nicht pgxmock: Die geprüfte Logik lebt in SQL
// (Constraints, partielle Unique-Indizes, Filter, Trigger) — pgxmock würde nur
// nachgespielte Antworten prüfen, nicht die Korrektheit (siehe die Paket-Köpfe
// der jeweiligen *_pg_test.go).
package pgtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvVar gated die Integrationstests: lokal ohne die Variable werden sie
// übersprungen; in CI setzt der Workflow sie auf den Postgres-Service-Container
// (und db/pgtest_guard_test.go stellt sicher, dass CI sie wirklich setzt).
const EnvVar = "TEST_DATABASE_URL"

// lockKey serialisiert die Test-DB-Nutzung über die Paketgrenzen hinweg. Alle
// Pakete teilen sich EINE Test-DB; `go test ./...` startet ihre Binaries
// parallel. Ohne diesen Lock machen mehrere gleichzeitig DROP SCHEMA und ziehen
// sich die Tabellen weg (Deadlock).
const lockKey int64 = 0x42DB0001

// Pool und Schema werden je Test-Binary genau einmal aufgebaut: schema.sql ist
// nicht idempotent (CREATE TYPE bricht beim zweiten Lauf ab); aufräumen müssen
// die Tests selbst (Rollback oder TRUNCATE ihrer Tabellen).
var (
	once sync.Once
	pool *pgxpool.Pool
	err  error
	// lockConn hält den paketübergreifenden Advisory-Lock über eine dedizierte
	// Connection, die absichtlich bis zum Prozessende (Ende der Paket-Tests)
	// offen bleibt — dann gibt Postgres den Lock automatisch frei und das
	// nächste Test-Binary darf ran.
	lockConn *pgx.Conn //nolint:unused // Referenz hält die Verbindung am Leben
)

// Pool liefert den gemeinsamen Test-Pool mit frisch geladenem schema.sql.
// Ohne TEST_DATABASE_URL wird der Test übersprungen (lokal ist kein Postgres
// Pflicht).
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(EnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", EnvVar)
	}
	once.Do(func() { pool, err = baueTestDB(dsn) })
	if err != nil {
		t.Fatalf("Test-DB konnte nicht vorbereitet werden: %v", err)
	}
	return pool
}

// schemaPfad findet schema.sql relativ zu DIESER Datei — unabhängig davon, aus
// welchem Paketverzeichnis das Test-Binary läuft.
func schemaPfad() string {
	_, dieseDatei, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(dieseDatei), "..", "..", "schema.sql")
}

// baueTestDB verbindet zur Test-DB und spielt schema.sql ein — den Zustand, den
// auch eine Neuinstallation erhält. Das Schema wird vorher verworfen, denn
// schema.sql ist nicht idempotent; ohne den Reset wäre der Lauf nur gegen eine
// jungfräuliche DB grün.
func baueTestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	// Paketübergreifenden Lock nehmen, bevor irgendjemand das Schema anfasst.
	lc, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if _, err := lc.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return nil, err
	}
	lockConn = lc // offen halten bis Prozessende

	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		return nil, err
	}
	if err := pruefeTestDatenbank(ctx, p); err != nil {
		p.Close()
		return nil, err
	}

	// Reset auf leeres Schema — sicher, weil pruefeTestDatenbank oben abgesichert
	// hat, dass wir NICHT auf einer produktiven Datenbank arbeiten.
	if _, err := p.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		return nil, err
	}
	sqlText, err := os.ReadFile(schemaPfad())
	if err != nil {
		return nil, err
	}
	if _, err := p.Exec(ctx, string(sqlText)); err != nil {
		return nil, err
	}
	return p, nil
}

// pruefeTestDatenbank ist die Notbremse vor dem DROP SCHEMA: Diese Tests löschen
// das gesamte Schema. Zeigt TEST_DATABASE_URL versehentlich auf eine echte
// Datenbank, wäre das ein Totalverlust. Deshalb muss der Datenbankname „test"
// enthalten.
func pruefeTestDatenbank(ctx context.Context, p *pgxpool.Pool) error {
	var name string
	if err := p.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		return fmt.Errorf(
			"keine Test-Datenbank: %q enthält nicht \"test\" — diese Tests verwerfen das "+
				"gesamte Schema, %s darf nur auf eine Wegwerf-Datenbank zeigen", name, EnvVar)
	}
	return nil
}
