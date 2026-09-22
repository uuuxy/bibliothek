package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBQueryer interface abstracts pgxpool.Pool and pgx.Tx so that
// the repository can be used both inside and outside of transactions.
type DBQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	// Begin wird für Repository-Methoden gebraucht, die ihre eigene kurze Transaktion
	// aufmachen (z. B. der Inventur-Scan koordiniert per Zeilensperre gegen den
	// Abschluss). Sowohl *pgxpool.Pool als auch pgx.Tx erfüllen das (bei einer Tx ein
	// Savepoint) — die Repos werden ausschließlich mit einem von beidem erzeugt.
	Begin(ctx context.Context) (pgx.Tx, error)
}

// SequenceRepository vergibt die laufenden Ausweisnummern („A-10001").
type SequenceRepository struct {
	db DBQueryer
}

// NewSequenceRepository initializes a new SequenceRepository.
func NewSequenceRepository(db DBQueryer) *SequenceRepository {
	return &SequenceRepository{db: db}
}

// NaechsteAusweisnummer liefert die nächste freie laufende Ausweisnummer (ohne Vorsilbe;
// die gedruckte Form setzt api.AusweisNummer). Der Generator steht seit Migration 136 in
// der Datenbank (ausweis_nummer_start), weil auch der Trigger aktives_konto_hat_ausweis ihn
// braucht: Ein Nummernkreis hat EINEN Zähler — zwei daneben waren der Fehler aus Migration
// 068. Bis dahin rechnete GetNextSequence dieselbe Regel in Go.
//
// Die Regeln: numerisch statt lexikografisch (A-100000 > A-99999), Fallback 10001, Nummern
// mit mehr als 15 Ziffern (verrutschter Scan) werden übergangen. Der Advisory-Lock hält bis
// zum Ende der Transaktion — wer mehrere Nummern braucht (LUSD-Lauf), zieht einmal und zählt
// in derselben Transaktion selbst weiter.
//
// Die Exemplarnummern der Bücher kommen NICHT von hier, sondern aus barcode_seq
// (repository/barcode_vergabe.go).
func (r *SequenceRepository) NaechsteAusweisnummer(ctx context.Context) (int, error) {
	var n int64
	if err := r.db.QueryRow(ctx, `SELECT ausweis_nummer_start()`).Scan(&n); err != nil {
		return 0, fmt.Errorf("nächste ausweisnummer: %w", err)
	}
	return int(n), nil
}
