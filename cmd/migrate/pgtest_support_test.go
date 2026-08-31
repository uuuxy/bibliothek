package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Tests in diesem Paket laufen gegen ECHTES PostgreSQL, und das ist keine Kür:
// Der Fehler, den sie absichern, existiert ausschließlich dort. Postgres versetzt eine
// Transaktion beim ersten Fehler in den Abbruchzustand (SQLSTATE 25P02) — ein Mock kennt
// diesen Zustand nicht und lässt jedes `continue` in der Schleife plausibel aussehen.
// Genau deshalb ist der Datenverlust in cmd/migrate jahrelang unbemerkt geblieben.
//
// Pool-Aufbau, Advisory-Lock und Notbremse liegen seit dem 31.08.2026 in
// internal/pgtest (vorher fünffach kopiert); hier stehen nur noch die Helfer
// dieses Pakets.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
}

// leereBestand setzt Titel und Exemplare zurück. Anders als in db/ ist kein Rollback um
// den Testfall möglich: insertBatch öffnet seine eigene Transaktion, und genau deren
// COMMIT ist hier der Prüfgegenstand.
func leereBestand(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE buecher_exemplare, buecher_titel RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Bestand konnte nicht geleert werden: %v", err)
	}
}

// testLogger legt das Fehlerprotokoll in einem temporären Verzeichnis an und liefert eine
// Funktion, die es geleert zurückliest — das Protokoll ist gepuffert, ungeflusht steht
// dort nichts.
func testLogger(t *testing.T) (*uebernahme.Protokoll, func() string) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "migration_errors.log")
	el, err := newErrLoggerAt(pfad)
	if err != nil {
		t.Fatalf("Fehlerprotokoll konnte nicht angelegt werden: %v", err)
	}
	t.Cleanup(el.Schliessen)
	return el, func() string {
		if err := el.Leeren(); err != nil {
			t.Fatalf("Fehlerprotokoll konnte nicht geschrieben werden: %v", err)
		}
		b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
		if err != nil {
			t.Fatalf("Fehlerprotokoll konnte nicht gelesen werden: %v", err)
		}
		return string(b)
	}
}
