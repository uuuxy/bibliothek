package repository

import "context"

// SetzeTitelSignatur schreibt die Signatur eines Titels und liefert, was danach in der Zeile
// steht; pgx.ErrNoRows, wenn es den Titel nicht gibt.
func SetzeTitelSignatur(ctx context.Context, db DBQueryer, titelID, signatur string) (string, error) {
	var neu string
	err := db.QueryRow(ctx, `
			UPDATE buecher_titel SET signatur = $2, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
			RETURNING coalesce(signatur, '')
		`, titelID, signatur).Scan(&neu)
	return neu, err
}

// SetzeTitelLernmittel schreibt das Lernmittel-Kennzeichen eines Titels und liefert, was danach
// in der Zeile steht; pgx.ErrNoRows, wenn es den Titel nicht gibt. Fällt das Kennzeichen,
// fällt der Mehrjahresband mit: Die Datenbank lässt ihn nur an einem Lernmittel zu.
func SetzeTitelLernmittel(ctx context.Context, db DBQueryer, titelID string, istLernmittel bool) (bool, error) {
	var neu bool
	err := db.QueryRow(ctx, `
			UPDATE buecher_titel SET ist_lernmittel = $2, mehrjahresband = mehrjahresband AND $2,
			       aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
			RETURNING ist_lernmittel
		`, titelID, istLernmittel).Scan(&neu)
	return neu, err
}
