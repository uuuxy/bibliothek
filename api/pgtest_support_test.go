package api

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PG-Integrationstests fürs api-Paket (gated auf TEST_DATABASE_URL, wie db/ und
// repository/). Nötig für die order-/graduates-Bugs, deren Kern in SQL-Filtern liegt
// (bereits bestellte Exemplare, numerische Barcode-Sortierung, Abgänger-Filter) —
// pgxmock würde nur nachgespielte Antworten prüfen, nicht die SQL-Korrektheit.
//
// Pool-Aufbau, Advisory-Lock und Notbremse liegen seit dem 31.08.2026 in
// internal/pgtest (vorher fünffach kopiert); hier stehen nur noch die Helfer
// dieses Pakets.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
}

// resetBestandsdaten leert Bestands-, Bestell- und Personendaten zwischen Tests.
// klassen gehört mit in den Reset, obwohl dort keine Testdaten im üblichen Sinn stehen:
// Die Tabelle ist das Vokabular, das der Trigger trg_schueler_klasse_vokabular
// (Migration 079) beim Schreiben nachschlägt. Steht dort schon "7a", wird ein später
// eingefügtes "07a" stillschweigend als "7a" gespeichert — steht sie leer, bleibt "07a"
// stehen und wird selbst zur kanonischen Form.
//
// Damit hing die SCHREIBWEISE einer Klasse davon ab, welcher Test vorher gelaufen war.
// Am 23.08.2026 machte das einen neuen Test allein grün und in der vollen Suite rot; die
// erste Erklärung dafür ("ein anderer Test schreibt alle Schülerzeilen") war falsch — das
// Paket kennt kein t.Parallel(), und schueler wird ohnehin geleert. Die Kopplung lief über
// diese eine nicht zurückgesetzte Tabelle. Die gefährliche Richtung ist die umgekehrte:
// ein Test, den fremdes Vokabular still grün hält.
//
// Truncate mit CASCADE räumt die vier referenzierenden Tabellen mit (schueler,
// klassen_lehrer_mapping, class_books, klassensatz_reservierungen) — alles Testdaten.
// Befüllt wird klassen von keiner Migration, sie entsteht allein durch den Trigger.
func resetBestandsdaten(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE buecher_exemplare, buecher_titel, ausleihen, schueler, benutzer, klassen
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("Reset fehlgeschlagen: %v", err)
	}
}

// titelMitMeldebestand legt einen Titel mit gegebenem Meldebestand an.
func titelMitMeldebestand(t *testing.T, pool *pgxpool.Pool, titel string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, meldebestand) VALUES ($1, $2) RETURNING id`,
		titel, meldebestand).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	return id
}

// titelMitSignatur legt einen Titel mit expliziter Signatur an — für Fälle, in denen
// das LMF-Kennzeichen (anders als bei titelMitMeldebestand) nicht im Titel steht,
// sondern nur in der Signatur.
func titelMitSignatur(t *testing.T, pool *pgxpool.Pool, titel, signatur string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, signatur, meldebestand) VALUES ($1, $2, $3) RETURNING id`,
		titel, signatur, meldebestand).Scan(&id); err != nil {
		t.Fatalf("Titel mit Signatur anlegen: %v", err)
	}
	return id
}

// exemplar legt ein Exemplar mit Verleih-/Aussonderungsstatus und Notiz an.
func exemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string, ausleihbar bool, notiz string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		titelID, barcode, ausleihbar, notiz).Scan(&id); err != nil {
		t.Fatalf("Exemplar %q anlegen: %v", barcode, err)
	}
	return id
}
