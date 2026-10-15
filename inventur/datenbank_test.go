package inventur

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
)

func TestNewTitelAbfragen(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	abfragen := NewTitelAbfragen(mockPool)

	assert.NotNil(t, abfragen)
	assert.Equal(t, mockPool, abfragen.pool)
}

func TestUebersetzeDatenbankfehler(t *testing.T) {
	faelle := []struct {
		name     string
		inputErr error
		expected error
	}{
		{
			name:     "nil error",
			inputErr: nil,
			expected: nil,
		},
		{
			name:     "generic error",
			inputErr: errors.New("some generic error"),
			expected: errors.New("some generic error"),
		},
		{
			name: "duplicate ISBN error",
			inputErr: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "buecher_titel_isbn_key",
			},
			expected: ErrISBNDoppelt,
		},
		{
			name: "other pg error code",
			inputErr: &pgconn.PgError{
				Code:           "23502",
				ConstraintName: "some_other_key",
			},
			expected: &pgconn.PgError{
				Code:           "23502",
				ConstraintName: "some_other_key",
			},
		},
		{
			name: "duplicate error but different constraint",
			inputErr: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "some_other_key",
			},
			expected: &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "some_other_key",
			},
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			ergebnis := uebersetzeDatenbankfehler(f.inputErr)
			assert.Equal(t, f.expected, ergebnis)
		})
	}
}
