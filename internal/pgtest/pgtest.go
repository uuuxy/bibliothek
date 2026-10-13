// Package pgtest stellt den Tests an der Datenbank die Testdatenbank bereit: einen gemeinsamen
// Pool mit frisch eingespieltem schema.sql.
//
// Es ist ein gewöhnliches Paket, das nur Testdateien einbinden; eine Testdatei lässt sich
// nicht über die Grenze ihres Pakets einbinden. Die Liste des toten Codes führt seine
// Funktionen deshalb mit Begründung.
//
// Geprüft wird an einem echten Postgres und nicht mit pgxmock, weil die Regeln in SQL stehen
// (Constraints, partielle Unique-Indizes, Filter, Trigger); pgxmock prüfte nur nachgespielte
// Antworten.
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

// Variable nennt die Umgebungsvariable mit der Adresse der Testdatenbank. Ohne sie
// überspringen sich die Tests an der Datenbank; in der CI setzt der Workflow sie auf den
// Postgres des Laufs, und db/pgtest_guard_test.go hält fest, dass er es tut.
const Variable = "TEST_DATABASE_URL"

// sperrSchluessel reiht die Pakete an der Testdatenbank ein. Alle Pakete teilen sich eine
// Datenbank, und `go test ./...` startet ihre Programme nebeneinander; ohne die Sperre
// verwürfen mehrere zugleich das Schema und zögen einander die Tabellen weg.
const sperrSchluessel int64 = 0x42DB0001

// Pool und Schema entstehen je Testprogramm einmal: schema.sql lässt sich nicht zweimal
// einspielen (CREATE TYPE bricht beim zweiten Mal ab). Aufräumen müssen die Tests selbst, mit
// Rollback oder TRUNCATE ihrer Tabellen.
var (
	once   sync.Once
	pool   *pgxpool.Pool
	fehler error
	// sperrVerbindung hält die Sperre über eine eigene Verbindung, die bis zum Ende des
	// Prozesses offen bleibt. Dann gibt Postgres die Sperre frei, und das nächste
	// Testprogramm ist an der Reihe.
	sperrVerbindung *pgx.Conn //nolint:unused // Referenz hält die Verbindung am Leben
)

// Pool liefert den gemeinsamen Pool der Tests mit frisch eingespieltem schema.sql. Ohne
// TEST_DATABASE_URL überspringt sich der Test; lokal ist kein Postgres Pflicht.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv(Variable)
	if dsn == "" {
		t.Skipf("%s nicht gesetzt — DB-Integrationstest übersprungen", Variable)
	}
	once.Do(func() { pool, fehler = baueTestDB(dsn) })
	if fehler != nil {
		t.Fatalf("Test-DB konnte nicht vorbereitet werden: %v", fehler)
	}
	return pool
}

// schemaPfad findet schema.sql von dieser Datei aus, gleich aus welchem Paketordner das
// Testprogramm läuft.
func schemaPfad() string {
	_, dieseDatei, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(dieseDatei), "..", "..", "schema.sql")
}

// baueTestDB verbindet zur Testdatenbank und spielt schema.sql ein, den Stand einer neuen
// Installation. Das Schema wird vorher verworfen, weil schema.sql sich nicht zweimal
// einspielen lässt.
func baueTestDB(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()

	// Die Sperre zuerst, bevor jemand das Schema anfasst.
	verbindung, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if _, err := verbindung.Exec(ctx, "SELECT pg_advisory_lock($1)", sperrSchluessel); err != nil {
		return nil, err
	}
	sperrVerbindung = verbindung // bleibt offen bis zum Ende des Prozesses

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

	// Das Verwerfen ist sicher, weil pruefeTestDatenbank eine echte Datenbank ausgeschlossen
	// hat.
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

// pruefeTestDatenbank schützt vor dem Verwerfen des Schemas: Zeigt TEST_DATABASE_URL aus
// Versehen auf eine echte Datenbank, wäre alles verloren. Der Name der Datenbank muss deshalb
// „test" enthalten.
func pruefeTestDatenbank(ctx context.Context, p *pgxpool.Pool) error {
	var name string
	if err := p.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		return fmt.Errorf(
			"keine Test-Datenbank: %q enthält nicht \"test\" — diese Tests verwerfen das "+
				"gesamte Schema, %s darf nur auf eine Wegwerf-Datenbank zeigen", name, Variable)
	}
	return nil
}
