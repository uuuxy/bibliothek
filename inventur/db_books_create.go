package inventur

import (
	"context"
	"fmt"
)

// CreateBook inserts a new book record.
func (repo *BookRepository) CreateBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12, $13)
		RETURNING id`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	properties := book.ErweiterteEigenschaften
	if properties == nil {
		properties = make(map[string]any)
	}

	var id string
	err := repo.db.QueryRow(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		book.Subject,
		book.GradeLevel,
		book.Track,
		book.Stock,
		book.LastCounted,
		medientyp,
		properties,
		book.JahrgangVon,
		book.JahrgangBis,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", handleDbError(err))
	}

	// Bestand synchronisieren
	if book.Stock > 0 {
		if syncErr := repo.syncBookStock(ctx, id, book.Stock); syncErr != nil {
			// Log error, but don't fail the creation
			fmt.Printf("Warnung: Konnte Exemplare nach Erstellung nicht synchronisieren: %v\n", syncErr)
		}
	}

	return id, nil
}

// UpsertBooksBatch handles batch upserting book records.
func (repo *BookRepository) UpsertBooksBatch(ctx context.Context, books []Book) (int64, error) {
	if len(books) == 0 {
		return 0, nil
	}

	isbns := make([]string, len(books))
	titles := make([]string, len(books))
	authors := make([]string, len(books))
	coverUrls := make([]string, len(books))
	subjects := make([]string, len(books))
	grades := make([]int16, len(books))
	tracks := make([]string, len(books))
	stocks := make([]int32, len(books))
	lastCounteds := make([]*string, len(books))
	medientypen := make([]string, len(books))
	jahrgaengeVon := make([]int, len(books))
	jahrgaengeBis := make([]int, len(books))

	for i, b := range books {
		isbns[i] = b.ISBN
		titles[i] = b.Title
		authors[i] = b.Author
		coverUrls[i] = b.CoverURL
		subjects[i] = b.Subject
		grades[i] = b.GradeLevel
		tracks[i] = b.Track
		// #nosec G115 - Stock is a physical book count, fits easily in int32
		stocks[i] = int32(b.Stock)
		lastCounteds[i] = b.LastCounted
		medientypen[i] = b.Medientyp
		if medientypen[i] == "" {
			medientypen[i] = "Buch"
		}
		jahrgaengeVon[i] = b.JahrgangVon
		jahrgaengeBis[i] = b.JahrgangBis
	}

	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis)
		SELECT t.isbn, t.titel, t.autor, t.cover_url, t.subject, t.grade_level, t.track, t.stock, NULLIF(t.last_counted_text, '')::date, t.medientyp, t.jahrgang_von, t.jahrgang_bis
		FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::smallint[], $7::text[], $8::int[], $9::text[], $10::text[], $11::int[], $12::int[])
		AS t(isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted_text, medientyp, jahrgang_von, jahrgang_bis)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			cover_url = EXCLUDED.cover_url,
			subject = EXCLUDED.subject,
			grade_level = EXCLUDED.grade_level,
			track = EXCLUDED.track,
			stock = buecher_titel.stock + EXCLUDED.stock,
			last_counted = EXCLUDED.last_counted,
			medientyp = EXCLUDED.medientyp,
			jahrgang_von = EXCLUDED.jahrgang_von,
			jahrgang_bis = EXCLUDED.jahrgang_bis
	`

	cmdTag, err := repo.db.Exec(
		ctx,
		query,
		isbns,
		titles,
		authors,
		coverUrls,
		subjects,
		grades,
		tracks,
		stocks,
		lastCounteds,
		medientypen,
		jahrgaengeVon,
		jahrgaengeBis,
	)
	if err != nil {
		return 0, fmt.Errorf("bücher konnten nicht im batch importiert werden: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// UpsertBook inserts or updates a book record.
func (repo *BookRepository) UpsertBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			subject = EXCLUDED.subject,
			grade_level = EXCLUDED.grade_level,
			track = EXCLUDED.track,
			stock = buecher_titel.stock + EXCLUDED.stock,
			last_counted = EXCLUDED.last_counted,
			medientyp = EXCLUDED.medientyp,
			jahrgang_von = EXCLUDED.jahrgang_von,
			jahrgang_bis = EXCLUDED.jahrgang_bis
		RETURNING id`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	var id string
	err := repo.db.QueryRow(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		book.Subject,
		book.GradeLevel,
		book.Track,
		book.Stock,
		book.LastCounted,
		medientyp,
		book.JahrgangVon,
		book.JahrgangBis,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}

	return id, nil
}
