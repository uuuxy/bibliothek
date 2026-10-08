package repository

import (
	"context"
	"time"
)

// MailVorlage ist eine Zeile der Tabelle mail_vorlagen.
type MailVorlage struct {
	ID, Typ, Betreff, TextBody string
	UpdatedAt                  time.Time
}

// ListeMailVorlagen liefert alle Vorlagen nach ihrem Typ geordnet.
func ListeMailVorlagen(ctx context.Context, db DBQueryer) ([]MailVorlage, error) {
	rows, err := db.Query(ctx, "SELECT id, typ, betreff, text_body, updated_at FROM mail_vorlagen ORDER BY typ ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vorlagen []MailVorlage
	for rows.Next() {
		var v MailVorlage
		if err := rows.Scan(&v.ID, &v.Typ, &v.Betreff, &v.TextBody, &v.UpdatedAt); err == nil {
			vorlagen = append(vorlagen, v)
		}
	}
	return vorlagen, rows.Err()
}

// AendereMailVorlage setzt Betreff und Text einer Vorlage und nennt ihren Typ und welches der
// beiden Felder sich dabei geändert hat: Das Protokoll führt die Namen der geänderten Felder,
// nicht den Wortlaut. Der Stand davor wird in derselben Anweisung gesperrt gelesen. Gibt es
// die Vorlage nicht, kommt pgx.ErrNoRows zurück.
func AendereMailVorlage(ctx context.Context, db DBQueryer, id, betreff, text string) (typ string, betreffNeu, textNeu bool, err error) {
	err = db.QueryRow(ctx, `
			WITH alt AS (SELECT betreff, text_body FROM mail_vorlagen WHERE id = $3 FOR UPDATE)
			UPDATE mail_vorlagen v SET betreff = $1, text_body = $2
			  FROM alt
			 WHERE v.id = $3
			RETURNING v.typ, alt.betreff IS DISTINCT FROM $1, alt.text_body IS DISTINCT FROM $2
		`, betreff, text, id).Scan(&typ, &betreffNeu, &textNeu)
	return typ, betreffNeu, textNeu, err
}

// LadeBestellVorlage liest Betreff und Text der Vorlage für die Bestellung beim Händler.
func LadeBestellVorlage(ctx context.Context, db DBQueryer) (betreff, text string, err error) {
	err = db.QueryRow(ctx, "SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'BESTELLUNG_HAENDLER'").Scan(&betreff, &text)
	return betreff, text, err
}

// LadeMahnVorlage liest Betreff und Text der Vorlage für den Mahnbrief an die Eltern.
func LadeMahnVorlage(ctx context.Context, db DBQueryer) (betreff, text string, err error) {
	err = db.QueryRow(ctx, "SELECT betreff, text_body FROM mail_vorlagen WHERE typ = 'MAHNUNG_ELTERN'").Scan(&betreff, &text)
	return betreff, text, err
}
