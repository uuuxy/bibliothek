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
func LegeLeserAn(ctx context.Context, db DBQueryer, l LeserNeu) (string, error) {
	var id string
	qInsert := `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, art)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := db.QueryRow(ctx, qInsert, l.Barcode, l.Vorname, l.Nachname, l.Klasse, l.Geburtsdatum, l.AbgaengerJahr, l.Art).Scan(&id)
	return id, err
}

// SchuelerDubletteVorhanden erkennt einen bereits existierenden Schüler anhand von
// Vor-/Nachname und Geburtsdatum (case-insensitive, ohne soft-gelöschte Datensätze).
//
// Ein fehlendes Geburtsdatum ist bewusst KEIN Duplikat-Kriterium: Zwei namensgleiche
// Schüler ohne (noch nicht aus der LUSD übernommenes) Geburtsdatum sind nicht
// automatisch dieselbe Person. Nur bei beidseitig bekanntem, identischem Geburtsdatum
// liegt ein echtes Duplikat vor. `geburtsdatum = $3` liefert genau das: Ist eine der
// beiden Seiten NULL, ist der Vergleich SQL-NULL ("nicht gleich"), die Zeile zählt nicht
// als Treffer. Vorher stülpte coalesce beiden Seiten '1900-01-01' über und machte damit
// namensgleiche Schüler OHNE Geburtsdatum fälschlich zu Duplikaten (Zwillings-Blockade):
// der zweite "Leon Müller" ohne Geburtsdatum konnte gar nicht angelegt werden.
//
// Seit Migration 108 vergleicht die Prüfung in der Normalform suchnorm — derselben, in
// der der Unique-Index und der LUSD-Schlüssel rechnen: „Müller" und „Mueller" sind ein
// Mensch. Vorher nur lower(): Die Schreibvariante rutschte an der Prüfung vorbei und
// stand als zweite Zeile in der Datenbank.
func SchuelerDubletteVorhanden(ctx context.Context, db DBQueryer, vorname, nachname string, gebdatum *time.Time) (bool, error) {
	var isDuplicate bool
	q := `SELECT EXISTS(SELECT 1 FROM schueler WHERE suchnorm(vorname) = suchnorm($1) AND suchnorm(nachname) = suchnorm($2) AND geburtsdatum = $3::DATE AND deleted_at IS NULL)`
	err := db.QueryRow(ctx, q, vorname, nachname, gebdatum).Scan(&isDuplicate)
	return isDuplicate, err
}

// LeserNamensdubletteVorhanden sucht einen aktiven Leser mit demselben Namen — in der
// Normalform suchnorm, also „Müller" wie „Mueller".
//
// Gefragt wird die TABELLE `leser` und über ALLE Arten: Ein Kollege, der schon als
// Schülerzeile aus dem Altbestand steht, ist derselbe Mensch. Die Prüfung ist bewusst
// grob — sie weist auch zwei echte Namensvettern ab. Das ist der seltenere Fall, und er
// meldet sich sofort; ein stiller zweiter Eintrag meldet sich nie.
func LeserNamensdubletteVorhanden(ctx context.Context, db DBQueryer, vorname, nachname string) (bool, error) {
	var belegt bool
	q := `SELECT EXISTS(SELECT 1 FROM leser
	       WHERE suchnorm(vorname) = suchnorm($1) AND suchnorm(nachname) = suchnorm($2)
	         AND deleted_at IS NULL)`
	err := db.QueryRow(ctx, q, vorname, nachname).Scan(&belegt)
	return belegt, err
}

// AusweisnummerVergeben sagt, ob ein aktiver Leser die Ausweisnummer trägt. Gefragt wird die
// Tabelle leser über alle Arten; gelöschte Leser geben ihre Nummer frei, wie
// uniq_schueler_barcode_active.
func AusweisnummerVergeben(ctx context.Context, db DBQueryer, barcode string) (bool, error) {
	var vergeben bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM leser WHERE barcode_id = $1 AND deleted_at IS NULL)`,
		barcode).Scan(&vergeben)
	return vergeben, err
}
