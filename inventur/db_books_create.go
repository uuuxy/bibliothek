package inventur

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// CreateBook inserts a new book record.
func (repo *BookRepository) CreateBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, signatur)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12, $13, $14, $15, $16, $17, NULLIF($18, ''))
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
		book.Signatur,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", handleDbError(err))
	}

	// Bestand synchronisieren
	if book.Stock > 0 {
		if syncErr := repo.syncBookStock(ctx, id, book.Stock); syncErr != nil {
			// Log error, but don't fail the creation
			log.Printf("Warnung: Konnte Exemplare nach Erstellung nicht synchronisieren: %v\n", syncErr)
		}
	}

	return id, nil
}

type bookBatchData struct {
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
	signaturen              []string
	erweiterteEigenschaften [][]byte
}

func prepareUpsertBatchData(books []Book) bookBatchData {
	data := bookBatchData{
		isbns:                   make([]string, len(books)),
		titles:                  make([]string, len(books)),
		authors:                 make([]string, len(books)),
		coverUrls:               make([]string, len(books)),
		subjects:                make([]string, len(books)),
		grades:                  make([]int16, len(books)),
		tracks:                  make([]string, len(books)),
		stocks:                  make([]int32, len(books)),
		lastCounteds:            make([]*string, len(books)),
		medientypen:             make([]string, len(books)),
		jahrgaengeVon:           make([]int, len(books)),
		jahrgaengeBis:           make([]int, len(books)),
		untertitel:              make([]string, len(books)),
		verlage:                 make([]string, len(books)),
		erscheinungsjahre:       make([]int, len(books)),
		beschreibungen:          make([]string, len(books)),
		signaturen:              make([]string, len(books)),
		erweiterteEigenschaften: make([][]byte, len(books)),
	}

	for i, b := range books {
		data.isbns[i] = b.ISBN
		data.titles[i] = b.Title
		data.authors[i] = b.Author
		data.coverUrls[i] = b.CoverURL
		data.subjects[i] = b.Subject
		data.grades[i] = b.GradeLevel
		data.tracks[i] = b.Track
		// #nosec G115 - parseBestand begrenzt Stock beim Import auf [0, MaxInt32]
		data.stocks[i] = int32(b.Stock)
		data.lastCounteds[i] = b.LastCounted
		data.medientypen[i] = b.Medientyp
		if data.medientypen[i] == "" {
			data.medientypen[i] = "Buch"
		}
		data.jahrgaengeVon[i] = b.JahrgangVon
		data.jahrgaengeBis[i] = b.JahrgangBis
		data.untertitel[i] = b.Untertitel
		data.verlage[i] = b.Verlag
		data.erscheinungsjahre[i] = b.Erscheinungsjahr
		data.beschreibungen[i] = b.Beschreibung
		data.signaturen[i] = b.Signatur

		props := b.ErweiterteEigenschaften
		if props == nil {
			props = make(map[string]any)
		}
		// In JSON umwandeln für pgx JSONB-Array Kompatibilität
		jsonProps, _ := json.Marshal(props) //nolint:errcheck
		data.erweiterteEigenschaften[i] = jsonProps
	}
	return data
}

func (repo *BookRepository) executeUpsertBatchQuery(ctx context.Context, data bookBatchData) (int64, error) {
	// signatur: NULLIF beim Insert + COALESCE beim Konflikt — Import-Läufe
	// dürfen eine physisch verklebte Signatur nie mit Leerwerten überschreiben.
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften, signatur)
		SELECT t.isbn, t.titel, t.autor, t.cover_url, t.subject, t.grade_level, t.track, t.stock, NULLIF(t.last_counted_text, '')::date, t.medientyp, t.jahrgang_von, t.jahrgang_bis, t.untertitel, t.verlag, t.erscheinungsjahr, t.beschreibung, t.erweiterte_eigenschaften, NULLIF(t.signatur, '')
		FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::smallint[], $7::text[], $8::int[], $9::text[], $10::text[], $11::int[], $12::int[], $13::text[], $14::text[], $15::int[], $16::text[], $17::jsonb[], $18::text[])
		AS t(isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted_text, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften, signatur)
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
			erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften,
			signatur = COALESCE(NULLIF(EXCLUDED.signatur, ''), buecher_titel.signatur)
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
		data.signaturen,
	)
	if err != nil {
		return 0, fmt.Errorf("bücher konnten nicht im batch importiert werden: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// UpsertBooksBatch handles batch upserting book records.
func (repo *BookRepository) UpsertBooksBatch(ctx context.Context, books []Book) (int64, error) {
	if len(books) == 0 {
		return 0, nil
	}

	data := prepareUpsertBatchData(books)
	return repo.executeUpsertBatchQuery(ctx, data)
}

// UpsertBook inserts or updates a book record.
func (repo *BookRepository) UpsertBook(ctx context.Context, book Book) (string, error) {
	query := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, grade_level, track, stock, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, beschreibung, erweiterte_eigenschaften, signatur)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9::text, '')::date, $10, $11, $12, $13, $14, $15, $16, $17, NULLIF($18, ''))
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
			erweiterte_eigenschaften = EXCLUDED.erweiterte_eigenschaften,
			signatur = COALESCE(NULLIF(EXCLUDED.signatur, ''), buecher_titel.signatur)
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
		book.Signatur,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}

	return id, nil
}
