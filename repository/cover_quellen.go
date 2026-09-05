package repository

import (
	"context"

	"bibliothek/db"
)

// CoverQuellenFuerISBN liefert die gespeicherten Cover-Adressen aller Titel mit dieser
// ISBN und ob es überhaupt einen solchen Titel gibt. Die ISBN steht in buecher_titel mit
// oder ohne Trennstriche — verglichen wird die bereinigte Form (isbnutil.CleanISBN).
//
// Für den Cover-Proxy (api/cover_quelle_bindung.go): Er lädt nur Adressen, die der Katalog
// für diese ISBN kennt oder die sich aus ihr herleiten lassen — sonst bestimmte der
// erste unangemeldete Aufrufer, welches Bild unter einer Schulbuch-ISBN liegt.
func CoverQuellenFuerISBN(ctx context.Context, pool db.PgxPoolIface, isbnSauber string) (quellen []string, imKatalog bool, err error) {
	rows, err := pool.Query(ctx,
		`SELECT COALESCE(cover_url, '') FROM buecher_titel WHERE regexp_replace(isbn, '[- ]', '', 'g') = $1`,
		isbnSauber)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, false, err
		}
		imKatalog = true
		if u != "" {
			quellen = append(quellen, u)
		}
	}
	return quellen, imKatalog, rows.Err()
}
