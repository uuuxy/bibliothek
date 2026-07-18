package inventur

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewBookRepository(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	repo := NewBookRepository(mockPool)

	assert.NotNil(t, repo)
	assert.Equal(t, mockPool, repo.db)
}

func TestHandleDbError(t *testing.T) {
	tests := []struct {
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
				ConstraintName: "books_isbn_key",
			},
			expected: ErrDuplicateISBN,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleDbError(tt.inputErr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
