package inventur

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
)

func TestNewTitelAbfragen(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mockPool.Close()

	abfragen := NewTitelAbfragen(mockPool)
	if abfragen == nil || abfragen.pool != mockPool {
		t.Errorf("NewTitelAbfragen(mockPool) = %+v, erwartet die Abfragen am übergebenen Pool", abfragen)
	}
}

func TestUebersetzeDatenbankfehler(t *testing.T) {
	// Was keine doppelte ISBN meldet, kommt unverändert zurück: dasselbe Objekt, nicht nur ein
	// gleich aussehendes.
	allgemein := errors.New("some generic error")
	andererCode := &pgconn.PgError{Code: "23502", ConstraintName: "some_other_key"}
	andereRegel := &pgconn.PgError{Code: "23505", ConstraintName: "some_other_key"}
	faelle := []struct {
		name    string
		eingabe error
		want    error
	}{
		{name: "kein Fehler", eingabe: nil, want: nil},
		{name: "anderer Fehler", eingabe: allgemein, want: allgemein},
		{
			name:    "doppelte ISBN",
			eingabe: &pgconn.PgError{Code: "23505", ConstraintName: "buecher_titel_isbn_key"},
			want:    errISBNDoppelt,
		},
		{name: "anderer Code", eingabe: andererCode, want: andererCode},
		{name: "Verstoß gegen eine andere Regel", eingabe: andereRegel, want: andereRegel},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := uebersetzeDatenbankfehler(f.eingabe); got != f.want {
				t.Errorf("uebersetzeDatenbankfehler(%v) = %v, erwartet %v", f.eingabe, got, f.want)
			}
		})
	}
}
