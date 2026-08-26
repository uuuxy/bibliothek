package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestKlassenVokabular sichert die Invarianten aus Migration 079: Die Klasse ist ein
// kontrolliertes Vokabular (Tabelle klassen), durchgesetzt an der Datenbank selbst.
// BEFORE-Trigger registrieren unbekannte Klassen automatisch und kanonisieren jede
// Schreibvariante ("05A", "5 A" → "05A"); FKs mit ON UPDATE CASCADE ziehen ein
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
			erwarteErfolg(t, tx, "Schüler mit neuer Klasse", insSchueler, "KV-1", "09Z")
			var registriert int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM klassen WHERE name = '09Z'`).Scan(&registriert); err != nil {
				t.Fatalf("Vokabular prüfen: %v", err)
			}
			if registriert != 1 {
				t.Fatalf("Klasse wurde nicht auto-registriert (count=%d)", registriert)
			}
		})
	})

	t.Run("schreibvarianten laufen auf die registrierte form", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Erstschreiber", insSchueler, "KV-2a", "05A")
			erwarteErfolg(t, tx, "Null-Präfix-Variante", insSchueler, "KV-2b", "05A")
			erwarteErfolg(t, tx, "Leerzeichen-Variante", insSchueler, "KV-2c", " 5 a ")
			if k := klasseVon(t, tx, "KV-2b"); k != "05A" {
				t.Errorf("'05A' muss auf die Anzeigeform '05A' laufen, war %q", k)
			}
			if k := klasseVon(t, tx, "KV-2c"); k != "05A" {
				t.Errorf("' 5 a ' muss zu '05A' kanonisiert werden, war %q", k)
			}
			var eintraege int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM klassen WHERE klassen_normkey(name) = klassen_normkey('05A')`).Scan(&eintraege); err != nil {
				t.Fatalf("Vokabular zählen: %v", err)
			}
			if eintraege != 1 {
				t.Errorf("Schreibvarianten dürfen keine Vokabular-Dubletten erzeugen (count=%d)", eintraege)
			}
		})
	})

	t.Run("kanonisierung wirkt in allen vier tabellen", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchueler, "KV-3", "07B")
			erwarteErfolg(t, tx, "Lehrkraft-Zuordnung mit Variante",
				`INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('07B', 'x@example.org')`)
			var mappingKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT klasse FROM klassen_lehrer_mapping WHERE lehrer_email = 'x@example.org'`).Scan(&mappingKlasse); err != nil {
				t.Fatalf("Mapping lesen: %v", err)
			}
			if mappingKlasse != "07B" {
				t.Errorf("Mapping muss auf '07B' kanonisiert werden, war %q — genau der Drift, der Mahn-Mails ins Leere schickte", mappingKlasse)
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
			if listenKlasse != "07B" {
				t.Errorf("Bücherliste muss auf '07B' kanonisiert werden, war %q", listenKlasse)
			}

			erwarteErfolg(t, tx, "Klassensatz mit Variante",
				`INSERT INTO klassensatz_reservierungen (titel_id, klasse) VALUES ($1, '07 b')`, titelID)
			var ksKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT klasse FROM klassensatz_reservierungen WHERE titel_id = $1`, titelID).Scan(&ksKlasse); err != nil {
				t.Fatalf("Klassensatz lesen: %v", err)
			}
			if ksKlasse != "07B" {
				t.Errorf("Klassensatz muss auf '07B' kanonisiert werden, war %q", ksKlasse)
			}
		})
	})

	// Anzeigeform (Migration 087): Bis dahin gewann die ERSTE Schreibweise — „9G4" neben
	// „09G1", „10g1" neben „10G2". Jetzt wird beim Registrieren vereinheitlicht; Sonderwerte
	// und Kursnamen bleiben unangetastet.
	t.Run("anzeigeform: jahrgang zweistellig, rest gross", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			for eingabe, erwartet := range map[string]string{
				"9g4": "09G4", "10g1": "10G1", "07r1": "07R1", "5 f 1": "05F1", "lehrer": "lehrer", "ABG": "ABG", "Kurs Mathe": "Kurs Mathe",
			} {
				erwarteErfolg(t, tx, "Schüler "+eingabe, insSchueler, "KV-AF-"+eingabe, eingabe)
				if k := klasseVon(t, tx, "KV-AF-"+eingabe); k != erwartet {
					t.Errorf("%q muss als %q registriert werden, war %q", eingabe, erwartet, k)
				}
			}
		})
	})

	t.Run("umbenennen im vokabular zieht alle tabellen mit", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchueler, "KV-4", "06C")
			erwarteErfolg(t, tx, "Zuordnung",
				`INSERT INTO klassen_lehrer_mapping (klasse, lehrer_email) VALUES ('06C', 'y@example.org')`)
			erwarteErfolg(t, tx, "Umbenennung",
				`UPDATE klassen SET name = '06D' WHERE name = '06C'`)
			if k := klasseVon(t, tx, "KV-4"); k != "06D" {
				t.Errorf("CASCADE muss den Schüler mitziehen, war %q", k)
			}
			var mappingKlasse string
			if err := tx.QueryRow(ctx,
				`SELECT klasse FROM klassen_lehrer_mapping WHERE lehrer_email = 'y@example.org'`).Scan(&mappingKlasse); err != nil {
				t.Fatalf("Mapping lesen: %v", err)
			}
			if mappingKlasse != "06D" {
				t.Errorf("CASCADE muss die Zuordnung mitziehen, war %q", mappingKlasse)
			}
		})
	})

	t.Run("loeschen ist gesperrt solange der name benutzt wird", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "Schüler", insSchueler, "KV-5", "08E")
			erwarteConstraintVerletzung(t, tx, "fk_schueler_klasse_vokabular",
				`DELETE FROM klassen WHERE name = '08E'`)
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
