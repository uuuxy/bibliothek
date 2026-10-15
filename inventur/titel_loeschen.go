package inventur

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// LoescheTitel löscht Titel samt allem, was an ihnen hängt.
//
// Die abhängigen DELETEs (Schadensfälle, Ausleihen, Titel; das Löschen des Titels räumt die
// Exemplare über ON DELETE CASCADE mit) laufen in einer Transaktion. Ohne sie hinterließe ein
// Absturz zwischen zwei Schritten einen bösartigen Halbzustand: Forderungen und Ausleihen
// gelöscht, Titel und Exemplare aber erhalten.
//
// „Alles" heißt wirklich alles, auch gerade verliehene Exemplare. Bräche der Lauf ab, sobald ein
// einziges Exemplar unterwegs ist, ließe sich ein versehentlich importierter Titel nicht
// aufräumen, bis das letzte Buch zurück ist. Das ist eine Entscheidung des Betreibers.
//
// Sie hat einen Preis, und der ist der Grund für protokolliereOffeneAusleihen weiter unten: Das
// Buch liegt bei jemandem zu Hause, und das Programm vergisst es. Wer es zurückbringt, findet
// beim Scannen nichts mehr vor. Deshalb steht jede dabei abgeräumte offene Ausleihe einzeln im
// Protokoll, mit Barcode, Titel und Entleiher, damit die Rückgabe wenigstens nachzuschlagen
// bleibt. Ohne das wäre die Löschung spurlos, und ein spurlos verschwundenes Buch ist ein
// verlorenes Buch.
//
// Der Barcode wird dabei frei; neu vergeben wird er nicht (barcode_seq zählt nur vorwärts), das
// zurückkommende Buch kann also nicht mit einem anderen verwechselt werden.
func (a *TitelAbfragen) LoescheTitel(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("löschen konnte nicht begonnen werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	// Bücher in mehreren Auflagen: Ihre Werke müssen danach noch stimmen.
	// Als erstes — die Sperre der Auflagen kommt vor jeder Zeilensperre (repository.LadeWerkeDerTitel).
	werke, err := repository.LadeWerkeDerTitel(ctx, tx, ids)
	if err != nil {
		return err
	}

	// Alles, was eine Spur braucht, wird in der Transaktion gelesen: Was ein Lesen davor sähe, wäre
	// nicht zwingend das, was fällt.
	//
	// Schnappschüsse der Barcodes aller Exemplare, bevor die Zeilen fallen: Die Auskunft am Tresen
	// findet gelöschte Exemplare nur darüber (Begründung an leseExemplarSnapshots).
	exemplarSnaps, err := leseExemplarSnapshots(ctx, tx, ids)
	if err != nil {
		return err
	}
	// Vormerkungen, Reservierungen und Zuordnungen von Klassensätzen fallen über ON DELETE CASCADE
	// mit dem Titel; die DDL sagt das, der Code soll es auch sagen. Vorher lesen, danach ist es
	// weg.
	wartende, err := repository.LadeWartendeBezuege(ctx, tx, ids)
	if err != nil {
		return err
	}
	lokaleCover, err := repository.LadeLokaleCoverNurDieserTitel(ctx, tx, ids)
	if err != nil {
		return err
	}

	// Zugehörige Datensätze aller Exemplare dieser Titel entfernen, sonst greift der
	// ON DELETE RESTRICT der FKs. Die Reihenfolge (Schadensfall → Ausleihe → Exemplar)
	// erzwingen die RESTRICT-FKs, Atomarität die Tx. Beide Löschbefehle liefern ihre Spur
	// gleich mit (RETURNING): die unbezahlten Forderungen und die laufenden Ausleihen —
	// genau die Zeilen, die fallen, nicht die, die ein Leser kurz davor sah. Was hier
	// verschwindet, steht vor dem Commit im Audit-Log.
	offeneSchaeden, err := loescheSchaedenMitSpur(ctx, tx, ids)
	if err != nil {
		return err
	}
	offene, err := loescheAusleihenMitSpur(ctx, tx, ids)
	if err != nil {
		return err
	}

	ergebnis, err := tx.Exec(ctx, `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if ergebnis.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}
	if err := repository.RaeumeWerkeAuf(ctx, tx, werke); err != nil {
		return err
	}

	// In derselben Transaktion: entweder die Löschung und ihre Spur oder keins von beidem.
	if err := protokolliereWasMitDenTitelnFiel(ctx, tx, offeneSchaeden, wartende, offene, exemplarSnaps); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("löschen konnte nicht abgeschlossen werden: %w", err)
	}

	// Erst nach dem Commit: Die Cover-Dateien kommen nicht zurück, wenn das Löschen in der
	// Datenbank doch scheitert.
	repository.LoescheCoverDateien(lokaleCover)
	return nil
}

// protokolliereWasMitDenTitelnFiel schreibt fest, was das Löschen mitgenommen hat: unbezahlte
// Forderungen, Wartende, laufende Ausleihen und die Barcodes der Exemplare.
func protokolliereWasMitDenTitelnFiel(ctx context.Context, tx pgx.Tx, schaeden []offenerSchaden, wartende []repository.WartenderBezug, ausleihen []offeneAusleihe, exemplare []exemplarSnapshot) error {
	if err := protokolliereOffeneSchaeden(ctx, tx, schaeden); err != nil {
		return err
	}
	if err := repository.ProtokolliereWartendeBezuege(ctx, tx, wartende); err != nil {
		return err
	}
	if err := protokolliereOffeneAusleihen(ctx, tx, ausleihen); err != nil {
		return err
	}
	return protokolliereGeloeschteExemplare(ctx, tx, exemplare)
}
