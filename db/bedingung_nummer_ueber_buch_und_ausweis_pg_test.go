package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Eine Nummer ist entweder ein Buch oder ein Ausweis, nie beides.
//
// Die Theke löst einen Scan ohne Vorsilbe zuerst als Buch auf (verarbeiteOhneVorsilbe,
// ErkenneScan): Trägt ein Leser die Nummer eines Exemplars, lädt sein Ausweis das Buch, und
// niemand sieht warum. Die Littera-Übernahme prüft das für ihren Lauf auch selbst
// (internal/littera/schreiber_personen.go); die Regel an beiden Tabellen bremst auch, wer von
// Hand eine Ausweisnummer einträgt oder ein Buch umetikettiert.
//
// Blind ist der Test für das gleichzeitige Vergeben: Ohne Advisory-Lock je Nummer (ein Lock je
// Zeile sprengte bei einem großen Import die Sperrtabelle) kommen zwei Arbeitsplätze, die im
// selben Augenblick dieselbe neue Nummer vergeben, einer als Ausweis, einer als Buch, beide durch.
const (
	constraintNummer = "uniq_nummer_ueber_buch_und_ausweis"
	insExemplar      = `INSERT INTO buecher_exemplare (titel_id, barcode_id)
	                    VALUES ((SELECT id FROM buecher_titel WHERE titel = 'Nummernprobe'), $1)`
)

func TestNummerIstBuchOderAusweis(t *testing.T) {
	pool := pgTestPool(t)
	titel := func(tx pgx.Tx) {
		erwarteErfolg(t, tx, "Titel", `INSERT INTO buecher_titel (titel) VALUES ('Nummernprobe')`)
	}

	t.Run("Barcode eines Buchs ist für einen Leser vergeben", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titel(tx)
			erwarteErfolg(t, tx, "Exemplar", insExemplar, "NB-1")
			erwarteVerletzteBedingung(t, tx, constraintNummer, insLeserSchueler, "NB-1")
			erwarteVerletzteBedingung(t, tx, constraintNummer, insLeserLehrkraft, "NB-1")
		})
	})

	t.Run("Ausweisnummer ist für ein Exemplar vergeben", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titel(tx)
			erwarteErfolg(t, tx, "Schüler", insLeserSchueler, "NB-2")
			erwarteVerletzteBedingung(t, tx, constraintNummer, insExemplar, "NB-2")
		})
	})

	t.Run("Umetikettieren auf eine Ausweisnummer, Ausweis ändern auf einen Buch-Barcode", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titel(tx)
			erwarteErfolg(t, tx, "Exemplar", insExemplar, "NB-3B")
			erwarteErfolg(t, tx, "Lehrkraft", insLeserLehrkraft, "NB-3A")
			erwarteVerletzteBedingung(t, tx, constraintNummer,
				`UPDATE buecher_exemplare SET barcode_id = $1 WHERE barcode_id = 'NB-3B'`, "NB-3A")
			erwarteVerletzteBedingung(t, tx, constraintNummer,
				`UPDATE leser SET barcode_id = $1 WHERE barcode_id = 'NB-3A'`, "NB-3B")
		})
	})

	// Ein Leser im Papierkorb ist an der Theke unsichtbar; seine Nummer darf ein Buch tragen.
	// Kommt er zurück, stößt er an, wie beim Zwilling unter den Lesern.
	t.Run("gelöschter Leser gibt die Nummer frei, Wiederherstellen stößt an", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titel(tx)
			erwarteErfolg(t, tx, "gelöschter Schüler", insLeserGeloescht, "NB-4")
			erwarteErfolg(t, tx, "Exemplar mit der freigegebenen Nummer", insExemplar, "NB-4")
			erwarteVerletzteBedingung(t, tx, constraintNummer,
				`UPDATE leser SET deleted_at = NULL WHERE barcode_id = $1`, "NB-4")
		})
	})

	// Ein ausgesondertes Exemplar trägt seinen Barcode weiter (die Auskunft an der Theke findet
	// es darüber); erst das endgültige Löschen gibt die Nummer frei.
	t.Run("ausgesondertes Exemplar hält seine Nummer, gelöschtes gibt sie frei", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titel(tx)
			erwarteErfolg(t, tx, "Exemplar", insExemplar, "NB-5")
			erwarteErfolg(t, tx, "aussondern",
				`UPDATE buecher_exemplare SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'VERLUST' WHERE barcode_id = $1`, "NB-5")
			erwarteVerletzteBedingung(t, tx, constraintNummer, insLeserSchueler, "NB-5")
			erwarteErfolg(t, tx, "endgültig löschen", `DELETE FROM buecher_exemplare WHERE barcode_id = $1`, "NB-5")
			erwarteErfolg(t, tx, "Schüler mit der freigegebenen Nummer", insLeserSchueler, "NB-5")
		})
	})

	t.Run("Gegenprobe: verschiedene Nummern und Leser ohne Ausweis", func(t *testing.T) {
		inTx(t, pool, func(tx pgx.Tx) {
			titel(tx)
			erwarteErfolg(t, tx, "Exemplar", insExemplar, "NB-6B")
			erwarteErfolg(t, tx, "Schüler", insLeserSchueler, "NB-6S")
			erwarteErfolg(t, tx, "Lehrkraft ohne Ausweis", insLeserLehrkraft, nil)
			// Ein Update, das die Nummer nicht anfasst, darf nicht an der eigenen stoßen.
			erwarteErfolg(t, tx, "Exemplar ohne Nummernwechsel",
				`UPDATE buecher_exemplare SET ist_ausleihbar = false WHERE barcode_id = $1`, "NB-6B")
			erwarteErfolg(t, tx, "Leser ohne Nummernwechsel",
				`UPDATE leser SET vorname = 'Neu' WHERE barcode_id = $1`, "NB-6S")
		})
	})
}

// Ein Leser ohne Ausweis (NULL) ist kein Buch, und die Prüfung darf nicht an NULL scheitern.
func TestNummerIstBuchOderAusweis_NullIstKeineNummer(t *testing.T) {
	pool := pgTestPool(t)
	inTx(t, pool, func(tx pgx.Tx) {
		erwarteErfolg(t, tx, "Lehrkraft ohne Ausweis", insLeserLehrkraft, nil)
		var n int
		if err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM leser WHERE nachname = 'Rer' AND barcode_id IS NULL`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%d Lehrkräfte ohne Ausweis, erwartet 1", n)
		}
	})
}
