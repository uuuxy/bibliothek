package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// KatalogTreffer ist ein Titel in der Trefferliste eines Katalogs für Leser: Titelangaben
// und Zählwerte, keine Ausleih- und keine Personendaten.
type KatalogTreffer struct {
	ID, Titel, Autor, ISBN, CoverURL string
	Verfuegbar, Gesamt, ImZulauf     int
}

// KatalogSuche ist eine Anfrage an SucheImKatalog.
type KatalogSuche struct {
	// Sichtbar ist das Prädikat über buecher_titel als `bt`, das festlegt, welche Titel der
	// Katalog zeigt: OeffentlichSichtbar("bt") ohne Anmeldung, KollegiumSichtbar("bt") in
	// „Mein Portal".
	Sichtbar     string
	Suchtext     string
	SchlagwortID string
	Grenze       int
}

// KollegiumSichtbar ist das Prädikat des Katalogs in „Mein Portal": jeder Titel mit einem
// Exemplar, das nicht ausgesondert ist — im Regal, verliehen oder bestellt. Lernmittel
// gehören dazu: Das Kollegium reserviert dort die Klassensätze, und ohne sie fände die
// Suche das Biologiebuch der 8 nicht, das im Bestand steht.
func KollegiumSichtbar(titelAlias string) string {
	return SQLTitelHatExemplar(titelAlias)
}

// SucheImKatalog liefert die Titel eines Katalogs alphabetisch, höchstens s.Grenze, und die
// Zahl aller Treffer vor der Kappung. Ohne Suchtext und ohne Schlagwort ist die Antwort leer.
//
// Der öffentliche Katalog und der des Kollegiums suchen über diese eine Abfrage und
// unterscheiden sich allein in s.Sichtbar. Von den Ausleihen liest sie nur, ob eine läuft;
// kein Wert einer Ausleihe und kein Leser erreicht die Antwort.
//
// Der Suchtext geht zweimal in die Abfrage: für die Volltextsuche, und mit maskierten
// LIKE-Jokern für die Teilstring-Vergleiche. Ein nacktes „%" träfe sonst den ganzen Bestand.
// Beide Male in der Form, in der Titeltexte und Schlagworte gespeichert sind
// (TiteltextNormalform).
func SucheImKatalog(ctx context.Context, q DBQueryer, s KatalogSuche) ([]KatalogTreffer, int, error) {
	s.Suchtext = TiteltextNormalform(s.Suchtext)
	if s.Suchtext == "" && s.SchlagwortID == "" {
		return []KatalogTreffer{}, 0, nil
	}
	bedingungen := []string{s.Sichtbar}
	args := []any{}
	if s.Suchtext != "" {
		args = append(args, s.Suchtext, maskiereLikeJoker(s.Suchtext))
		bedingungen = append(bedingungen, `(bt.search_vector @@ plainto_tsquery('german', $1)
		   OR bt.titel ILIKE '%' || $2 || '%'
		   OR bt.autor ILIKE '%' || $2 || '%'
		   OR regexp_replace(coalesce(bt.isbn, ''), '[- ]', '', 'g') ILIKE '%' || regexp_replace($2, '[- ]', '', 'g') || '%'
		   OR `+SQLSuchtextIstISBN("bt", "$1")+`
		   OR `+SQLTitelUeberSchlagwort("bt", "$2")+`)`)
	}
	if s.SchlagwortID != "" {
		args = append(args, s.SchlagwortID)
		bedingungen = append(bedingungen, SQLTitelMitSchlagwort("bt", fmt.Sprintf("$%d", len(args))))
	}

	rows, err := q.Query(ctx, fmt.Sprintf(`
		SELECT bt.id, bt.titel, COALESCE(bt.autor, ''), COALESCE(bt.isbn, ''),
		       COALESCE(bt.cover_url, ''),
		       COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
		       COUNT(e.id) FILTER (WHERE %s) AS gesamt,
		       %s AS im_zulauf,
		       count(*) OVER () AS treffer
		FROM buecher_titel bt
		LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
		LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		WHERE %s
		GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.cover_url
		ORDER BY bt.titel
		LIMIT %d`,
		SQLExemplarImBestand, SQLFilterImZulauf, strings.Join(bedingungen, " AND "), s.Grenze), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("katalog durchsuchen: %w", err)
	}
	defer rows.Close()

	treffer := make([]KatalogTreffer, 0)
	gesamt := 0
	for rows.Next() {
		var t KatalogTreffer
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.CoverURL,
			&t.Verfuegbar, &t.Gesamt, &t.ImZulauf, &gesamt); err != nil {
			return nil, 0, fmt.Errorf("katalogtreffer lesen: %w", err)
		}
		treffer = append(treffer, t)
	}
	// Ein Abbruch mitten in der Liste ist ein Fehler: Ein Teil der Treffer sähe aus wie alle.
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("katalogtreffer lesen: %w", err)
	}
	return treffer, gesamt, nil
}

// maskiereLikeJoker macht aus einer Eingabe einen wörtlichen LIKE-Teilstring: Backslash,
// Prozent und Unterstrich verlieren ihre Sonderbedeutung (Postgres-Vorgabe ESCAPE '\').
func maskiereLikeJoker(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// SchlagwortFilterImKatalog liefert die Wörter, die die Pflegeseite als Filter markiert hat
// (ist_filter), alphabetisch — aber nur die, zu denen der Katalog mit dem Prädikat sichtbar
// (über buecher_titel als `bt`) mindestens einen Titel zeigt. Ein Filter, der nichts
// findet, stünde sonst als Sackgasse über der Suche.
//
// Die Abfrage geht vom Wort zu seinen Titeln, nicht umgekehrt: Eine Bedingung je Wort über
// alle Titel lief an 13.000 Titeln und 20 Filterwörtern 353 ms, diese Form 4 ms.
func SchlagwortFilterImKatalog(ctx context.Context, q DBQueryer, sichtbar string) ([]SchlagwortFilter, error) {
	rows, err := q.Query(ctx, `
		SELECT f.id::text, f.wort
		FROM schlagworte f
		WHERE f.id IN (
		      SELECT sw.id FROM `+sqlWortZumTitel+`
		      JOIN buecher_titel bt ON bt.id = tsw.titel_id
		      WHERE sw.ist_filter AND `+sichtbar+`)
		ORDER BY lower(f.wort)`)
	if err != nil {
		return nil, fmt.Errorf("schlagwort-filter lesen: %w", err)
	}
	filter, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SchlagwortFilter])
	if err != nil {
		return nil, fmt.Errorf("schlagwort-filter lesen: %w", err)
	}
	return filter, nil
}
