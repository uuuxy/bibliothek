package api

import (
	"context"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

// abfrageZaehler zählt die Abfragen nach einer einzelnen Zeile, die an den Pool gehen.
type abfrageZaehler struct {
	pgxmock.PgxPoolIface
	abfragen int
}

func (p *abfrageZaehler) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	p.abfragen++
	return p.PgxPoolIface.QueryRow(ctx, sql, args...)
}

// Der Start ruft den Bereitschafts-Alarm am Server (main.go), und der Server reicht ihn an die
// Türen der Systempflege weiter. Bliebe der Aufruf dort liegen, fiele die tägliche Warnmail aus,
// ohne dass ein anderer Test es merkt. Die Umgebung „test" hält den Alarm vor dem Versand an;
// dass er lief, zeigt die Lage, die er vorher aus der Datenbank sammelt.
func TestBereitschaftsAlarm_ServerReichtAnDieSystempflegeWeiter(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	pool := &abfrageZaehler{PgxPoolIface: mock}
	srv := &Server{Basis: tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	srv.BereitschaftsAlarm(t.Context())

	if pool.abfragen == 0 {
		t.Fatal("der Alarm hat keine Lage gesammelt: Der Server reicht den Aufruf nicht weiter")
	}
}
