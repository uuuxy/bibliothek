package repository

import (
	"context"
	"strings"
)

// FindeTitel sucht mit der Titelsuche des Programms (sqlTitelTrifft) und liefert höchstens 50
// Titel mit einem Exemplar (sqlTitelHatExemplar); vorn steht, was der Volltext am besten trifft.
func (a *KatalogAbfragen) FindeTitel(ctx context.Context, suchtext string) ([]Titel, error) {
	trifft, args := sqlTitelTrifft("b", suchtext, 2)
	query := `
		SELECT 
			b.id, coalesce(b.titel, ''), coalesce(b.untertitel, ''), coalesce(b.autor, ''), coalesce(b.isbn, ''), coalesce(b.verlag, ''), coalesce(b.erscheinungsjahr, 0), coalesce(b.cover_url, ''), coalesce(b.medientyp, ''), coalesce(b.signatur, ''), coalesce(b.auflage, ''), b.mehrjahresband, b.ist_lernmittel, b.erstellt_am, b.aktualisiert_am, coalesce(b.erweiterte_eigenschaften, '{}'::jsonb),
			` + SQLBestandGesamt + `, ` + SQLBestandVerfuegbar + `, ` + SQLBestandImZulauf + `
		FROM buecher_titel b
		WHERE ` + trifft + ` AND ` + sqlTitelHatExemplar("b") + `
		ORDER BY ts_rank(b.search_vector, plainto_tsquery('german', $1::text)) DESC, b.titel COLLATE deutsch ASC
		LIMIT 50
	`
	rows, err := a.db.Query(ctx, query, append([]any{TiteltextNormalform(suchtext)}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var treffer []Titel
	for rows.Next() {
		t, err := leseTitelMitBestand(rows)
		if err != nil {
			return nil, err
		}
		treffer = append(treffer, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return treffer, nil
}

// FindeTitelUnscharf sucht für die Liste beim Tippen mit der Titelsuche des Programms
// (sqlTitelTrifft), höchstens limit Titel mit einem Exemplar, und liefert zusätzlich die
// Gesamtzahl der Treffer (nicht nur die des Limits). Vorn steht, wessen Titel mit den meisten
// eingegebenen Wörtern beginnt; suchtext beginnt mit dem normalisierten Titel, und ein Wort ohne
// Zeilenumbruch reicht nicht über ihn hinaus.
func (a *KatalogAbfragen) FindeTitelUnscharf(ctx context.Context, suchtext string, limit int) ([]Titel, int, error) {
	woerter := strings.Fields(suchtext)
	if len(woerter) == 0 {
		return nil, 0, nil
	}
	trifft, args := sqlTitelTrifft("b", suchtext, 3)
	// MATERIALIZED: Sonst rechnet Postgres suchnorm der Wörter für jeden Treffer neu, bei einem
	// Buchstaben über 10.000-mal.
	query := `
		WITH tokens AS MATERIALIZED (
			SELECT suchnorm(t) AS norm FROM unnest($1::text[]) AS t
		)
		SELECT
			id, coalesce(titel, ''), coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(cover_url, ''), coalesce(medientyp, ''), coalesce(signatur, ''), coalesce(auflage, ''), mehrjahresband, ist_lernmittel, erstellt_am, aktualisiert_am, coalesce(erweiterte_eigenschaften, '{}'::jsonb),
			` + SQLBestandGesamt + `, ` + SQLBestandVerfuegbar + `, ` + SQLBestandImZulauf + `,
			count(*) OVER () AS treffer_gesamt
		FROM buecher_titel b
		WHERE ` + trifft + ` AND ` + sqlTitelHatExemplar("b") + `
		ORDER BY
			(SELECT count(*) FROM tokens WHERE b.suchtext LIKE ` + sqlLikeMuster("tokens.norm") + ` || '%') DESC,
			b.titel COLLATE deutsch ASC
		LIMIT $2
	`
	rows, err := a.db.Query(ctx, query, append([]any{woerter, limit}, args...)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var treffer []Titel
	gesamt := 0
	for rows.Next() {
		var zeilenGesamt int
		t, err := leseTitelMitBestand(rows, &zeilenGesamt)
		if err != nil {
			return nil, 0, err
		}
		gesamt = zeilenGesamt
		treffer = append(treffer, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return treffer, gesamt, nil
}

// leseTitelMitBestand liest die feste Spaltenliste eines Titels und danach die Bestandszahlen.
// Getrennt von leseTitelMitZusatz, weil die übrigen Abfragen die Zahlen nicht mitliefern: Ein
// Titel mit nil-Bestand sagt „nicht gezählt", nicht „keine da".
func leseTitelMitBestand(row scanner, zusatz ...any) (*Titel, error) {
	var gesamt, verfuegbar, imZulauf int
	t, err := leseTitelMitZusatz(row, append([]any{&gesamt, &verfuegbar, &imZulauf}, zusatz...)...)
	if err != nil {
		return nil, err
	}
	t.Bestand = &gesamt
	t.Verfuegbar = &verfuegbar
	t.ImZulauf = &imZulauf
	return t, nil
}
