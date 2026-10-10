package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// BestellsucheTreffer ist ein Titel des eigenen Katalogs in der Bestellsuche.
type BestellsucheTreffer struct {
	ID            string
	Titel         string
	Autor         string
	ISBN          string
	Verlag        string
	CoverURL      string
	Signatur      string
	IstLernmittel bool
	// Bestand zählt die Exemplare, die nicht ausgesondert sind, auch die bestellten.
	Bestand int
}

// SucheTitelZumBestellen durchsucht den eigenen Katalog über den Volltext und über ein Stück von
// Titel, Autor oder ISBN und liefert höchstens 50 Treffer, den besten zuerst. Anders als die
// Suche der Theke nennt sie auch einen Titel ohne Exemplar: Der lässt sich bestellen. Der
// Suchtext geht in die Form, in der Titeltexte gespeichert sind (TiteltextNormalform).
func SucheTitelZumBestellen(ctx context.Context, db DBQueryer, suchtext string) ([]BestellsucheTreffer, error) {
	rows, err := db.Query(ctx, `
		WITH matched_titels AS (
			SELECT t.id, t.titel, t.autor, t.isbn, t.verlag, t.cover_url, t.signatur, t.ist_lernmittel, t.search_vector
			FROM buecher_titel t
			WHERE
				t.search_vector @@ plainto_tsquery('german', $1)
				OR t.titel ILIKE '%' || $1 || '%'
				OR t.autor ILIKE '%' || $1 || '%'
				OR regexp_replace(coalesce(t.isbn, ''), '[- ]', '', 'g') ILIKE '%' || regexp_replace($1, '[- ]', '', 'g') || '%'
				OR replace(t.isbn, '-', '') = replace($1, '-', '')
				OR `+SQLSuchtextIstISBN("t", "$1")+`
			ORDER BY ts_rank(t.search_vector, plainto_tsquery('german', $1)) DESC, t.titel ASC
			LIMIT 50
		)
		SELECT mt.id, mt.titel, coalesce(mt.autor, ''), coalesce(mt.isbn, ''), coalesce(mt.verlag, ''),
		       `+sqlCoverOderDNB("mt")+`,
		       coalesce(mt.signatur, ''), mt.ist_lernmittel,
		       COALESCE(e.current_stock, 0)
		FROM matched_titels mt
		LEFT JOIN LATERAL (
			SELECT COUNT(id)::int AS current_stock
			FROM buecher_exemplare
			WHERE titel_id = mt.id AND ist_ausgesondert = false
		) e ON true
		ORDER BY ts_rank(mt.search_vector, plainto_tsquery('german', $1)) DESC, mt.titel ASC
	`, TiteltextNormalform(suchtext))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var treffer []BestellsucheTreffer
	for rows.Next() {
		var t BestellsucheTreffer
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag, &t.CoverURL, &t.Signatur, &t.IstLernmittel, &t.Bestand); err != nil {
			return nil, err
		}
		treffer = append(treffer, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return treffer, nil
}

// ISBNsImKatalog liefert von den genannten ISBNs die, die ein Titel trägt. Beide Seiten stehen
// in der Normalform: Der Aufrufer reicht sie so (isbnutil.Normalform), die Spalte geht durch
// isbn_normalform, weil eine Altzeile ihre ISBN noch in anderer Schreibweise tragen kann.
func ISBNsImKatalog(ctx context.Context, db DBQueryer, isbns []string) (map[string]struct{}, error) {
	vorhanden := make(map[string]struct{})
	if len(isbns) == 0 {
		return vorhanden, nil
	}
	rows, err := db.Query(ctx, "SELECT isbn_normalform(isbn) FROM buecher_titel WHERE isbn_normalform(isbn) = ANY($1)", isbns)
	if err != nil {
		return nil, err
	}
	gefunden, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	for _, isbn := range gefunden {
		vorhanden[isbn] = struct{}{}
	}
	return vorhanden, nil
}
