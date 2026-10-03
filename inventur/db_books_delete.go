package inventur

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// DeleteBooks löscht Titel samt allem, was an ihnen hängt.
//
// Die abhängigen DELETEs (Schadensfälle, Ausleihen, Titel — der Titel-Delete räumt die
// Exemplare per ON DELETE CASCADE mit) laufen in EINER Transaktion. Ohne sie hinterließ
// ein Crash zwischen zwei Schritten einen bösartigen Halbzustand: Gebühren- und
// Ausleihhistorie gelöscht, Buch und Exemplare aber erhalten.
//
// „Alles" heißt seit dem 23.08.2026 wirklich alles — auch AKTUELL VERLIEHENE Exemplare.
// Vorher brach der Lauf ab, sobald ein einziges Exemplar unterwegs war; ein versehentlich
// importierter Titel liess sich dann nicht mehr aufräumen, bis das letzte Buch zurück
// war. Das ist eine bewusste Entscheidung des Betreibers.
//
// Sie hat einen Preis, und der ist der Grund für protokolliereOffeneAusleihen weiter
// unten: Das Buch liegt physisch bei jemandem zu Hause, und das System vergisst es. Wer
// es zurückbringt, findet beim Scannen nichts mehr vor. Deshalb wird JEDE dabei
// abgeräumte offene Ausleihe einzeln im Audit-Log festgehalten — mit Barcode, Titel und
// Entleiher —, damit die Rückgabe später wenigstens nachschlagbar bleibt. Ohne das wäre
// die Löschung spurlos, und ein spurlos verschwundenes Buch ist ein verlorenes Buch.
//
// Der Barcode wird dabei frei; neu vergeben wird er nicht (barcode_seq zählt nur
// vorwärts), das zurückkommende Buch kann also nicht mit einem anderen verwechselt werden.
func (repo *BookRepository) DeleteBooks(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("löschen konnte nicht begonnen werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Bücher in mehreren Auflagen (docs/OFFEN.md 4.18): Ihre Werke müssen danach noch stimmen.
	// Als erstes — die Sperre der Auflagen kommt vor jeder Zeilensperre (repository.WerkeDerTitel).
	werke, err := repository.WerkeDerTitel(ctx, tx, ids)
	if err != nil {
		return err
	}

	// Alles, was eine Spur braucht, wird IN der Transaktion gelesen. Bis zum 21.09.2026
	// standen drei Leser vor dem Begin (OFFEN.md 5.5): Was sie sahen, war nicht
	// zwingend das, was fiel.
	//
	// Barcode-Snapshots ALLER Exemplare, bevor die Zeilen fallen — die Tresen-Auskunft
	// findet gelöschte Exemplare nur darüber (Begründung an leseExemplarSnapshots).
	exemplarSnaps, err := leseExemplarSnapshots(ctx, tx, ids)
	if err != nil {
		return err
	}
	// Vormerkungen, Klassensatz-Reservierungen und Klassensatz-Zuordnungen fallen per
	// ON DELETE CASCADE mit dem Titel — die DDL sagt das, der Code sagte es nicht
	// (Frage 12 „Gegenrichtung Schema", 06.09.2026). Vorher lesen, danach ist es weg.
	wartende, err := repository.LeseWartendeBezuege(ctx, tx, ids)
	if err != nil {
		return err
	}
	localCovers, err := repository.LokaleCoverNurDieserTitel(ctx, tx, ids)
	if err != nil {
		return err
	}

	// Zugehörige Datensätze ALLER Exemplare dieser Titel entfernen, sonst greift der
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

	result, err := tx.Exec(ctx, `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
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
	repository.LoescheCoverDateien(localCovers)
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
