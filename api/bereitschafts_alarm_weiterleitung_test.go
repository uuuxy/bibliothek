package api

import (
	"context"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/pkg/logger"

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

// Der Start gibt dem Server den Fehlerstand, an dem das Log zählt (main.go), und die Systempflege
// muss denselben bekommen. Bekäme sie keinen, meldete die Seite „nicht erhoben"; bekäme sie einen
// eigenen, zeigte sie dauerhaft keine Fehler.
func TestNewServer_ReichtDenFehlerstandAnDieSystempflege(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	t.Cleanup(mock.Close)
	stand := logger.NewFehlerstand()
	srv := NewServer(&db.Verbindung{Pool: mock}, nil, nil, ServerKonfig{Fehlerstand: stand})
	if got := srv.system().Fehlerstand; got != stand {
		t.Errorf("system().Fehlerstand = %p, erwartet den Fehlerstand des Starts %p", got, stand)
	}
}
