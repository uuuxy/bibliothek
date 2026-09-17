package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// Die Wächter der beiden Spalten aus Migration 127 (docs/OFFEN.md 9.8).
//
// Beide Werte gehen am Ende in einen Ersatzbetrag ein, und der steht in einem Bescheid an
// Erziehungsberechtigte. Ein negativer Listenpreis oder eine Abwertung von 150 % ergäbe
// dort eine Zahl, die niemand erklären kann — und zwar still, weil die Rechnung einfach
// durchläuft.
func TestListenpreisUndAbwertungConstraints(t *testing.T) {
	pool := pgTestPool(t)

	t.Run("Listenpreis darf nicht negativ sein", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteConstraintVerletzung(t, tx, "chk_listenpreis_nonneg",
				`INSERT INTO buecher_titel (titel, listenpreis) VALUES ('Negativ', -0.01)`)
		})
	})

	t.Run("Listenpreis darf FEHLEN — das ist kein Fehler, sondern eine Auskunft", func(t *testing.T) {
		// NULL heißt „nicht erfasst": Die Staffel weicht dann auf den Kaufpreis aus und
		// sagt das in ihrer Herleitung. Wäre die Spalte NOT NULL DEFAULT 0, hieße dasselbe
		// „dieses Buch kostet heute nichts" — und der Bescheid nennte 0,00 €.
		inTx(t, pool, func(tx pgx.Tx) {
			erwarteErfolg(t, tx, "ohne Listenpreis",
				`INSERT INTO buecher_titel (titel) VALUES ('Ohne Preis')`)
			erwarteErfolg(t, tx, "Listenpreis 0 ist erlaubt, aber etwas anderes als NULL",
				`INSERT INTO buecher_titel (titel, listenpreis) VALUES ('Verschenkt', 0)`)
		})
	})

	t.Run("Abwertung bleibt zwischen 0 und 100 Prozent", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteConstraintVerletzung(t, tx, "chk_zustand_abwertung_bereich",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_abwertung_prozent)
				 VALUES ($1, 'ABW-NEG', -1)`, titelID)
			erwarteConstraintVerletzung(t, tx, "chk_zustand_abwertung_bereich",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_abwertung_prozent)
				 VALUES ($1, 'ABW-101', 101)`, titelID)
		})
	})

	t.Run("0 und 100 Prozent sind beide erlaubt", func(t *testing.T) {
		// 100 % ist kein Unfall, sondern die zweite Fallgruppe des Musteranschreibens:
		// so stark beschädigt, dass unbenutzbar. Der Ersatzbetrag ist dann 0 — die Schule
		// fordert für ein wertloses Buch nichts, sie sondert es aus.
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			erwarteErfolg(t, tx, "0 Prozent",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_abwertung_prozent)
				 VALUES ($1, 'ABW-0', 0)`, titelID)
			erwarteErfolg(t, tx, "100 Prozent",
				`INSERT INTO buecher_exemplare (titel_id, barcode_id, zustand_abwertung_prozent)
				 VALUES ($1, 'ABW-100', 100)`, titelID)
		})
	})

	t.Run("ein neues Exemplar startet bei 0 Prozent", func(t *testing.T) {
		// Die Vorgabe muss greifen, ohne dass ein Schreibpfad sie kennt: Jeder vorhandene
		// INSERT im Baum nennt die Spalte nicht.
		inTx(t, pool, func(tx pgx.Tx) {
			titelID := neuerTitel(t, tx)
			var prozent int
			if err := tx.QueryRow(t.Context(), `
				INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ($1, 'ABW-DEF')
				RETURNING zustand_abwertung_prozent`, titelID).Scan(&prozent); err != nil {
				t.Fatalf("Exemplar ohne Abwertung anlegen: %v", err)
			}
			if prozent != 0 {
				t.Errorf("zustand_abwertung_prozent = %d, want 0", prozent)
			}
		})
	})
}
