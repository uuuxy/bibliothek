package inventur

import (
	"bibliothek/db"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// TitelAbfragen ist der Datenbankzugang des Inventur-Moduls (Titel, Klassensätze,
// Metadaten). Nicht zu verwechseln mit *repository.KatalogAbfragen — jenes bedient den
// Ausleihbetrieb.
type TitelAbfragen struct {
	pool db.Pool
}

// Fachliche Fehler, die die Tür unterscheiden muss: Bedienfehler (HTTP 400 oder 409), keine
// Störung; die Tür darf daraus keinen 500 machen. Einen unbekannten Titel meldet
// repository.ErrTitelNichtGefunden.
var (
	errISBNDoppelt = errors.New("ein buch mit dieser ISBN existiert bereits")
	// errDubletteTitel: derselbe Titel desselben Autors steht ohne ISBN schon im
	// Katalog — die Dublettenkontrolle ohne Nummer.
	errDubletteTitel = errors.New("dieser titel steht ohne ISBN bereits im katalog")
	// errISBNFormat: Die ISBN wurde geändert, und der neue Wert hat nicht die Form einer ISBN.
	errISBNFormat = errors.New("ungültiges ISBN-Format")
	// errAutorGeleert: Eine Änderung nennt keinen Autor, der Titel trägt aber einen.
	errAutorGeleert = errors.New("der autor des titels darf nicht geleert werden")
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
			return errISBNDoppelt
		}
	}
	return err
}
