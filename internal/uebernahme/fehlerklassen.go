package uebernahme

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrZeile markiert einen Fehler, der nur diesen einen Datensatz betrifft und nicht von
// Postgres stammt, etwa ein Freitextfeld, das sich nicht schreiben lässt. Die Fehler von
// Postgres ordnet istZeilenfehler über ihren SQLSTATE ein.
var ErrZeile = errors.New("Datensatzfehler")

// istZeilenfehler sagt, ob ein Fehler nur diesen einen Datensatz betrifft. Dann wird auf den
// Savepoint zurückgerollt und mit dem nächsten weitergemacht; betrifft er die ganze Übernahme,
// bricht sie ab.
//
// An dieser Unterscheidung hängt, ob „übersprungen" im Protokoll stimmt: Ohne sie erschiene ein
// Abbruch der Verbindung bei Titel 300 als 79.700 einzeln übersprungene Zeilen.
func istZeilenfehler(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrZeile) {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || len(pgErr.Code) < 2 {
		return false // Fehler des Netzes oder des Treibers betreffen nicht die Zeile
	}
	switch pgErr.Code[:2] {
	case "22", // Datenausnahme, etwa 22001 Wert zu lang oder 22003 Zahl außerhalb des Bereichs
		"23": // Integritätsverletzung: 23505 doppelte ISBN oder Barcode, 23502 NOT NULL, 23514 CHECK
		return true
	default:
		// 08 Verbindung, 25P02 abgebrochene Transaktion, 40 Deadlock, 53 Ressourcen, 57 Eingriff
		// des Administrators, XX intern: Nichts davon behebt der nächste Datensatz.
		return false
	}
}

// werteMuster trifft den Wert in einer DETAIL-Angabe von Postgres: alles zwischen ")=(" und
// der letzten schließenden Klammer. Gierig bis zur letzten Klammer, damit ein Wert, der selbst
// Klammern enthält, ganz ersetzt wird; lieber eine Angabe zu viel geschwärzt als ein Datenwert
// zu wenig.
var werteMuster = regexp.MustCompile(`\)=\(.*\)`)

// ohneDatenwerte schwärzt den Wert in einer DETAIL-Angabe und behält ihre Form:
// "Key (email)=(x@y.de) already exists." → "Key (email)=(…) already exists."
func ohneDatenwerte(detail string) string {
	return werteMuster.ReplaceAllString(detail, `)=(…)`)
}

// BeschreibeFehler übersetzt einen Fehler von Postgres in eine Zeile, mit der sich die
// Quelldatei berichtigen lässt: SQLSTATE, Constraint und Spalte statt nur der Meldung.
//
// Der Wert aus der DETAIL-Angabe bleibt draußen (ohneDatenwerte). Postgres schreibt dort den
// Inhalt der Zeile, etwa eine E-Mail-Adresse, und die Zeile geht in littera_import.log: eine
// unverschlüsselte Datei im Arbeitsverzeichnis ohne Löschfrist. Spalte und Constraint sagen,
// wo es klemmt; welcher Wert es war, steht in der Quelldatei neben der protokollierten ID.
func BeschreibeFehler(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err.Error()
	}
	teile := []string{fmt.Sprintf("SQLSTATE %s: %s", pgErr.Code, pgErr.Message)}
	if pgErr.ConstraintName != "" {
		teile = append(teile, "Constraint "+pgErr.ConstraintName)
	}
	if pgErr.ColumnName != "" {
		teile = append(teile, "Spalte "+pgErr.ColumnName)
	}
	if pgErr.Detail != "" {
		teile = append(teile, ohneDatenwerte(pgErr.Detail))
	}
	// Der umschließende Zusammenhang, etwa der Barcode, ginge sonst verloren, weil errors.As
	// bis zum PgError durchgreift.
	return err.Error() + " [" + strings.Join(teile, " | ") + "]"
}
