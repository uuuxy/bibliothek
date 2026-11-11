package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// AusweisLayoutSchluessel ist der Schlüssel, unter dem das Ausweis-Design in
// system_einstellungen liegt. Es liegt am Server, damit jeder Arbeitsplatz denselben Stand
// sieht: die Ausleihe vorn wie der Druck im Büro.
const AusweisLayoutSchluessel = "ausweis_layout"

// ErrAusweisLayoutVeraltet meldet, dass das Design seit dem Stand des Rufers an einem anderen
// Platz gespeichert wurde. Geschrieben ist dann nichts.
var ErrAusweisLayoutVeraltet = errors.New("ausweis-design: der stand des rufers ist veraltet")

// sqlAusweisLayoutStand ist der Änderungsstand der Zeile e als Text, auf die Mikrosekunde. Die
// Datenbank schreibt und vergleicht ihn selbst, damit kein Umweg über eine Uhr oder ein
// Zeitformat des Servers eine Stelle verliert.
const sqlAusweisLayoutStand = `to_char(e.aktualisiert_am AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`

// LadeAusweisLayout liest das gespeicherte Ausweis-Design und seinen Änderungsstand, oder
// pgx.ErrNoRows, wenn noch keines gespeichert ist. Der Aufrufer muss einen Lesefehler von „noch
// keines" unterscheiden: Nach einem leeren Wert speichert der Designer seine Vorgaben für alle.
func LadeAusweisLayout(ctx context.Context, db Datenbank) (wert, stand string, err error) {
	err = db.QueryRow(ctx,
		`SELECT e.wert, `+sqlAusweisLayoutStand+` FROM system_einstellungen e WHERE e.schluessel = $1`,
		AusweisLayoutSchluessel).Scan(&wert, &stand)
	return wert, stand, err
}

// SchreibeAusweisLayout legt das Ausweis-Design an oder ersetzt es und liefert den neuen
// Änderungsstand. Es schreibt nur, wenn das gespeicherte Design noch den Stand trägt, den der
// Rufer geladen hat; der leere Stand heißt „noch keines gespeichert". Sonst ist es
// ErrAusweisLayoutVeraltet: Ein Designer, der seit Stunden offen steht, ersetzte mit seiner
// nächsten Änderung sonst das ganze Design, das ein anderer Platz inzwischen gespeichert hat.
//
// Prüfen und Schreiben sind eine Anweisung. Das UPDATE vergleicht die Spalte der Zeile selbst:
// Wartet es auf einen zweiten Platz, prüft Postgres die Bedingung danach an dessen Stand neu.
func SchreibeAusweisLayout(ctx context.Context, db Datenbank, wert, stand string) (string, error) {
	var neu string
	err := db.QueryRow(ctx, `
		WITH ersetzt AS (
			UPDATE system_einstellungen e SET wert = $2, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE e.schluessel = $1 AND `+sqlAusweisLayoutStand+` = $3
			RETURNING `+sqlAusweisLayoutStand+` AS stand
		), angelegt AS (
			INSERT INTO system_einstellungen AS e (schluessel, wert, aktualisiert_am)
			SELECT $1, $2, CURRENT_TIMESTAMP
			WHERE $3 = '' AND NOT EXISTS (SELECT 1 FROM system_einstellungen WHERE schluessel = $1)
			ON CONFLICT (schluessel) DO NOTHING
			RETURNING `+sqlAusweisLayoutStand+` AS stand
		)
		SELECT stand FROM ersetzt UNION ALL SELECT stand FROM angelegt`,
		AusweisLayoutSchluessel, wert, stand).Scan(&neu)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrAusweisLayoutVeraltet
	}
	return neu, err
}
