// Package inventur ist der Medienkatalog mit eigenen Türen und eigener Datenbankschicht: Titel
// lesen, anlegen, ändern und löschen, der Import aus Excel, die Abfrage über die ISBN, die
// Cover und die hochgeladenen Dateien (docs/ARCHITEKTUR.md 5.2.3).
package inventur

import (
	"bibliothek/db"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// TitelAbfragen ist der Datenbankzugang des Inventur-Moduls (Titel, Klassensätze,
// Metadaten). Nicht zu verwechseln mit repository.KatalogAbfragen — jenes bedient den
// Ausleihbetrieb.
type TitelAbfragen struct {
	pool db.Pool
}

// Fachliche Fehler, die der Aufrufer unterscheiden können muss: Beide sind
// Bedienfehler (HTTP 400/404), keine Störung; die Tür darf daraus keinen 500 machen.
var (
	ErrTitelNichtGefunden = errors.New("kein buch mit dieser ID gefunden")
	ErrISBNDoppelt        = errors.New("ein buch mit dieser ISBN existiert bereits")
	// ErrDubletteTitel: derselbe Titel desselben Autors steht ohne ISBN schon im
	// Katalog — die Dublettenkontrolle ohne Nummer.
	ErrDubletteTitel = errors.New("dieser titel steht ohne ISBN bereits im katalog")
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
