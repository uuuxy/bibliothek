package inventur

import (
	"context"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbSchreiber ist der minimale Datenbankzugriff der Schreibpfade dieses Pakets. Pool
// (db.Pool) und pgx.Tx erfüllen ihn beide — der Sammelimport arbeitet
// innerhalb seiner Transaktion.
type dbSchreiber interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// StelleFaecherSicher trägt unbekannte Fächer in die Systematik ein, bevor ein Schreibweg sie in
// buecher_titel.subject schreibt (ein Fremdschlüssel). Die Funktion steht in repository, damit
// auch die Importe dort sie erreichen; hier bleibt der Name für die Schreibwege dieses Pakets.
func StelleFaecherSicher(ctx context.Context, datenbank dbSchreiber, faecher []string) (map[string]string, error) {
	return repository.StelleFaecherSicher(ctx, datenbank, faecher)
}
