package inventur

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateBook inserts a new book record.
func (repo *BookRepository) CreateBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12, $13, $14, $15, $16, $17)
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
		book.Untertitel,
		book.Verlag,
		book.Erscheinungsjahr,
		book.Beschreibung,
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

type bookBatchArrays struct {
	isbns                   []string
	titles                  []string
	authors                 []string
	coverUrls               []string
	subjects                []string
	grades                  []int16
	tracks                  []string
	stocks                  []int32
	lastCounteds            []*string
	medientypen             []string
	jahrgaengeVon           []int
	jahrgaengeBis           []int
	untertitel              []string
	verlage                 []string
	erscheinungsjahre       []int
	beschreibungen          []string
	erweiterteEigenschaften [][]byte
}

func prepareBookBatchArrays(books []Book) bookBatchArrays {
	// Deduplicate in-memory by ISBN
	// Note: We use the first occurrence for metadata, but accumulate the stock.
	bookMap := make(map[string]*Book)
	uniqueISBNs := make([]string, 0, len(books))

	for i := range books {
		b := &books[i]
		if existing, found := bookMap[b.ISBN]; found {
			existing.Stock += b.Stock
		} else {
			// Copy book to avoid modifying original array
			bookCopy := *b
			bookMap[b.ISBN] = &bookCopy
			uniqueISBNs = append(uniqueISBNs, b.ISBN)
		}
	}

	uniqueCount := len(uniqueISBNs)
	data := bookBatchArrays{
		isbns:                   make([]string, uniqueCount),
		titles:                  make([]string, uniqueCount),
		authors:                 make([]string, uniqueCount),
		coverUrls:               make([]string, uniqueCount),
		subjects:                make([]string, uniqueCount),
		grades:                  make([]int16, uniqueCount),
		tracks:                  make([]string, uniqueCount),
		stocks:                  make([]int32, uniqueCount),
		lastCounteds:            make([]*string, uniqueCount),
		medientypen:             make([]string, uniqueCount),
		jahrgaengeVon:           make([]int, uniqueCount),
		jahrgaengeBis:           make([]int, uniqueCount),
		untertitel:              make([]string, uniqueCount),
		verlage:                 make([]string, uniqueCount),
		erscheinungsjahre:       make([]int, uniqueCount),
		beschreibungen:          make([]string, uniqueCount),
		erweiterteEigenschaften: make([][]byte, uniqueCount),
	}

	for i, isbn := range uniqueISBNs {
		b := bookMap[isbn]
		data.isbns[i] = b.ISBN
		data.titles[i] = b.Title
		data.authors[i] = b.Author
		data.coverUrls[i] = b.CoverURL
		data.subjects[i] = b.Subject
		data.grades[i] = b.GradeLevel
		data.tracks[i] = b.Track
		// #nosec G115 - Stock is a physical book count, fits easily in int32
		data.stocks[i] = int32(b.Stock)
		data.lastCounteds[i] = b.LastCounted

		medientyp := b.Medientyp
		if medientyp == "" {
			medientyp = "Buch"
		}
		data.medientypen[i] = medientyp

		data.jahrgaengeVon[i] = b.JahrgangVon
		data.jahrgaengeBis[i] = b.JahrgangBis
		data.untertitel[i] = b.Untertitel
		data.verlage[i] = b.Verlag
		data.erscheinungsjahre[i] = b.Erscheinungsjahr
		data.beschreibungen[i] = b.Beschreibung

		props := b.ErweiterteEigenschaften
		if props == nil {
			props = make(map[string]any)
		}
		// In JSON umwandeln für pgx JSONB-Array Kompatibilität
		jsonProps, _ := json.Marshal(props)
		data.erweiterteEigenschaften[i] = jsonProps
	}

	return data
}

// UpsertBooksBatch handles batch upserting book records.
func (repo *BookRepository) UpsertBooksBatch(ctx context.Context, books []Book) (int64, error) {
	if len(books) == 0 {
		return 0, nil
	}

	data := prepareBookBatchArrays(books)

	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften)
		SELECT t.isbn, t.titel, t.autor, t.cover_url, t.subject, t.grade_level, t.track, t.stock, NULLIF(t.last_counted_text, '')::date, t.medientyp, t.jahrgang_von, t.jahrgang_bis, t.untertitel, t.verlag, t.erscheinungsjahr, t.beschreibung, t.erweiterte_eigenschaften
		FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::smallint[], $7::text[], $8::int[], $9::text[], $10::text[], $11::int[], $12::int[], $13::text[], $14::text[], $15::int[], $16::text[], $17::jsonb[])
		AS t(isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted_text, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften)
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
			jahrgang_bis = EXCLUDED.jahrgang_bis,
			untertitel = EXCLUDED.untertitel,
			verlag = EXCLUDED.verlag,
			erscheinungsjahr = EXCLUDED.erscheinungsjahr,
			beschreibung = EXCLUDED.beschreibung,
			erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften
	`

	cmdTag, err := repo.db.Exec(
		ctx,
		query,
		data.isbns,
		data.titles,
		data.authors,
		data.coverUrls,
		data.subjects,
		data.grades,
		data.tracks,
		data.stocks,
		data.lastCounteds,
		data.medientypen,
		data.jahrgaengeVon,
		data.jahrgaengeBis,
		data.untertitel,
		data.verlage,
		data.erscheinungsjahre,
		data.beschreibungen,
		data.erweiterteEigenschaften,
	)
	if err != nil {
		return 0, fmt.Errorf("bücher konnten nicht im batch importiert werden: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// UpsertBook inserts or updates a book record.
func (repo *BookRepository) UpsertBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12, $13, $14, $15, $16, $17)
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
			jahrgang_bis = EXCLUDED.jahrgang_bis,
			untertitel = EXCLUDED.untertitel,
			verlag = EXCLUDED.verlag,
			erscheinungsjahr = EXCLUDED.erscheinungsjahr,
			beschreibung = EXCLUDED.beschreibung,
			erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften
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
		book.JahrgangVon,
		book.JahrgangBis,
		book.Untertitel,
		book.Verlag,
		book.Erscheinungsjahr,
		book.Beschreibung,
		properties,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}

	return id, nil
}
