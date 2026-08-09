package inventur

import (
	"bibliothek/db"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// BookRepository ist der Datenbankzugang des Inventur-Moduls (Titel, Klassensätze,
// Metadaten). Nicht zu verwechseln mit repository.BookRepository — jenes bedient den
// Ausleihbetrieb.
type BookRepository struct {
	db db.PgxPoolIface
}

// Fachliche Fehler, die der Aufrufer unterscheiden können muss: Beide sind
// Bedienfehler (HTTP 400/404), keine Störung — der Handler darf daraus keinen 500 machen.
var (
	ErrBookNotFound  = errors.New("kein buch mit dieser ID gefunden")
	ErrDuplicateISBN = errors.New("ein buch mit dieser ISBN existiert bereits")
)

// NewBookRepository bindet das Inventur-Repository an einen Verbindungspool.
func NewBookRepository(db db.PgxPoolIface) *BookRepository {
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
