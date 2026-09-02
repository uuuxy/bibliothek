package inventur

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// buchListenSelect ist der gemeinsame SELECT für Buchlisten und Einzel-Reads:
// Stammdaten (inkl. Signatur) plus Verfügbarkeits-/Bestandszählung über die
// Exemplare. Verwender hängen WHERE an; buchListenGroupBy muss folgen.
const buchListenSelect = `
	SELECT
		bt.id, COALESCE(bt.isbn, '') AS isbn, bt.titel AS title, COALESCE(bt.autor, '') AS author,
		COALESCE(bt.signatur, '') AS signatur,
		COALESCE(bt.cover_url, '') AS cover_url, COALESCE(bt.subject, '') AS subject,
		COALESCE(bt.grade_level, 0) AS grade_level, COALESCE(bt.track, '') AS track, bt.ist_lernmittel,
		COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
		COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt,
		TO_CHAR(bt.last_counted, 'YYYY-MM-DD') as last_counted, bt.sort_order, COALESCE(bt.medientyp, 'Buch') AS medientyp,
		COALESCE(bt.jahrgang_von, 5) AS jahrgang_von, COALESCE(bt.jahrgang_bis, 10) AS jahrgang_bis,
		COALESCE(bt.untertitel, '') AS untertitel, COALESCE(bt.verlag, '') AS verlag,
		COALESCE(bt.erscheinungsjahr, 0) AS erscheinungsjahr, COALESCE(bt.beschreibung, '') AS beschreibung,
		bt.erweiterte_eigenschaften
	FROM buecher_titel bt
	LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
	LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
`

const buchListenGroupBy = `
	GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.signatur, bt.cover_url, bt.subject, bt.grade_level, bt.track, bt.ist_lernmittel, bt.last_counted, bt.sort_order, bt.medientyp, bt.jahrgang_von, bt.jahrgang_bis, bt.untertitel, bt.verlag, bt.erscheinungsjahr, bt.beschreibung, bt.erweiterte_eigenschaften
`

// buchListenSelectSchlank ist die LISTEN-Variante: identische Spaltenzahl/-reihenfolge
// (scanBuchZeilen bleibt unverändert), aber die zwei schwersten Spalten — beschreibung
// (TEXT) und erweiterte_eigenschaften (JSONB) — werden als leere Konstanten geliefert
// statt vom Server übertragen. Die Listen-/Katalogansicht und ihre clientseitige Suche
// nutzen diese Felder NICHT (nur Titel/Autor/ISBN/Fach/Track/Jahrgang); das Detail holt
// sie per Einzel-Read (ListBooksByIDs) mit dem vollen buchListenSelect nach. Das
// verkleinert die /api/books-Payload je Titel um ein Vielfaches — ohne Verhaltensänderung.
const buchListenSelectSchlank = `
	SELECT
		bt.id, COALESCE(bt.isbn, '') AS isbn, bt.titel AS title, COALESCE(bt.autor, '') AS author,
		COALESCE(bt.signatur, '') AS signatur,
		COALESCE(bt.cover_url, '') AS cover_url, COALESCE(bt.subject, '') AS subject,
		COALESCE(bt.grade_level, 0) AS grade_level, COALESCE(bt.track, '') AS track, bt.ist_lernmittel,
		COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
		COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt,
		TO_CHAR(bt.last_counted, 'YYYY-MM-DD') as last_counted, bt.sort_order, COALESCE(bt.medientyp, 'Buch') AS medientyp,
		COALESCE(bt.jahrgang_von, 5) AS jahrgang_von, COALESCE(bt.jahrgang_bis, 10) AS jahrgang_bis,
		COALESCE(bt.untertitel, '') AS untertitel, COALESCE(bt.verlag, '') AS verlag,
		COALESCE(bt.erscheinungsjahr, 0) AS erscheinungsjahr, '' AS beschreibung,
		'{}'::jsonb AS erweiterte_eigenschaften
	FROM buecher_titel bt
	LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
	LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
`

// buchListenGroupBySchlank lässt die beiden Konstanten-Spalten aus der Gruppierung weg
// (Konstanten müssen nicht gruppiert werden — spart dem Server das Hashen großer Werte).
const buchListenGroupBySchlank = `
	GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.signatur, bt.cover_url, bt.subject, bt.grade_level, bt.track, bt.ist_lernmittel, bt.last_counted, bt.sort_order, bt.medientyp, bt.jahrgang_von, bt.jahrgang_bis, bt.untertitel, bt.verlag, bt.erscheinungsjahr
`

// listBooksSicherheitsLimit kappt die Katalogliste als reine Runaway-/Speicher-Bremse.
// Bewusst weit über jedem realistischen Katalog (~14k Titel), damit die clientseitige
// Sofort-Suche (lädt den ganzen gefilterten Satz) im Normalbetrieb NIE etwas verliert.
// Wird die Grenze je erreicht, ist das das Signal, auf echte serverseitige Pagination
// umzustellen — und es wird geloggt, statt still Titel zu verschlucken.
const listBooksSicherheitsLimit = 50000

