package repository

import "context"

// AusweisLayoutSchluessel ist der Schlüssel, unter dem das Ausweis-Design in
// system_einstellungen liegt. Es liegt am Server, damit jeder Arbeitsplatz denselben Stand
// sieht: die Ausleihe vorn wie der Druck im Büro.
const AusweisLayoutSchluessel = "ausweis_layout"

// LadeAusweisLayout liest das gespeicherte Ausweis-Design, oder pgx.ErrNoRows, wenn noch
// keines gespeichert ist. Der Aufrufer muss einen Lesefehler von „noch keines" unterscheiden:
// Nach einem leeren Wert speichert der Designer seine Vorgaben für alle.
func LadeAusweisLayout(ctx context.Context, db DBQueryer) (string, error) {
	var wert string
	err := db.QueryRow(ctx,
		`SELECT wert FROM system_einstellungen WHERE schluessel = $1`, AusweisLayoutSchluessel).Scan(&wert)
	return wert, err
}

// SpeichereAusweisLayout legt das Ausweis-Design an oder ersetzt es.
func SpeichereAusweisLayout(ctx context.Context, db DBQueryer, wert string) error {
	_, err := db.Exec(ctx,
		`INSERT INTO system_einstellungen (schluessel, wert, aktualisiert_am)
			 VALUES ($1, $2, CURRENT_TIMESTAMP)
			 ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert, aktualisiert_am = CURRENT_TIMESTAMP`,
		AusweisLayoutSchluessel, wert)
	return err
}
