package repository

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBQueryer interface abstracts pgxpool.Pool and pgx.Tx so that
// the repository can be used both inside and outside of transactions.
type DBQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// SequenceRepository provides methods to calculate sequential strings,
// such as auto-incrementing barcodes with prefixes (e.g., "B-10001").
type SequenceRepository struct {
	db DBQueryer
}

// NewSequenceRepository initializes a new SequenceRepository.
func NewSequenceRepository(db DBQueryer) *SequenceRepository {
	return &SequenceRepository{db: db}
}

// sequenceStartNum ist die erste vergebene laufende Nummer, wenn noch kein Barcode
// mit dem Präfix existiert (Format z. B. "S-10001" / "B-10001").
const sequenceStartNum = 10001

// advisoryLockKey leitet aus Tabelle+Spalte einen stabilen 64-Bit-Schlüssel für den
// transaktionalen Advisory-Lock ab. Verschiedene Sequenzen (z. B. Schüler "S-" vs.
// Exemplare "B-") bekommen verschiedene Schlüssel und blockieren sich gegenseitig
// nicht.
func advisoryLockKey(tableName, colName string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(tableName + "\x00" + colName))
	return int64(h.Sum64())
}

// GetNextSequence fetches the highest existing barcode with a given prefix,
// increments the numeric part by one, and returns it.
// Default fallback starts at 10001 if no prior entry is found.
//
// Note: Since table and column names cannot be parameterized in Postgres,
// they are securely formatted using Sprintf. We trust the inputs as they are
// developer-defined constants (never user-provided).
//
// Zwei Fehler dieser Funktion sind hier zentral behoben, weil sie ALLE laufenden
// Barcodes (Schüler "S-", Exemplare "B-") speist:
//
//  1. Lexikografischer Kollaps: Das frühere ORDER BY <spalte> DESC sortierte den
//     Barcode als String. Damit gilt 'B-99999' > 'B-100000' (die '9' schlägt die '1').
//     Ab dem Übergang auf sechsstellige Nummern lieferte die Query dauerhaft 'B-99999'
//     als Maximum; das System versuchte endlos, erneut 'B-100000' anzulegen, und lief
//     jedes Mal in den UNIQUE-Constraint. Fix: Das Zahlensuffix NACH dem Präfix wird
//     numerisch verglichen (Cast auf bigint), nicht lexikografisch.
//
//  2. Race-Condition: "Höchsten Wert lesen und in Go +1 rechnen" ist ohne Sperre nicht
//     atomar — zwei gleichzeitige Anlagen lesen denselben Maximalwert und erzeugen
//     denselben nächsten Barcode; einer läuft in eine Constraint-Verletzung (harter
//     500er). Fix: pg_advisory_xact_lock serialisiert die Vergabe pro Sequenz. Der Lock
//     wird in der Transaktion des Aufrufers gehalten, bis dieser den zugehörigen INSERT
//     committet — der zweite Aufrufer blockiert so lange und liest danach den bereits
//     erhöhten Maximalwert. Läuft die Funktion (Preview-Endpunkt) ohne umschließende
//     Transaktion auf dem Pool, wird der Lock am Statement-Ende sofort wieder frei —
//     dort wird nichts eingefügt, also ist keine Serialisierung nötig.
func (r *SequenceRepository) GetNextSequence(ctx context.Context, tableName, colName, prefix string) (int, error) {
	// Das Zahlensuffix beginnt direkt hinter dem Präfix (1-indizierte Startposition).
	suffixStart := len(prefix) + 1

	// Der Advisory-Lock steht auf der IMMER vorhandenen Lock-Zeile (LEFT JOIN von links),
	// damit er auch bei leerer Tabelle (allererster Barcode) sicher genommen wird — ein
	// CROSS JOIN mit leerer Tabelle würde die Lock-Zeile nie auswerten.
	query := fmt.Sprintf(`
		SELECT coalesce(max(substr(t.%[1]s, $2)::bigint), 0)
		FROM (SELECT pg_advisory_xact_lock($3)) AS _lock
		LEFT JOIN %[2]s t
		       ON t.%[1]s LIKE $1
		      AND substr(t.%[1]s, $2) ~ '^[0-9]+$'
	`, colName, tableName)

	var lastNum int64
	err := r.db.QueryRow(ctx, query,
		prefix+"%",                          // $1
		suffixStart,                         // $2
		advisoryLockKey(tableName, colName), // $3
	).Scan(&lastNum)
	if err != nil {
		return 0, fmt.Errorf("failed to query next sequence: %w", err)
	}

	if lastNum > 0 {
		return int(lastNum) + 1, nil
	}
	return sequenceStartNum, nil
}
