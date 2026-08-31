package repository

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integrationstests gegen echtes Postgres (gated auf TEST_DATABASE_URL, wie im
// db-Paket). Warum echtes PG und nicht pgxmock: Die Inventur-Session-Logik lebt fast
// vollständig im SQL (Scope-Bedingungen, partielle Unique-Indizes, Verlust-UPDATE mit
// Erfassungs-Join) — pgxmock würde nur nachgespielte Antworten prüfen.
//
// Pool-Aufbau, Advisory-Lock und Notbremse liegen seit dem 31.08.2026 in
// internal/pgtest (vorher fünffach kopiert); hier stehen nur noch die Helfer
// dieses Pakets.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
}

// resetInventurDaten räumt zwischen Tests die Bestands-, Ausleih- und Personendaten
// leer, damit jeder Test von einer bekannten Basis startet (schema-Load passiert nur
// einmal). CASCADE räumt abhängige Zeilen (u. a. schadensfaelle) mit.
func resetInventurDaten(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE inventur_erfassungen, inventur_sessions, schadensfaelle, ausleihen,
		         buecher_exemplare, buecher_titel, schueler, benutzer, klassen
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("Reset der Testdaten fehlgeschlagen: %v", err)
	}
}

// seedSignaturMitExemplaren legt n ausleihbare Exemplare unter der Signatur an und
// liefert die Exemplar-IDs. Die Signatur ist seit Migration 060 der Text auf dem
// Buchrücken (buecher_titel.signatur), kein Fremdschlüssel mehr.
func seedSignaturMitExemplaren(t *testing.T, pool *pgxpool.Pool, signatur string, n int) []string {
	t.Helper()
	ctx := context.Background()

	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		var titelID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, signatur) VALUES ($1, $2) RETURNING id`,
			fmt.Sprintf("%s-Buch-%d", signatur, i), signatur).Scan(&titelID); err != nil {
			t.Fatalf("Titel anlegen: %v", err)
		}
		var exID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
			titelID, fmt.Sprintf("BC-%s-%d", signatur, i)).Scan(&exID); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		ids = append(ids, exID)
	}
	return ids
}

// sichereTestSystematik registriert das Fach in systematik_kategorien —
// buecher_titel.subject ist seit Migration 078 ein FK auf die Bezeichnung, direkte
// Titel-INSERTs mit Fach scheitern sonst. (Die produktiven Schreibpfade registrieren
// selbst, via inventur.StelleFaecherSicher; Test-Seeds schreiben daran vorbei.)
func sichereTestSystematik(t *testing.T, pool *pgxpool.Pool, subject string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
		VALUES (replace($1, ' ', ''), $1)
		ON CONFLICT (lower(bezeichnung)) DO NOTHING
	`, subject); err != nil {
		t.Fatalf("Test-Systematik %q anlegen: %v", subject, err)
	}
}

// seedSignaturFachExemplar legt einen Titel an, der BEIDE Scope-Dimensionen trägt
// (Signatur UND Fach/Jahrgang) — genau die Konstellation, in der ein Signatur- und ein
// Filter-Scope dasselbe Exemplar treffen. Liefert die Exemplar-ID.
func seedSignaturFachExemplar(t *testing.T, pool *pgxpool.Pool, signatur, subject string, jvon, jbis int, barcode string) string {
	t.Helper()
	ctx := context.Background()
	sichereTestSystematik(t, pool, subject)

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, signatur, subject, jahrgang_von, jahrgang_bis)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		fmt.Sprintf("%s-%s-%s", signatur, subject, barcode), signatur, subject, jvon, jbis).Scan(&titelID); err != nil {
		t.Fatalf("Signatur+Fach-Titel anlegen: %v", err)
	}
	var exID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
		titelID, barcode).Scan(&exID); err != nil {
		t.Fatalf("Signatur+Fach-Exemplar anlegen: %v", err)
	}
	return exID
}

// seedFachExemplar legt einen Titel mit Fach (subject) + Jahrgangsbereich und genau
// einem ausleihbaren Exemplar an; liefert die Exemplar-ID. Für Filter-Scope-Tests.
func seedFachExemplar(t *testing.T, pool *pgxpool.Pool, subject string, jvon, jbis int, barcode string) string {
	t.Helper()
	ctx := context.Background()
	sichereTestSystematik(t, pool, subject)

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, subject, jahrgang_von, jahrgang_bis)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		fmt.Sprintf("%s-Kl%d-%s", subject, jvon, barcode), subject, jvon, jbis).Scan(&titelID); err != nil {
		t.Fatalf("Fach-Titel anlegen: %v", err)
	}
	var exID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, $2) RETURNING id`,
		titelID, barcode).Scan(&exID); err != nil {
		t.Fatalf("Fach-Exemplar anlegen: %v", err)
	}
	return exID
}
