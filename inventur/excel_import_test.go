package inventur

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/pashagolub/pgxmock/v4"
)

func TestProcessImportRows(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := NewBookRepository(mockDB)
	metadatenClient := NeuerMetadatenClient()

	handler := &APIHandler{
		repo:      repo,
		metadaten: metadatenClient,
	}

	ctx := context.Background()
	colIdx := map[string]int{
		"isbn":    0,
		"titel":   1,
		"autor":   2,
		"fach":    3,
		"klasse":  4,
		"bestand": 5,
	}

	dataRows := [][]string{
		{"9781234567897", "Math Book", "John Doe", "Mathematik", "7", "5"},
		{"9789876543210", "Science Book", "Jane Smith", "Biologie", "8", "3"},
		{"", "Empty ISBN Book", "Nobody", "Unbekannt", "", ""}, // Should be skipped
	}

	booksToUpsert, failed, firstError := handler.processImportRows(ctx, dataRows, colIdx)

	assert.NoError(t, firstError)
	assert.Equal(t, int32(0), failed)

	// We expect 2 books to be processed, empty ISBN is skipped
	assert.Len(t, booksToUpsert, 2)

	// We cannot guarantee the order because it's processed concurrently
	var foundMath, foundScience bool
	for _, book := range booksToUpsert {
		if book.ISBN == "9781234567897" {
			foundMath = true
			assert.Equal(t, "Math Book", book.Title)
			assert.Equal(t, "John Doe", book.Author)
			assert.Equal(t, "Mathematik", book.Subject)
			assert.Equal(t, int16(7), book.GradeLevel)
			assert.Equal(t, 5, book.Stock)
		} else if book.ISBN == "9789876543210" {
			foundScience = true
			assert.Equal(t, "Science Book", book.Title)
			assert.Equal(t, "Jane Smith", book.Author)
			assert.Equal(t, "Biologie", book.Subject)
			assert.Equal(t, int16(8), book.GradeLevel)
			assert.Equal(t, 3, book.Stock)
		} else {
			t.Errorf("Unexpected book returned: %+v", book)
		}
	}
	assert.True(t, foundMath, "Math Book not found")
	assert.True(t, foundScience, "Science Book not found")
}

func TestProcessImportRows_MissingColumns(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := NewBookRepository(mockDB)
	metadatenClient := NeuerMetadatenClient()

	handler := &APIHandler{
		repo:      repo,
		metadaten: metadatenClient,
	}

	ctx := context.Background()
	// Missing "bestand", and "fach"
	colIdx := map[string]int{
		"isbn":    0,
		"titel":   1,
		"autor":   2,
		"klasse":  3,
		"bestand": -1,
		"fach":    -1,
	}

	dataRows := [][]string{
		{"9781234567897", "Mathematik für Anfänger", "John Doe", "7"},
	}

	booksToUpsert, failed, firstError := handler.processImportRows(ctx, dataRows, colIdx)

	assert.NoError(t, firstError)
	assert.Equal(t, int32(0), failed)
	assert.Len(t, booksToUpsert, 1)

	book := booksToUpsert[0]
	assert.Equal(t, "9781234567897", book.ISBN)
	// Default subject should be used or inferred from title
	assert.Equal(t, "Mathematik", book.Subject) // Inferred from "Mathematik" in title
	assert.Equal(t, int16(7), book.GradeLevel)
	assert.Equal(t, 0, book.Stock) // Default stock is 0
}

func TestProcessImportRows_HighConcurrency(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := NewBookRepository(mockDB)
	metadatenClient := NeuerMetadatenClient()

	handler := &APIHandler{
		repo:      repo,
		metadaten: metadatenClient,
	}

	ctx := context.Background()
	colIdx := map[string]int{
		"isbn":    0,
		"titel":   1,
	}

	// Create 100 rows
	var dataRows [][]string
	for i := 0; i < 100; i++ {
		// Valid ISBN
		dataRows = append(dataRows, []string{"9781234567890", "A Title"})
	}

	booksToUpsert, failed, firstError := handler.processImportRows(ctx, dataRows, colIdx)

	assert.NoError(t, firstError)
	assert.Equal(t, int32(0), failed)
	assert.Len(t, booksToUpsert, 100)
}
