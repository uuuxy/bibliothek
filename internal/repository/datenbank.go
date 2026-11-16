package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Datenbank ist, woran eine Abfrage läuft: der Pool oder eine Transaktion. Eine Abfrage, die
// beides annimmt, lässt sich allein und innerhalb einer Transaktion des Rufers nutzen.
type Datenbank interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	// Begin brauchen Abfragen, die eine eigene kurze Transaktion öffnen, etwa der Scan der
	// Inventur, der sich über eine Zeilensperre mit dem Abschluss abstimmt. Der Pool und eine
	// Transaktion erfüllen es beide; an einer Transaktion ist es ein Savepoint.
	Begin(ctx context.Context) (pgx.Tx, error)
}
