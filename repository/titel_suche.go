package repository

import (
	"context"
)

// FindeTitel führt eine sprachenspezifische Volltextsuche über Buchtitel und Autoren durch und
// vergleicht daneben Teile des Texts und die ISBN; die Treffer stehen nach ihrem Rang, höchstens
// 50, und nur Titel mit einem Exemplar (SQLTitelHatExemplar). Der Teilstring-Vergleich nimmt den
// Suchtext in der Form, in der Titeltexte gespeichert sind (TiteltextNormalform).
func (a *KatalogAbfragen) FindeTitel(ctx context.Context, suchtext string) ([]Titel, error) {
	suchtext = TiteltextNormalform(suchtext)
	query := `
		SELECT 
			b.id, coalesce(b.titel, ''), coalesce(b.untertitel, ''), coalesce(b.autor, ''), coalesce(b.isbn, ''), coalesce(b.verlag, ''), coalesce(b.erscheinungsjahr, 0), coalesce(b.cover_url, ''), coalesce(b.medientyp, ''), coalesce(b.signatur, ''), coalesce(b.auflage, ''), b.mehrjahresband, b.ist_lernmittel, b.erstellt_am, b.aktualisiert_am, coalesce(b.erweiterte_eigenschaften, '{}'::jsonb),
			` + SQLBestandGesamt + `, ` + SQLBestandVerfuegbar + `, ` + SQLBestandImZulauf + `
		FROM buecher_titel b
		WHERE (
			b.search_vector @@ plainto_tsquery('german', $1::text) 
			OR b.titel ILIKE '%' || $1::text || '%'
			OR b.autor ILIKE '%' || $1::text || '%'
			OR regexp_replace(coalesce(b.isbn, ''), '[- ]', '', 'g') ILIKE '%' || regexp_replace($1::text, '[- ]', '', 'g') || '%'
			OR replace(b.isbn, '-', '') = replace($1::text, '-', '')
			OR ` + SQLSuchtextIstISBN("b", "$1::text") + `
		) AND ` + SQLTitelHatExemplar("b") + `
		ORDER BY ts_rank(b.search_vector, plainto_tsquery('german', $1::text)) DESC, b.titel ASC
		LIMIT 50
	`
	rows, err := a.db.Query(ctx, query, suchtext)
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

// FindeTitelUnscharf führt eine Wildcard-Suche für Auto-Vervollständigungen oder ungenaue Anfragen aus,
// über Titel, Autor, ISBN und Signatur, höchstens limit Treffer, und liefert zusätzlich die
// Gesamtzahl der Treffer (nicht nur die des Limits).
//
// Wie bei der Schülersuche muss jedes Token irgendeine Spalte treffen, statt die
// komplette Eingabe eine einzelne: "rowling harry" findet damit den Titel über
// Autor + Titel, was der frühere Ganzstring-Vergleich nicht konnte. Die
// ISBN-Gleichheit bleibt als eigener Zweig bestehen — dort wird die ganze,
// bindestrichbereinigte Eingabe verglichen, nicht tokenweise.
func (a *KatalogAbfragen) FindeTitelUnscharf(ctx context.Context, suchtext string, limit int) ([]Titel, int, error) {
	tokens := suchTokens(suchtext)
	if len(tokens) == 0 {
		return nil, 0, nil
	}

	query := `
		WITH tokens AS (
			SELECT suchnorm(t) AS norm, lower(t) AS roh FROM unnest($1::text[]) AS t
		)
		SELECT
			id, coalesce(titel, ''), coalesce(untertitel, ''), coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0), coalesce(cover_url, ''), coalesce(medientyp, ''), coalesce(signatur, ''), coalesce(auflage, ''), mehrjahresband, ist_lernmittel, erstellt_am, aktualisiert_am, coalesce(erweiterte_eigenschaften, '{}'::jsonb),
			` + SQLBestandGesamt + `, ` + SQLBestandVerfuegbar + `, ` + SQLBestandImZulauf + `,
			count(*) OVER () AS treffer_gesamt
		FROM buecher_titel b
		WHERE (
			-- Index-Anker: erstes Token als direkte LIKE-Bedingung, damit der Planer
			-- die Ausdrucks-Indizes per BitmapOr zieht. Die
			-- Ausdrücke müssen buchstäblich denen der Indizes entsprechen — ein
			-- coalesce() darum herum macht den Zweig unindexierbar und kippt die
			-- ganze Abfrage in den Seq Scan (siehe LeserAbfragen.FindeUnscharf).
			   suchnorm(b.titel)    LIKE '%' || suchnorm($4::text) || '%'
			OR suchnorm(b.autor)    LIKE '%' || suchnorm($4::text) || '%'
			OR lower(b.isbn)        LIKE '%' || lower($4::text) || '%'
			OR lower(b.signatur)    LIKE '%' || lower($4::text) || '%'
			OR replace(b.isbn, '-', '') = replace($2::text, '-', '')
			OR ` + SQLSuchtextIstISBN("b", "$2::text") + `
		  )
		  AND (
			   replace(b.isbn, '-', '') = replace($2::text, '-', '')
			OR ` + SQLSuchtextIstISBN("b", "$2::text") + `
			OR (
				SELECT bool_and(
					   suchnorm(coalesce(b.titel, ''))     LIKE '%' || tokens.norm || '%'
					OR suchnorm(coalesce(b.autor, ''))     LIKE '%' || tokens.norm || '%'
					OR lower(coalesce(b.isbn, ''))         LIKE '%' || tokens.roh || '%'
					OR lower(coalesce(b.signatur, ''))     LIKE '%' || tokens.roh || '%')
				FROM tokens
			   )
		  )
		  AND ` + SQLTitelHatExemplar("b") + `
		ORDER BY
			(SELECT count(*) FROM tokens WHERE suchnorm(coalesce(b.titel, '')) LIKE tokens.norm || '%') DESC,
			b.titel ASC
		LIMIT $3
	`
	rows, err := a.db.Query(ctx, query, tokens, suchtext, limit, tokens[0])
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
