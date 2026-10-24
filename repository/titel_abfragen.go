package repository

import (
	"bibliothek/db"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// TitelAbfragen sind die Abfragen des Medienkatalogs: Titelliste, Buchmaske, Listenimport, Cover,
// Klassensätze und die Schulbücher im Portal. KatalogAbfragen bedient den Ausleihbetrieb an der
// Theke.
type TitelAbfragen struct {
	pool db.Pool
}

// Fachliche Fehler, die die Tür unterscheiden muss: Bedienfehler (HTTP 400 oder 409), keine
// Störung; die Tür darf daraus keinen 500 machen. Einen unbekannten Titel meldet
// ErrTitelNichtGefunden.
var (
	ErrISBNDoppelt = errors.New("ein buch mit dieser ISBN existiert bereits")
	// errDubletteTitel: derselbe Titel desselben Autors steht ohne ISBN schon im
	// Katalog — die Dublettenkontrolle ohne Nummer.
	errDubletteTitel = errors.New("dieser titel steht ohne ISBN bereits im katalog")
	// ErrISBNFormat: Die ISBN wurde geändert, und der neue Wert hat nicht die Form einer ISBN.
	ErrISBNFormat = errors.New("ungültiges ISBN-Format")
	// ErrAutorGeleert: Eine Änderung nennt keinen Autor, der Titel trägt aber einen.
	ErrAutorGeleert = errors.New("der autor des titels darf nicht geleert werden")
)

// NewTitelAbfragen bindet das Inventur-Repository an einen Verbindungspool.
func NewTitelAbfragen(pool db.Pool) *TitelAbfragen {
	return &TitelAbfragen{pool: pool}
}

func uebersetzeDatenbankfehler(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" && pgErr.ConstraintName == "buecher_titel_isbn_key" {
			return ErrISBNDoppelt
		}
	}
	return err
}
