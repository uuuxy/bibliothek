package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestKlassenVokabular sichert die Invarianten aus Migration 079: Die Klasse ist ein
// kontrolliertes Vokabular (Tabelle klassen), durchgesetzt an der Datenbank selbst.
// BEFORE-Trigger registrieren unbekannte Klassen automatisch und kanonisieren jede
// Schreibvariante ("05A", "5 A" → "5a"); FKs mit ON UPDATE CASCADE ziehen ein
// Umbenennen durch alle vier Tabellen, ON DELETE RESTRICT schützt benutzte Namen.
// Getestet wird der von schema.sql ausgelieferte Endzustand (den auch der
// Migrationslauf herstellen muss).
func TestKlassenVokabular(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	const insSchueler = `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
	                     VALUES ($1, 'Test', 'Test', $2, 2030)`

	klasseVon := func(t *testing.T, tx pgx.Tx, barcode string) string {
		t.Helper()
		var klasse string
		if err := tx.QueryRow(ctx,
			`SELECT klasse FROM schueler WHERE barcode_id = $1`, barcode).Scan(&klasse); err != nil {
			t.Fatalf("Klasse lesen: %v", err)
		}
		return klasse
	}

	t.Run("unbekannte klasse wird automatisch registriert", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler mit neuer Klasse", insSchueler, "KV-1", "9z")
			var registriert int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM klassen WHERE name = '9z'`).Scan(&registriert); err != nil {
				t.Fatalf("Vokabular prüfen: %v", err)
			}
			if registriert != 1 {
				t.Fatalf("Klasse wurde nicht auto-registriert (count=%d)", registriert)
			}
		})
	})

	t.Run("schreibvarianten laufen auf die registrierte form", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Erstschreiber", insSchueler, "KV-2a", "5a")
			erwarteErfolg(t, tx, "Null-Präfix-Variante", insSchueler, "KV-2b", "05A")
			erwarteErfolg(t, tx, "Leerzeichen-Variante", insSchueler, "KV-2c", " 5 a ")
			if k := klasseVon(t, tx, "KV-2b"); k != "5a" {
				t.Errorf("'05A' muss zu '5a' kanonisiert werden, war %q", k)
			}
			if k := klasseVon(t, tx, "KV-2c"); k != "5a" {
				t.Errorf("' 5 a ' muss zu '5a' kanonisiert werden, war %q", k)
			}
			var eintraege int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM klassen WHERE klassen_normkey(name) = klassen_normkey('5a')`).Scan(&eintraege); err != nil {
				t.Fatalf("Vokabular zählen: %v", err)
			}
			if eintraege != 1 {
				t.Errorf("Schreibvarianten dürfen keine Vokabular-Dubletten erzeugen (count=%d)", eintraege)
			}
		})
	})

	t.Run("kanonisierung wirkt in allen vier tabellen", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchueler, "KV-3", "7b")
			erwarteErfolg(t, tx, "Lehrkraft-Zuordnung mit Variante",
				`INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('07B', 'x@example.org')`)
			var mappingKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT klasse FROM klassen_lehrer_mapping WHERE lehrer_email = 'x@example.org'`).Scan(&mappingKlasse); err != nil {
				t.Fatalf("Mapping lesen: %v", err)
			}
			if mappingKlasse != "7b" {
				t.Errorf("Mapping muss auf '7b' kanonisiert werden, war %q — genau der Drift, der Mahn-Mails ins Leere schickte", mappingKlasse)
			}

			var titelID string
			if err := tx.QueryRow(ctx,
				`INSERT INTO buecher_titel (titel) VALUES ('KV-Buch') RETURNING id`).Scan(&titelID); err != nil {
				t.Fatalf("Titel anlegen: %v", err)
			}
			erwarteErfolg(t, tx, "Bücherliste mit Variante",
				`INSERT INTO class_books (class_name, book_id) VALUES ('7 B', $1)`, titelID)
			var listenKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT class_name FROM class_books WHERE book_id = $1`, titelID).Scan(&listenKlasse); err != nil {
				t.Fatalf("Bücherliste lesen: %v", err)
			}
			if listenKlasse != "7b" {
				t.Errorf("Bücherliste muss auf '7b' kanonisiert werden, war %q", listenKlasse)
			}

			erwarteErfolg(t, tx, "Klassensatz mit Variante",
				`INSERT INTO klassensatz_reservierungen (titel_id, klasse) VALUES ($1, '07 b')`, titelID)
			var ksKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT klasse FROM klassensatz_reservierungen WHERE titel_id = $1`, titelID).Scan(&ksKlasse); err != nil {
				t.Fatalf("Klassensatz lesen: %v", err)
			}
			if ksKlasse != "7b" {
				t.Errorf("Klassensatz muss auf '7b' kanonisiert werden, war %q", ksKlasse)
			}
		})
	})

	t.Run("umbenennen im vokabular zieht alle tabellen mit", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchueler, "KV-4", "6c")
			erwarteErfolg(t, tx, "Zuordnung",
				`INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('6c', 'y@example.org')`)
			erwarteErfolg(t, tx, "Umbenennung",
				`UPDATE klassen SET name = '6d' WHERE name = '6c'`)
			if k := klasseVon(t, tx, "KV-4"); k != "6d" {
				t.Errorf("CASCADE muss den Schüler mitziehen, war %q", k)
			}
			var mappingKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT klasse FROM klassen_lehrer_mapping WHERE lehrer_email = 'y@example.org'`).Scan(&mappingKlasse); err != nil {
				t.Fatalf("Mapping lesen: %v", err)
			}
			if mappingKlasse != "6d" {
				t.Errorf("CASCADE muss die Zuordnung mitziehen, war %q", mappingKlasse)
			}
		})
	})

	t.Run("loeschen ist gesperrt solange der name benutzt wird", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchueler, "KV-5", "8e")
			erwarteConstraintVerletzung(t, tx, "fk_schueler_klasse_vokabular",
				`DELETE FROM klassen WHERE name = '8e'`)
		})
	})

	t.Run("sonderwerte ABG und leer sind gewoehnliche vokabeln", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Abgänger", insSchueler, "KV-6a", "ABG")
			erwarteErfolg(t, tx, "DSGVO-anonymisiert", insSchueler, "KV-6b", "")
			var registriert int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM klassen WHERE name IN ('ABG', '')`).Scan(&registriert); err != nil {
				t.Fatalf("Sonderwerte prüfen: %v", err)
			}
			if registriert != 2 {
				t.Errorf("'ABG' und '' müssen als Vokabeln registriert sein (count=%d)", registriert)
			}
		})
	})
}
