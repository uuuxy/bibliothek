package repository

import "context"

// TitelZuISBN sind die Angaben eines Titels, die das Bestellen über eine ISBN braucht. ISBN
// ist die Nummer, wie der Katalog sie trägt.
type TitelZuISBN struct {
	TitelID       string
	Titel         string
	Autor         string
	Verlag        string
	CoverURL      string
	Signatur      string
	IstLernmittel bool
	ISBN          string
}

// FindeTitelZuISBN sucht den Titel, der die ISBN trägt, in jeder Schreibweise und in beiden
// Längen; pgx.ErrNoRows, wenn keiner sie trägt.
func FindeTitelZuISBN(ctx context.Context, db DBQueryer, isbn string) (TitelZuISBN, error) {
	var t TitelZuISBN
	err := db.QueryRow(ctx, `
		SELECT id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,''), coalesce(signatur,''), ist_lernmittel, isbn
		FROM buecher_titel WHERE `+SQLTitelTraegtISBN("", "$1")+` LIMIT 1
	`, isbn).Scan(&t.TitelID, &t.Titel, &t.Autor, &t.Verlag, &t.CoverURL, &t.Signatur, &t.IstLernmittel, &t.ISBN)
	return t, err
}

// TitelMetadaten sind die Angaben aus dem Nachschlagen einer ISBN, aus denen ein Titel
// entsteht. Fach ist die Bezeichnung einer vorhandenen Sachgruppe oder leer; ein Listenpreis
// von nil heißt: nicht ermittelbar.
type TitelMetadaten struct {
	Titel            string
	Autor            string
	ISBN             string
	Verlag           string
	Erscheinungsjahr *int
	CoverURL         string
	Fach             string
	Untertitel       string
	Listenpreis      *float64
}

// LegeTitelAusMetadatenAn legt den Titel zur ISBN an. Steht er schon da, gewinnt bei
// Untertitel, Listenpreis und Cover, was erfasst ist; Titel, Autor, Verlag und Jahr folgen den
// neuen Angaben. Signatur und Lernmittel-Kennzeichen bleiben unberührt.
func LegeTitelAusMetadatenAn(ctx context.Context, db DBQueryer, m TitelMetadaten) (TitelZuISBN, error) {
	var t TitelZuISBN
	err := db.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, cover_url, subject, untertitel, listenpreis)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF(btrim($8), ''), $9)
		ON CONFLICT (isbn) DO UPDATE
			SET titel      = EXCLUDED.titel,
			    autor      = EXCLUDED.autor,
			    verlag     = EXCLUDED.verlag,
			    erscheinungsjahr = EXCLUDED.erscheinungsjahr,
			    cover_url  = COALESCE(NULLIF(EXCLUDED.cover_url, ''), buecher_titel.cover_url),
			    untertitel = COALESCE(NULLIF(buecher_titel.untertitel, ''), EXCLUDED.untertitel),
			    listenpreis = COALESCE(buecher_titel.listenpreis, EXCLUDED.listenpreis),
			    aktualisiert_am = CURRENT_TIMESTAMP
		RETURNING id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,''), coalesce(signatur,''), ist_lernmittel, isbn
	`, m.Titel, m.Autor, m.ISBN, m.Verlag, m.Erscheinungsjahr, m.CoverURL, m.Fach,
		m.Untertitel, m.Listenpreis).
		Scan(&t.TitelID, &t.Titel, &t.Autor, &t.Verlag, &t.CoverURL, &t.Signatur, &t.IstLernmittel, &t.ISBN)
	return t, err
}
