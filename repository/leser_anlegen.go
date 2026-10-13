package repository

import (
	"context"
	"time"
)

// LeserNeu sind die Werte einer neuen Leserzeile. Klasse und AbgaengerJahr sind bei einem
// Kollegen nil und nicht leer: Ein leerer Text wäre eine Klasse namens „nichts", und die
// Klassenlisten fragen auf NULL.
type LeserNeu struct {
	Barcode       string
	Vorname       string
	Nachname      string
	Klasse        *string
	Geburtsdatum  *time.Time
	AbgaengerJahr *int
	Art           string
}

// LegeLeserAn schreibt eine Leserzeile und liefert ihre Kennung. Geschrieben wird die Tabelle
// leser und nicht die Sicht schueler: Durch die Sicht könnte ein Kollege nicht entstehen. Die
// Paarung von Art und Klasse, Abgangsjahr und Ausweis prüft die Datenbank
// (chk_leser_schueler_pflichtfelder).
func LegeLeserAn(ctx context.Context, db Datenbank, l LeserNeu) (string, error) {
	var id string
	qInsert := `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, art)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := db.QueryRow(ctx, qInsert, l.Barcode, l.Vorname, l.Nachname, l.Klasse, l.Geburtsdatum, l.AbgaengerJahr, l.Art).Scan(&id)
	return id, err
}

// IstSchuelerDubletteVorhanden erkennt einen vorhandenen Schüler an Vorname, Nachname und
// Geburtsdatum, ohne Rücksicht auf Groß- und Kleinschreibung und ohne die Zeilen im Papierkorb.
//
// Ein fehlendes Geburtsdatum ist kein Merkmal einer Dublette: Zwei namensgleiche Schüler ohne
// Geburtsdatum (noch nicht aus der LUSD übernommen) sind nicht dieselbe Person. `geburtsdatum =
// $3` liefert genau das: Ist eine der beiden Seiten NULL, ist der Vergleich NULL, und die Zeile
// zählt nicht als Treffer. Ein coalesce auf ein festes Datum machte namensgleiche Schüler ohne
// Geburtsdatum zu Dubletten; der zweite "Leon Müller" ließe sich dann nicht anlegen.
//
// Verglichen wird in der Normalform suchnorm, derselben, in der der eindeutige Index und der
// LUSD-Schlüssel rechnen: „Müller" und „Mueller" sind ein Mensch. Mit lower() allein rutschte
// die Schreibvariante an der Prüfung vorbei und stünde als zweite Zeile in der Datenbank.
func IstSchuelerDubletteVorhanden(ctx context.Context, db Datenbank, vorname, nachname string, gebdatum *time.Time) (bool, error) {
	var isDuplicate bool
	q := `SELECT EXISTS(SELECT 1 FROM schueler WHERE suchnorm(vorname) = suchnorm($1) AND suchnorm(nachname) = suchnorm($2) AND geburtsdatum = $3::DATE AND deleted_at IS NULL)`
	err := db.QueryRow(ctx, q, vorname, nachname, gebdatum).Scan(&isDuplicate)
	return isDuplicate, err
}

// IstLeserNamensdubletteVorhanden sucht einen aktiven Leser mit demselben Namen — in der
// Normalform suchnorm, also „Müller" wie „Mueller".
//
// Gefragt wird die Tabelle `leser` und über alle Arten: Ein Kollege, der schon als
// Schülerzeile aus dem Altbestand steht, ist derselbe Mensch. Die Prüfung ist bewusst
// grob — sie weist auch zwei echte Namensvettern ab. Das ist der seltenere Fall, und er
// meldet sich sofort; ein stiller zweiter Eintrag meldet sich nie.
func IstLeserNamensdubletteVorhanden(ctx context.Context, db Datenbank, vorname, nachname string) (bool, error) {
	var belegt bool
	q := `SELECT EXISTS(SELECT 1 FROM leser
	       WHERE suchnorm(vorname) = suchnorm($1) AND suchnorm(nachname) = suchnorm($2)
	         AND deleted_at IS NULL)`
	err := db.QueryRow(ctx, q, vorname, nachname).Scan(&belegt)
	return belegt, err
}

// IstAusweisnummerVergeben sagt, ob ein aktiver Leser die Ausweisnummer trägt. Gefragt wird die
// Tabelle leser über alle Arten; gelöschte Leser geben ihre Nummer frei, wie
// uniq_schueler_barcode_active.
func IstAusweisnummerVergeben(ctx context.Context, db Datenbank, barcode string) (bool, error) {
	var vergeben bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM leser WHERE barcode_id = $1 AND deleted_at IS NULL)`,
		barcode).Scan(&vergeben)
	return vergeben, err
}