// scanBuchZeilen liest alle Zeilen eines buchListenSelect-Ergebnisses ein.
func scanBuchZeilen(rows pgx.Rows) ([]Book, error) {
	books := make([]Book, 0)
	for rows.Next() {
		var book Book
		err := rows.Scan(
			&book.ID,
			&book.ISBN,
			&book.Title,
			&book.Author,
			&book.Signatur,
			&book.CoverURL,
			&book.Subject,
			&book.GradeLevel,
			&book.Track,
			&book.IstLernmittel,
			&book.Verfuegbar,
			&book.Gesamt,
			&book.LastCounted,
			&book.SortOrder,
			&book.Medientyp,
			&book.JahrgangVon,
			&book.JahrgangBis,
			&book.Untertitel,
			&book.Verlag,
			&book.Erscheinungsjahr,
			&book.Beschreibung,
			&book.ErweiterteEigenschaften,
		)
		if err != nil {
			return nil, fmt.Errorf("daten konnten nicht gelesen werden: %w", err)
		}
		book.Stock = book.Gesamt
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim iterieren: %w", err)
	}
	return books, nil
}

// ListBooks lists books matching subject, grade level, and text query.
//
// Nutzt die schlanke Listen-Variante (ohne beschreibung/erweiterte_eigenschaften) und
// eine Sicherheits-Kappung gegen unbegrenztes Wachstum — siehe buchListenSelectSchlank
// bzw. listBooksSicherheitsLimit.
func (repo *BookRepository) ListBooks(ctx context.Context, subject string, grade *int16, searchQuery string) ([]Book, error) {
	query := buchListenSelectSchlank + `
		WHERE ($1 = '' OR bt.subject = $1)
		  AND ($2::smallint IS NULL OR bt.grade_level = $2)
		  AND ($3 = '' OR bt.titel ILIKE '%' || $3 || '%' OR bt.autor ILIKE '%' || $3 || '%' OR bt.isbn ILIKE '%' || $3 || '%' OR bt.subject ILIKE '%' || $3 || '%' OR CAST(bt.id AS TEXT) ILIKE '%' || $3 || '%')
	` + buchListenGroupBySchlank + `
		ORDER BY bt.sort_order ASC, bt.titel ASC
		LIMIT $4`

	rows, err := repo.db.Query(ctx, query, subject, grade, searchQuery, listBooksSicherheitsLimit)
	if err != nil {
		return nil, fmt.Errorf("bücher konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	books, err := scanBuchZeilen(rows)
	if err != nil {
		return nil, err
	}
	if len(books) >= listBooksSicherheitsLimit {
		log.Printf("WARNUNG: /api/books hat die Sicherheits-Kappung von %d Titeln erreicht — "+
			"die Katalogliste ist unvollständig. Jetzt auf serverseitige Pagination umstellen.",
			listBooksSicherheitsLimit)
	}
	return books, nil
}

// ListExternalCoverBooks lists books having external cover URLs.
func (repo *BookRepository) ListExternalCoverBooks(ctx context.Context, limit int) ([]Book, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(cover_url, '') AS cover_url
		FROM buecher_titel
		WHERE cover_url LIKE 'http%'
		ORDER BY id ASC
		LIMIT $1`

	rows, err := repo.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("bücher mit externen covern konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	books := make([]Book, 0)
	for rows.Next() {
		var book Book
		if scanErr := rows.Scan(&book.ID, &book.ISBN, &book.Title, &book.CoverURL); scanErr != nil {
			return nil, fmt.Errorf("bücher mit externen covern konnten nicht gelesen werden: %w", scanErr)
		}
		books = append(books, book)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("fehler beim iterieren externer cover-bücher: %w", rowsErr)
	}

	return books, nil
}

// ListBooksByIDs retrieves list of books for provided IDs.
// Liefert den VOLLEN Datensatz inkl. Signatur und Bestandszählung — der
// Einzel-Read GET /api/books/{id} (Buch-Akte, „Titel bearbeiten") hängt daran.
// Das frühere Minimal-SELECT (id, isbn, titel, cover_url) ließ Signatur und
// Exemplar-Zahlen im Frontend leer erscheinen.
func (repo *BookRepository) ListBooksByIDs(ctx context.Context, ids []string) ([]Book, error) {
	if len(ids) == 0 {
		return []Book{}, nil
	}

	query := buchListenSelect + `
		WHERE bt.id = ANY($1::uuid[])
	` + buchListenGroupBy + `
		ORDER BY bt.id ASC`

	rows, err := repo.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("bücher nach ids konnten nicht geladen werden: %w", err)
	}
	defer rows.Close()

	return scanBuchZeilen(rows)
}
