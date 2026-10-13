package testdaten

import (
	"context"
	"errors"
	"strings"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// EinstellungenLesen ist das Muster der Abfrage, mit der jede Tür die Einstellungen als Ganzes
// liest (repository.EinstellungenUeber).
const EinstellungenLesen = "SELECT schluessel, wert FROM system_einstellungen"

// ErrLesefehler ist der Fehler, den PoolMitLesefehler für die gestörte Abfrage liefert.
var ErrLesefehler = errors.New("testfehler: verbindung zur datenbank abgerissen")

// PoolMitLesefehler lässt jede Abfrage scheitern, deren SQL das Muster trägt, als wäre die
// Verbindung abgerissen. Alles andere läuft am Pool darunter: Ein Test stellt so nach, was eine
// Tür tut, wenn genau ein Lesen misslingt.
type PoolMitLesefehler struct {
	db.PgxPoolIface
	Muster string
}

// Query liefert für das Muster den Fehler und reicht jede andere Abfrage weiter.
func (p PoolMitLesefehler) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if strings.Contains(sql, p.Muster) {
		return nil, ErrLesefehler
	}
	return p.PgxPoolIface.Query(ctx, sql, args...) //nolint:rowserrcheck // reicht die Zeilen weiter, der Aufrufer prüft sie
}
