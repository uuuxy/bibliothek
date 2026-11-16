package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// constraintAusweisUeberPersonen ist der Name, unter dem der Trigger eine
// Ausweisnummer ablehnt, die schon eine Person der anderen Tabelle trägt (Schüler ↔ Kollegium).
const constraintAusweisUeberPersonen = "uniq_ausweis_ueber_personen"

// IstAusweisKollision sagt, ob err eine vergebene Ausweisnummer meldet — unter den aktiven
// Schülern oder über Schüler und Kollegium hinweg. Die
// Schreibwege prüfen vorher selbst; greift trotzdem die Datenbank, haben zwei Arbeitsplätze
// gleichzeitig dieselbe Nummer vergeben, und das ist eine Auskunft, kein Serverfehler.
func IstAusweisKollision(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == constraintAusweisUeberPersonen ||
		pgErr.ConstraintName == "uniq_schueler_barcode_active"
}

// ConstraintNummerUeberBuchUndAusweis ist der Name, unter dem der Wächter
// eine Nummer ablehnt, die auf der anderen Seite schon vergeben ist: ein Ausweis, der der
// Barcode eines Buchs wäre, oder ein Buch-Barcode, der der Ausweis eines Lesers wäre.
const ConstraintNummerUeberBuchUndAusweis = "uniq_nummer_ueber_buch_und_ausweis"

// IstNummerBuchOderAusweisKollision sagt, ob err diese Ablehnung meldet. Die Theke löst
// einen Scan ohne Vorsilbe zuerst als Buch auf — dieselbe Nummer für beides ließe einen
// Ausweis still ein Buch laden. Für den Aufrufer ist das eine Auskunft (409), kein 500.
func IstNummerBuchOderAusweisKollision(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == ConstraintNummerUeberBuchUndAusweis
}

// IstExemplarBarcodeVergeben sagt, ob err meldet, dass ein anderes Exemplar die Nummer schon
// trägt (buecher_exemplare.barcode_id ist eindeutig). Für den Aufrufer ist das eine Auskunft
// (409), kein 500.
func IstExemplarBarcodeVergeben(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == "buecher_exemplare_barcode_id_key"
}

// IstLMFNummerAlsBarcode sagt, ob err meldet, dass ein Exemplar eine LMF-Nummer als eigene Nummer
// bekommen sollte (chk_exemplar_barcode_keine_lmf_nummer, Migration 168). Eine LMF-Nummer steht
// nur als altes Etikett in lmf_barcode; für den Aufrufer ist das eine Auskunft, kein 500.
func IstLMFNummerAlsBarcode(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514" &&
		pgErr.ConstraintName == "chk_exemplar_barcode_keine_lmf_nummer"
}
