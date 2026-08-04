package littera

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Schreibpfad-Tests laufen gegen ECHTES PostgreSQL, und das ist keine Kür: Was hier
// abgesichert wird — Savepoint je Datensatz, Verhalten bei 23505, partielle Unique-Indizes
// wie uniq_ausleihen_aktiv_exemplar — existiert nur dort. Ein Mock kennt weder den
// Abbruchzustand einer Transaktion (25P02) noch die Constraints aus schema.sql und ließe
// jeden Datenverlust plausibel aussehen.
//
// Ohne TEST_DATABASE_URL werden sie übersprungen; in CI setzt der Workflow die Variable.

const testDBEnvVar = "TEST_DATABASE_URL"

// testDBLockKey serialisiert die Test-DB-Nutzung über db/, repository/, api/, cmd/migrate/
// und internal/littera/ — alle teilen sich EINE Test-DB, und `go test ./...` startet ihre
// Binaries parallel. Ohne den Lock kollidieren gleichzeitige DROP SCHEMA. Wert identisch
// in allen Paketen halten.
const testDBLockKey int64 = 0x42DB0001

var (
	pgTestOnce sync.Once
	pgTestDB   *pgxpool.Pool
	pgTestErr  error
	lockConn   *pgx.Conn // hält den Lock bis Prozessende
)

func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(testDBEnvVar)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", testDBEnvVar)
	}
	pgTestOnce.Do(func() { pgTestDB, pgTestErr = baueTestDB(dsn) })
	if pgTestErr != nil {
		t.Fatalf("Test-DB konnte nicht vorbereitet werden: %v", pgTestErr)
	}
	return pgTestDB
}

func schemaPfad() string {
	_, dieseDatei, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(dieseDatei), "..", "..", "schema.sql")
}

func baueTestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	lc, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if _, err := lc.Exec(ctx, "SELECT pg_advisory_lock($1)", testDBLockKey); err != nil {
		return nil, err
	}
	lockConn = lc

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	if err := pruefeTestDatenbank(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		return nil, err
	}
	sqlText, err := os.ReadFile(schemaPfad())
	if err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, string(sqlText)); err != nil {
		return nil, err
	}
	return pool, nil
}

// pruefeTestDatenbank ist die Notbremse vor dem DROP SCHEMA: Zeigt TEST_DATABASE_URL
// versehentlich auf eine echte Datenbank, wäre das ein Totalverlust.
func pruefeTestDatenbank(ctx context.Context, pool *pgxpool.Pool) error {
	var name string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		return fmt.Errorf(
			"Sicherheitsabbruch: Datenbank %q enthält nicht \"test\". Diese Tests verwerfen das "+
				"gesamte Schema — %s darf nur auf eine Wegwerf-Datenbank zeigen", name, testDBEnvVar)
	}
	return nil
}

// leereAlles setzt die vom Schreibpfad berührten Tabellen zurück. Ein Rollback um den
// Testfall ist nicht möglich: Der Schreiber öffnet eigene Transaktionen, und genau deren
// COMMIT ist hier der Prüfgegenstand.
func leereAlles(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`TRUNCATE ausleihen, buecher_exemplare, buecher_titel, schueler, benutzer RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("Tabellen konnten nicht geleert werden: %v", err)
	}
}

// testSchreiber baut einen Schreiber mit Protokoll in einem temporären Verzeichnis und
// liefert eine Funktion, die das Protokoll geleert zurückliest.
func testSchreiber(t *testing.T, pool *pgxpool.Pool, anpassen func(*Optionen)) (*Schreiber, func() string) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "littera_import.log")
	prot, err := uebernahme.NeuesProtokoll(pfad, "littera_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(prot.Schliessen)

	opt := StandardOptionen(time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC))
	opt.BatchGroesse = 3 // klein, damit die Tests mehrere Transaktionen durchlaufen
	if anpassen != nil {
		anpassen(&opt)
	}

	lies := func() string {
		if err := prot.Leeren(); err != nil {
			t.Fatalf("Protokoll schreiben: %v", err)
		}
		b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
		if err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		return string(b)
	}
	return NeuerSchreiber(pool, prot, opt), lies
}

func zaehle(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("Zählung (%s): %v", sql, err)
	}
	return n
}
