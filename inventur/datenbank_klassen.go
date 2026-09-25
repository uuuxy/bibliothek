package inventur

import (
	"context"
	"encoding/json"
	"fmt"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// insertClassBookBindings schreibt das Kreuzprodukt aus Klassen × Büchern per
// unnest-INSERT in class_books. Leere Eingaben werden ignoriert (kein Insert).
func insertClassBookBindings(ctx context.Context, tx pgx.Tx, newClassNames, bookIDs []string) error {
	if len(bookIDs) == 0 || len(newClassNames) == 0 {
		return nil
	}

	var insertClasses []string
	var insertBooks []string
	for _, className := range newClassNames {
		for _, bookID := range bookIDs {
			insertClasses = append(insertClasses, className)
			insertBooks = append(insertBooks, bookID)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO class_books (class_name, book_id)
		SELECT class_name, book_id::uuid FROM unnest($1::text[], $2::text[]) AS t(class_name, book_id)`, insertClasses, insertBooks); err != nil {
		return fmt.Errorf("neue zuweisung konnte nicht gespeichert werden: %w", err)
	}
	return nil
}

// KlassensatzMindestLeser ist die Mindestzahl: Ab so vielen Kindern derselben Klasse mit demselben Titel — und
// mehr als der Hälfte der Klasse — gilt ein Titel als Klassensatz „aus Ausleihen". Die
// Hälfte allein reichte nicht: In einer Restklasse mit vier Kindern wären zwei Leser schon
// ein Satz.
const KlassensatzMindestLeser = 5

// GetClassGroups liefert je Klasse ihre Klassensatz-Titel aus ZWEI Quellen (Absprache vom
// 05.09.2026: „über die LUSD wissen wir genau, in welcher Klasse jeder Schüler ist — die
// Liste kann sich selbstständig aktualisieren"): der von Hand gepflegten Zuordnung
// (class_books, Quelle „hand" — bleibt unangetastet, nichts wird geschrieben oder
// gelöscht) und den laufenden Ausleihen (Quelle „ausleihe", live gerechnet, nie
// gespeichert): Halten mehr als die Hälfte der aktiven Kinder einer Klasse, mindestens
// KlassensatzMindestLeser, denselben Titel, ist das ihr Klassensatz — Schulbuch oder
// Lektüre. Ein Titel, der schon von Hand zugeordnet ist, erscheint nur einmal (hand).
// Beide Quellen laufen durch dieselbe Sortierung; branch filtert auf den Bildungsgang,
// sortOrder bestimmt die Reihenfolge der Klassen.
//
// Auflagen (docs/OFFEN.md 4.18, Stufe 5, 25.09.2026): Gezählt wird am BUCH, nicht an der
// Auflage — COALESCE(werk_id, id) wie im Bestellbedarf. Je Auflage gezählt, verschwand ein
// Klassensatz, sobald er auf zwei Auflagen verteilt war (14 Kinder mit der 4., 14 mit der
// 3. Auflage: keine hat mehr als die Hälfte), und bei 20 zu 8 blieb die Mischung
// unsichtbar. Für ein Buch aus den Ausleihen steht die Auflage, die die meisten Kinder der
// Klasse haben, bei Gleichstand die neueste: Die Kachel zeigt, was die Klasse hat, nicht,
// was zuletzt erschien. Eine Zuordnung von Hand deckt das ganze Buch ab. Haben Kinder der
// Klasse eine andere Auflage als die der Kachel, trägt die Zeile die Aufschlüsselung.
func (repo *BookRepository) GetClassGroups(ctx context.Context, branch string, sortOrder string) ([]ClassGroup, error) {
	neueste := repository.SQLNeuesteAuflageZuerst("t")
	query := `
		WITH klassen_groesse AS (
			SELECT klassen_normkey(klasse) AS k, min(klasse) AS klasse, count(*) AS n
			FROM schueler
			WHERE deleted_at IS NULL AND ist_abgaenger = false AND klasse ~ '^\d'
			GROUP BY klassen_normkey(klasse)
		),
		offen AS (
			SELECT klassen_normkey(s.klasse) AS k, s.id AS kind, t.id AS titel_id,
			       COALESCE(t.werk_id, t.id) AS buch
			FROM ausleihen a
			JOIN schueler s ON s.id = a.schueler_id AND s.deleted_at IS NULL AND s.ist_abgaenger = false
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE a.rueckgabe_am IS NULL
		),
		je_auflage AS (
			SELECT k, buch, titel_id, count(DISTINCT kind) AS kinder
			FROM offen GROUP BY k, buch, titel_id
		),
		leser AS (
			SELECT k, buch, count(DISTINCT kind) AS leser
			FROM offen GROUP BY k, buch
		),
		vertreter AS (
			SELECT DISTINCT ON (j.k, j.buch) j.k, j.buch, j.titel_id
			FROM je_auflage j JOIN buecher_titel t ON t.id = j.titel_id
			ORDER BY j.k, j.buch, j.kinder DESC, ` + neueste + `
		),
		mischung AS (
			-- Nur Titel, die zu einem Buch mit mehreren Auflagen gehören: Ohne Werk ist buch = titel_id.
			SELECT j.k, j.buch, array_agg(j.titel_id) AS titel_ids,
			       json_agg(json_build_object('id', t.id, 'auflage', coalesce(t.auflage, ''),
			           'erscheinungsjahr', coalesce(t.erscheinungsjahr, 0), 'kinder', j.kinder)
			           ORDER BY j.kinder DESC, ` + neueste + `) AS auflagen
			FROM je_auflage j JOIN buecher_titel t ON t.id = j.titel_id
			WHERE j.buch <> j.titel_id
			GROUP BY j.k, j.buch
		),
		zuordnung AS (
			SELECT class_name, book_id, 'hand'::text AS quelle, 0::bigint AS leser FROM class_books
			UNION ALL
			SELECT COALESCE((SELECT min(h.class_name) FROM class_books h WHERE klassen_normkey(h.class_name) = l.k), g.klasse),
			       v.titel_id, 'ausleihe', l.leser
			FROM leser l
			JOIN klassen_groesse g ON g.k = l.k
			JOIN vertreter v ON v.k = l.k AND v.buch = l.buch
			WHERE l.leser * 2 > g.n AND l.leser >= $2
			  AND NOT EXISTS (SELECT 1 FROM class_books h JOIN buecher_titel ht ON ht.id = h.book_id
			                  WHERE klassen_normkey(h.class_name) = l.k AND COALESCE(ht.werk_id, ht.id) = l.buch)
		)
		SELECT
			cb.class_name, b.id, b.titel AS title, COALESCE(b.subject, '') AS subject,
			COALESCE(b.track, '') AS track, COALESCE(b.cover_url, '') AS cover_url,
			COALESCE(b.isbn, '') AS isbn,
			COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
			COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt,
			` + repository.SQLFilterImZulauf + ` AS im_zulauf,
			cb.quelle, cb.leser,
			(SELECT m.auflagen FROM mischung m
			 WHERE m.k = klassen_normkey(cb.class_name) AND m.buch = COALESCE(b.werk_id, b.id)
			   AND NOT (m.titel_ids <@ ARRAY[b.id])) AS auflagen
		FROM zuordnung cb
		JOIN buecher_titel b ON cb.book_id = b.id
		LEFT JOIN buecher_exemplare e ON e.titel_id = b.id
		LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
		WHERE ($1 = '' OR cb.class_name ILIKE '%' || $1 || '%')
		GROUP BY cb.class_name, b.id, b.titel, b.subject, b.track, b.cover_url, b.isbn, cb.quelle, cb.leser
		ORDER BY `

	if branch == "" {
		// Fallback: Workflow-Reihenfolge F, G, R, H
		query += `
			CASE 
				WHEN cb.class_name ILIKE '%F%' THEN 1
				WHEN cb.class_name ILIKE '%G%' THEN 2
				WHEN cb.class_name ILIKE '%R%' THEN 3
				WHEN cb.class_name ILIKE '%H%' THEN 4
				ELSE 5
			END, `
	}

	gradeCast := `CAST(SUBSTRING(cb.class_name FROM '^[0-9]+') AS INTEGER)`
	descending := sortOrder == "desc"

	if descending {
		query += gradeCast + ` DESC, cb.class_name DESC, b.titel ASC`
	} else {
		query += gradeCast + ` ASC, cb.class_name ASC, b.titel ASC`
	}

	rows, err := repo.db.Query(ctx, query, branch, KlassensatzMindestLeser)
	if err != nil {
		return nil, fmt.Errorf("klassen-bücher konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	groupsMap := make(map[string][]ClassBook)
	var classNames []string

	for rows.Next() {
		var className string
		var book ClassBook
		var auflagen []byte
		err := rows.Scan(&className, &book.ID, &book.Title, &book.Subject, &book.Track, &book.CoverURL, &book.ISBN, &book.Verfuegbar, &book.Gesamt, &book.ImZulauf, &book.Quelle, &book.Leser, &auflagen)
		if err != nil {
			return nil, fmt.Errorf("daten konnten nicht gelesen werden: %w", err)
		}
		if len(auflagen) > 0 {
			if err := json.Unmarshal(auflagen, &book.Auflagen); err != nil {
				return nil, fmt.Errorf("auflagen von %s in %s: %w", book.ID, className, err)
			}
		}

		if _, exists := groupsMap[className]; !exists {
			classNames = append(classNames, className)
		}
		groupsMap[className] = append(groupsMap[className], book)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("klassen-bücher konnten nicht vollständig gelesen werden: %w", err)
	}

	var result []ClassGroup
	for _, name := range classNames {
		result = append(result, ClassGroup{
			ClassName: name,
			Books:     groupsMap[name],
		})
	}

	return result, nil
}

// UpdateClassBooks schreibt die Buchzuordnung einer Klasse neu. oldClassName wird dabei
// durch newClassNames ersetzt — so lässt sich derselbe Satz in einem Zug auf mehrere
// Klassen verteilen, ohne ihn mehrfach anzulegen.
func (repo *BookRepository) UpdateClassBooks(ctx context.Context, oldClassName string, newClassNames []string, bookIDs []string) error {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaktion konnte nicht gestartet werden: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	// If there's an old class name, delete it.
	if oldClassName != "" {
		if _, err := tx.Exec(ctx, `DELETE FROM class_books WHERE class_name = $1`, oldClassName); err != nil {
			return fmt.Errorf("alte zuweisungen konnten nicht gelöscht werden: %w", err)
		}
	}

	// ⚡ Bolt: Batch DELETE existing bindings for all target classes (overwrite)
	if len(newClassNames) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM class_books WHERE class_name = ANY($1)`, newClassNames); err != nil {
			return fmt.Errorf("vorhandene zuweisungen des neuen namens konnten nicht gelöscht werden: %w", err)
		}
	}

	// ⚡ Bolt: Batch INSERT new bindings using PostgreSQL unnest
	if err := insertClassBookBindings(ctx, tx, newClassNames, bookIDs); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaktion konnte nicht abgeschlossen werden: %w", err)
	}

	return nil
}

