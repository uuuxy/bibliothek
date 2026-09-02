package inventur

import (
	"context"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbSchreiber ist der minimale Datenbankzugriff der Schreibpfade dieses Pakets. Pool
// (db.PgxPoolIface) und pgx.Tx erfüllen ihn beide — der Sammelimport arbeitet
// innerhalb seiner Transaktion.
type dbSchreiber interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// StelleFaecherSicher registriert unbekannte Fächer in der Systematik, bevor ein
// Schreibpfad sie in buecher_titel.subject einträgt (FK seit Migration 078). Die
// Funktion lebt seit dem 02.09.2026 im repository-Paket, damit auch die Importe
// dort sie erreichen; hier bleibt der Name für die Schreibpfade dieses Pakets.
func StelleFaecherSicher(ctx context.Context, db dbSchreiber, faecher []string) (map[string]string, error) {
	return repository.StelleFaecherSicher(ctx, db, faecher)
}
