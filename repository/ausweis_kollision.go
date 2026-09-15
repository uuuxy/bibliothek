package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ConstraintAusweisUeberPersonen ist der Name, unter dem der Trigger aus Migration 118 eine
// Ausweisnummer ablehnt, die schon eine Person der anderen Tabelle trägt (Schüler ↔ Kollegium).
const ConstraintAusweisUeberPersonen = "uniq_ausweis_ueber_personen"

// IstAusweisKollision sagt, ob err eine vergebene Ausweisnummer meldet — unter den aktiven
// Schülern (Migration 049) oder über Schüler und Kollegium hinweg (Migration 118). Die
// Schreibwege prüfen vorher selbst; greift trotzdem die Datenbank, haben zwei Arbeitsplätze
// gleichzeitig dieselbe Nummer vergeben, und das ist eine Auskunft, kein Serverfehler.
func IstAusweisKollision(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == ConstraintAusweisUeberPersonen ||
		pgErr.ConstraintName == "uniq_schueler_barcode_active"
}
