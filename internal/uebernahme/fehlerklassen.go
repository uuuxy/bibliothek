package uebernahme

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrZeile markiert einen Fehler, der eindeutig nur diesen einen Datensatz betrifft und
// nicht von Postgres stammt (etwa ein nicht serialisierbares Freitextfeld). Postgres-
// Fehler werden stattdessen über ihren SQLSTATE eingeordnet, siehe IstZeilenfehler.
var ErrZeile = errors.New("Datensatzfehler")

// IstZeilenfehler entscheidet, ob ein Fehler nur diesen einen Datensatz betrifft — dann
// wird auf den Savepoint zurückgerollt und mit dem nächsten weitergemacht — oder die
// gesamte Übernahme, dann wird sofort abgebrochen.
//
// Diese Unterscheidung ist die Voraussetzung dafür, dass „übersprungen" überhaupt wahr
// sein kann. Ohne sie erschiene ein Verbindungsabbruch bei Titel 300 als 79.700 einzelne
// Übersprungen-Zeilen im Protokoll: formal vollständig, inhaltlich eine Lüge.
func IstZeilenfehler(err error) bool {
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
		return false // Netzwerk- oder Treiberfehler: nicht zeilenbezogen
	}
	switch pgErr.Code[:2] {
	case "22", // Datenausnahme – z. B. 22001 Wert zu lang, 22003 Zahl außerhalb des Bereichs
		"23": // Integritätsverletzung – 23505 doppelte ISBN/Barcode, 23502 NOT NULL, 23514 CHECK
		return true
	default:
		// 08 Verbindung, 25P02 vergiftete Transaktion, 40 Deadlock, 53 Ressourcen,
		// 57 Administrator-Eingriff, XX intern — nichts davon repariert der nächste Datensatz.
		return false
	}
}

// BeschreibeFehler übersetzt einen Postgres-Fehler in eine Zeile, mit der jemand die
// Quelldatei reparieren kann: SQLSTATE, Constraint und betroffene Spalte statt nur der
// nackten Meldung. Bei einer Übernahme von 60.000 Zeilen ist genau das die Arbeit —
// nicht die Laufzeit.
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
		teile = append(teile, pgErr.Detail)
	}
	// Der umschließende Kontext (z. B. der Barcode) geht sonst verloren, weil errors.As
	// bis zum PgError durchgreift.
	return err.Error() + " [" + strings.Join(teile, " | ") + "]"
}
