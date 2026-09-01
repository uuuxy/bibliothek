package inventur

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"bibliothek/db"
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

	offene, err := repo.leseOffeneAusleihen(ctx, ids)
	if err != nil {
		return err
	}

	localCovers, err := repo.sammleLokaleCoverPfade(ctx, ids)
	if err != nil {
		return err
	}

	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("löschen konnte nicht begonnen werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Barcode-Snapshots ALLER Exemplare, bevor die Zeilen fallen — die Tresen-Auskunft
	// findet gelöschte Exemplare nur darüber (Begründung an leseExemplarSnapshots).
	exemplarSnaps, err := leseExemplarSnapshots(ctx, tx, ids)
	if err != nil {
		return err
	}

	// Zugehörige Datensätze ALLER Exemplare dieser Titel entfernen, sonst greift der
	// ON DELETE RESTRICT der FKs. Reihenfolge erzwingen die RESTRICT-FKs, Atomarität die Tx.
	if _, err := tx.Exec(ctx, "DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))", ids); err != nil {
		return fmt.Errorf("failed to delete damage records for titles: %w", err)
	}
	// Ohne Einschränkung auf rueckgabe_am: Auch die laufenden Ausleihen gehen mit, sonst
	// hielte ihr ON DELETE RESTRICT das Exemplar fest und der ganze Lauf scheiterte an
	// einer FK-Verletzung. Was hier verschwindet, steht vorher im Audit-Log.
	if _, err := tx.Exec(ctx, "DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = ANY($1::uuid[]))", ids); err != nil {
		return fmt.Errorf("failed to delete loans for titles: %w", err)
	}

	result, err := tx.Exec(ctx, `DELETE FROM buecher_titel WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return fmt.Errorf("bücher konnten nicht gelöscht werden: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	// In derselben Transaktion: Entweder die Löschung UND ihre Spur, oder keins von beidem.
	if err := protokolliereOffeneAusleihen(ctx, tx, offene); err != nil {
		return err
	}
	if err := protokolliereGeloeschteExemplare(ctx, tx, exemplarSnaps); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("löschen konnte nicht abgeschlossen werden: %w", err)
	}

	// Erst nach dem Commit: die lokalen Cover-Dateien sind unwiederbringlich, das darf
	// NICHT geschehen, solange das DB-Löschen noch scheitern (zurückrollen) könnte.
	loescheLokaleCoverDateien(localCovers)
	return nil
}

// sammleLokaleCoverPfade liefert die lokal gespeicherten Cover-Pfade (/uploads/...)
// der angegebenen Titel, damit sie nach dem Löschen entfernt werden können.
func (repo *BookRepository) sammleLokaleCoverPfade(ctx context.Context, ids []string) ([]string, error) {
	coverRows, err := repo.db.Query(ctx, "SELECT cover_url FROM buecher_titel WHERE id = ANY($1::uuid[]) AND cover_url LIKE '/uploads/%'", ids)
	if err != nil {
		return nil, fmt.Errorf("cover-dateien konnten nicht ermittelt werden: %w", err)
	}
	defer coverRows.Close()

	localCovers := make([]string, 0)
	for coverRows.Next() {
		var coverURL string
		if scanErr := coverRows.Scan(&coverURL); scanErr != nil {
			return nil, fmt.Errorf("cover-pfade konnten nicht gelesen werden: %w", scanErr)
		}
		localCovers = append(localCovers, coverURL)
	}
	if rowsErr := coverRows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("cover-pfade konnten nicht iteriert werden: %w", rowsErr)
	}
	return localCovers, nil
}

// loescheLokaleCoverDateien entfernt die lokalen Cover-Dateien (best-effort; Fehler
// werden ignoriert, da der DB-Datensatz bereits gelöscht ist).
func loescheLokaleCoverDateien(localCovers []string) {
	for _, coverURL := range localCovers {
		if !strings.HasPrefix(coverURL, "/uploads/") {
			continue
		}
		name := filepath.Base(coverURL)
		if name == "" || name == "." || name == "/" {
			continue
		}
		// loescheUploadDatei arbeitet über os.Root (uploads_pfad.go) — die Einhegung auf
		// das Upload-Verzeichnis macht das Betriebssystem, nicht mehr das filepath.Base
		// oben. Deshalb steht hier kein #nosec mehr: Es gibt kein os.Remove, das gosec
		// beanstanden könnte.
		_ = loescheUploadDatei(name) //nolint:errcheck // Best-Effort-Aufräumen nach dem DB-Löschen
	}
}