// DeleteClassGroup löst die Klassensätze der genannten Klassen auf. Betroffen ist nur
// die Zuordnung — die Titel selbst bleiben im Katalog.
func (repo *BookRepository) DeleteClassGroup(ctx context.Context, classNames []string) error {
	if len(classNames) == 0 {
		return nil
	}
	_, err := repo.db.Exec(ctx, `DELETE FROM class_books WHERE class_name = ANY($1)`, classNames)
	if err != nil {
		return fmt.Errorf("klassen konnten nicht gelöscht werden: %w", err)
	}
	return nil
}

// NormalizeAllClasses vereinheitlicht die Klassenbezeichnungen im Bestand (etwa „5A"
// und „5a" zur selben Klasse). Läuft in einer Transaktion: entweder alle oder keine.
func (repo *BookRepository) NormalizeAllClasses(ctx context.Context) error {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transaktion konnte nicht gestartet werden: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	// Step 1: Bereinigung von Leerzeichen.
	// Lösche Duplikate, die entstehen würden, wenn wir die Leerzeichen entfernen.
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE class_name LIKE '% %'
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = REPLACE(cb1.class_name, ' ', '')
			AND cb2.book_id = cb1.book_id
		)
	`)
	if err != nil {
		return fmt.Errorf("fehler beim bereinigen doppelter klassennamen vor leerzeichen-entfernung: %w", err)
	}

	// Update zum Entfernen von Leerzeichen
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = REPLACE(class_name, ' ', '')
		WHERE class_name LIKE '% %'
	`)
	if err != nil {
		return fmt.Errorf("fehler beim entfernen von leerzeichen in klassennamen: %w", err)
	}

	// Step 2: Delete books that would cause a unique key violation (führende Nullen)
	_, err = tx.Exec(ctx, `
		DELETE FROM class_books cb1
		WHERE (class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$')
		AND EXISTS (
			SELECT 1 FROM class_books cb2
			WHERE cb2.class_name = '0' || cb1.class_name
			AND cb2.book_id = cb1.book_id
		)
	`)
	if err != nil {
		return fmt.Errorf("fehler beim bereinigen doppelter klassennamen: %w", err)
	}

	// Step 2: Update the remaining rows
	_, err = tx.Exec(ctx, `
		UPDATE class_books
		SET class_name = '0' || class_name
		WHERE class_name ~ '^[1-9][^0-9]' OR class_name ~ '^[1-9]$'
	`)
	if err != nil {
		return fmt.Errorf("fehler beim normalisieren der klassennamen: %w", err)
	}

	return tx.Commit(ctx)
}

// AddBooksToClasses ergänzt Titel in mehreren Klassensätzen, ohne die bestehende
// Zuordnung zu ersetzen — anders als UpdateClassBooks, das sie neu schreibt.
func (repo *BookRepository) AddBooksToClasses(ctx context.Context, classNames []string, bookIDs []string) error {
	if len(classNames) == 0 || len(bookIDs) == 0 {
		return nil
	}

	var insertClasses []string
	var insertBooks []string

	for _, className := range classNames {
		for _, bookID := range bookIDs {
			insertClasses = append(insertClasses, className)
			insertBooks = append(insertBooks, bookID)
		}
	}

	// Use ON CONFLICT DO NOTHING so existing assignments are ignored and not duplicated
	query := `
		INSERT INTO class_books (class_name, book_id)
		SELECT class_name, book_id::uuid FROM unnest($1::text[], $2::text[]) AS t(class_name, book_id)
		ON CONFLICT (class_name, book_id) DO NOTHING
	`

	_, err := repo.db.Exec(ctx, query, insertClasses, insertBooks)
	if err != nil {
		return fmt.Errorf("fehler beim hinzufügen der bücher zu den klassen: %w", err)
	}

	return nil
}
