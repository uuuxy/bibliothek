package uebernahme

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Savepointergebnis sagt, was aus einem Datensatz geworden ist.
//
// Zurueckgerollt trägt den Grund, aus dem der Datensatz NICHT in der Datenbank steht.
// Er wird bewusst zurückgegeben statt hier protokolliert: Nur der Aufrufer kennt den
// Kontext, mit dem die Zeile in der Quelldatei wiederzufinden ist.
type Savepointergebnis struct {
	Uebernommen    bool
	Zurueckgerollt error
}

// ImSavepoint führt fn in einem SAVEPOINT aus.
//
// Der Savepoint ist der Kern dieses Pakets. Ohne ihn ist jedes `continue` in einer
// Schreibschleife wirkungslos: Postgres versetzt die Transaktion beim ersten Fehler in
// den Abbruchzustand (SQLSTATE 25P02), jedes weitere Statement scheitert nur noch daran,
// und der abschließende COMMIT wird zum ROLLBACK. Ein einziger zu langer Titel riss damit
// den gesamten Batch mit — protokolliert als lauter harmlos klingende Einzelmeldungen.
//
// Rückgabe:
//   - (übernommen, nil)      – fn lief durch, der Savepoint ist freigegeben
//   - (zurückgerollt, nil)   – fn meldete einen Datensatzfehler, der Savepoint ist zurück­
//     gerollt und die Transaktion wieder benutzbar; der Aufrufer protokolliert und macht weiter
//   - (leer, err)            – kein Datensatzfehler. Die Übernahme ist zu Ende.
//
// Kosten: eine Subtransaktion je Datensatz. Jenseits von 64 offenen Subtransaktionen je
// Transaktion verliert Postgres den Cache und muss pg_subtrans lesen. Für ein Werkzeug,
// das einmal von Hand auf einer ansonsten ruhenden Datenbank läuft, ist das folgenlos —
// die Savepoints werden zudem sofort wieder freigegeben, nicht gesammelt.
func ImSavepoint(ctx context.Context, tx pgx.Tx, kennung string, fn func(pgx.Tx) error) (Savepointergebnis, error) {
	sp, err := tx.Begin(ctx) // pgx bildet die verschachtelte Transaktion als SAVEPOINT ab
	if err != nil {
		return Savepointergebnis{}, fmt.Errorf("konnte den Savepoint für %s nicht setzen: %w", kennung, err)
	}

	fnErr := fn(sp)
	if fnErr == nil {
		if err := sp.Commit(ctx); err != nil { // RELEASE SAVEPOINT
			return Savepointergebnis{}, fmt.Errorf("konnte den Savepoint für %s nicht freigeben: %w", kennung, err)
		}
		return Savepointergebnis{Uebernommen: true}, nil
	}

	if !IstZeilenfehler(fnErr) {
		return Savepointergebnis{}, fmt.Errorf("abgebrochen bei %s (kein Datensatzfehler): %w", kennung, fnErr)
	}
	if err := sp.Rollback(ctx); err != nil { // ROLLBACK TO SAVEPOINT – hebt 25P02 wieder auf
		return Savepointergebnis{}, fmt.Errorf("konnte den Savepoint für %s nicht zurückrollen: %w", kennung, err)
	}
	return Savepointergebnis{Zurueckgerollt: fnErr}, nil
}
