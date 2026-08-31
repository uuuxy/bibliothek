package littera

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integrationstests für den Littera-Schreibpfad — gegen echtes Postgres, weil der
// Prüfgegenstand die COMMITs der Batch-Transaktionen sind (siehe testSchreiber).
//
// Pool-Aufbau, Advisory-Lock und Notbremse liegen seit dem 31.08.2026 in
// internal/pgtest (vorher fünffach kopiert); hier stehen nur noch die Helfer
// dieses Pakets.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
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
