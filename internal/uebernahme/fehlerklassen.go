package uebernahme

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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

// werteMuster trifft den Wertteil einer Postgres-DETAIL-Angabe: alles zwischen ")=(" und
// der schließenden Klammer. Bewusst gierig bis zur LETZTEN Klammer: Ein Wert, der selbst
// Klammern enthält, wird damit vollständig ersetzt statt halb stehen zu lassen — im
// Zweifel lieber eine Angabe zu viel geschwärzt als ein Datenwert zu wenig.
var werteMuster = regexp.MustCompile(`\)=\(.*\)`)

// ohneDatenwerte schwärzt den Wert in einer DETAIL-Angabe und behält ihre Form:
// "Key (email)=(x@y.de) already exists." → "Key (email)=(…) already exists."
func ohneDatenwerte(detail string) string {
	return werteMuster.ReplaceAllString(detail, `)=(…)`)
}

// BeschreibeFehler übersetzt einen Postgres-Fehler in eine Zeile, mit der jemand die
// Quelldatei reparieren kann: SQLSTATE, Constraint und betroffene Spalte statt nur der
// nackten Meldung. Bei einer Übernahme von 60.000 Zeilen ist genau das die Arbeit —
// nicht die Laufzeit.
//
// Der WERT aus der DETAIL-Angabe bleibt dabei draußen (ohneDatenwerte). Bis zum
// 23.08.2026 stand er wörtlich drin, und Postgres schreibt dort den Inhalt der Zeile:
//
//	DETAIL: Key (email)=(erika.mustermann@philipp-reis-schule.de) already exists.
//
// Das Ziel dieser Zeile ist `littera_import.log` — eine unverschlüsselte Datei im
// Arbeitsverzeichnis, ohne Frist, ohne Löschregel. Bei der Leser-Übernahme kollidieren
// E-Mail-Adressen und Ausweisnummern regelmäßig; jede Kollision hätte eine echte Adresse
// dort abgelegt. Spalte und Constraint bleiben — sie sagen, WO es klemmt; welcher Wert es
// war, steht in der Quelldatei neben der mitprotokollierten ID.
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
	// Der umschließende Kontext (z. B. der Barcode) geht sonst verloren, weil errors.As
	// bis zum PgError durchgreift.
	return err.Error() + " [" + strings.Join(teile, " | ") + "]"
}
