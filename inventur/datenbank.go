package inventur

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookRepository struct {
	db *pgxpool.Pool
}

var (
	ErrBookNotFound  = errors.New("kein buch mit dieser ID gefunden")
	ErrDuplicateISBN = errors.New("ein buch mit dieser ISBN existiert bereits")
)

func NewBookRepository(db *pgxpool.Pool) *BookRepository {
	return &BookRepository{db: db}
}

func handleDbError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" && pgErr.ConstraintName == "books_isbn_key" {
			return ErrDuplicateISBN
		}
	}
	return err
}
