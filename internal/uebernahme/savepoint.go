package uebernahme

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Savepointergebnis sagt, was aus einem Datensatz geworden ist.
//
// Zurueckgerollt trägt den Grund, aus dem der Datensatz nicht in der Datenbank steht. Er kommt
// zurück und wird nicht hier protokolliert, weil nur der Rufer weiß, womit sich die Zeile in
// der Quelldatei wiederfinden lässt.
type Savepointergebnis struct {
	Uebernommen    bool
	Zurueckgerollt error
}

// ImSavepoint führt fn in einem SAVEPOINT aus.
//
// Ohne Savepoint ist jedes `continue` in einer Schleife von Schreibern wirkungslos: Postgres
// versetzt die Transaktion beim ersten Fehler in den Abbruchzustand (SQLSTATE 25P02), jede
// weitere Anweisung scheitert daran, und aus dem COMMIT am Ende wird ein ROLLBACK. Ein einziger
// zu langer Titel nähme so den ganzen Stapel mit.
//
// Rückgabe:
//   - (übernommen, nil)      – fn lief durch, der Savepoint ist freigegeben
//   - (zurückgerollt, nil)   – fn meldete einen Datensatzfehler, der Savepoint ist zurück­
//     gerollt und die Transaktion wieder benutzbar; der Rufer protokolliert und macht weiter
//   - (leer, err)            – kein Datensatzfehler. Die Übernahme ist zu Ende.
//
// Der Preis ist eine Subtransaktion je Datensatz. Für ein Werkzeug, das einmal von Hand auf
// einer ruhenden Datenbank läuft, fällt er nicht ins Gewicht; die Savepoints werden sofort
// wieder freigegeben und nicht gesammelt.
func ImSavepoint(ctx context.Context, tx pgx.Tx, kennung string, fn func(pgx.Tx) error) (Savepointergebnis, error) {
	savepoint, err := tx.Begin(ctx) // pgx bildet die verschachtelte Transaktion als SAVEPOINT ab
	if err != nil {
		return Savepointergebnis{}, fmt.Errorf("konnte den Savepoint für %s nicht setzen: %w", kennung, err)
	}

	fnErr := fn(savepoint)
	if fnErr == nil {
		if err := savepoint.Commit(ctx); err != nil { // RELEASE SAVEPOINT
			return Savepointergebnis{}, fmt.Errorf("konnte den Savepoint für %s nicht freigeben: %w", kennung, err)
		}
		return Savepointergebnis{Uebernommen: true}, nil
	}

	if !istZeilenfehler(fnErr) {
		return Savepointergebnis{}, fmt.Errorf("abgebrochen bei %s (kein Datensatzfehler): %w", kennung, fnErr)
	}
	if err := savepoint.Rollback(ctx); err != nil { // ROLLBACK TO SAVEPOINT hebt 25P02 wieder auf
		return Savepointergebnis{}, fmt.Errorf("konnte den Savepoint für %s nicht zurückrollen: %w", kennung, err)
	}
	return Savepointergebnis{Zurueckgerollt: fnErr}, nil
}
