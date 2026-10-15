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

// Die Tests in diesem Paket laufen gegen echtes PostgreSQL, und das ist keine Kür: Der Fehler,
// den sie absichern, besteht nur dort. Postgres versetzt eine Transaktion beim ersten Fehler in
// den Zustand des Abbruchs (SQLSTATE 25P02); eine Attrappe kennt diesen Zustand nicht und lässt
// jedes `continue` in der Schleife plausibel aussehen.
//
// Aufbau des Pools, Sperre der Datenbank und Notbremse liegen in internal/pgtest; hier stehen nur
// die Helfer dieses Pakets.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
}

// leereBestand setzt Titel und Exemplare zurück. Anders als in db/ ist kein Zurückrollen um den
// Testfall möglich: legeBatchAn öffnet seine eigene Transaktion, und genau deren COMMIT wird hier
// geprüft.
func leereBestand(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE buecher_exemplare, buecher_titel RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("Bestand konnte nicht geleert werden: %v", err)
	}
}

// testFehlerprotokoll legt das Fehlerprotokoll in einem vorübergehenden Verzeichnis an und
// liefert eine Funktion, die es geleert zurückliest; das Protokoll ist gepuffert, ungeleert steht
// dort nichts.
func testFehlerprotokoll(t *testing.T) (*uebernahme.Protokoll, func() string) {
	t.Helper()
	pfad := filepath.Join(t.TempDir(), "migration_errors.log")
	fehlerprotokoll, err := newFehlerprotokollIn(pfad)
	if err != nil {
		t.Fatalf("Fehlerprotokoll konnte nicht angelegt werden: %v", err)
	}
	t.Cleanup(fehlerprotokoll.Schliesse)
	return fehlerprotokoll, func() string {
		if err := fehlerprotokoll.Leere(); err != nil {
			t.Fatalf("Fehlerprotokoll konnte nicht geschrieben werden: %v", err)
		}
		b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
		if err != nil {
			t.Fatalf("Fehlerprotokoll konnte nicht gelesen werden: %v", err)
		}
		return string(b)
	}
}
