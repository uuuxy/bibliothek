package inventur

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBook_JSONMarshaling(t *testing.T) {
	lastCounted := "2023-10-27T10:00:00Z"
	originalBook := Book{
		ID:                      "book-123",
		ISBN:                    "978-3-16-148410-0",
		Title:                   "Test Book Title",
		Author:                  "John Doe",
		Signatur:                "Test-Sig",
		CoverURL:                "https://example.com/cover.jpg",
		Subject:                 "Math",
		GradeLevel:              5,
		Track:                   "A",
		Stock:                   10,
		Verfuegbar:              8,
		Gesamt:                  10,
		LastCounted:             &lastCounted,
		SortOrder:               1,
		Medientyp:               "Book",
		JahrgangVon:             5,
		JahrgangBis:             6,
		Untertitel:              "An Introduction",
		Verlag:                  "Test Publisher",
		Erscheinungsjahr:        2023,
		Beschreibung:            "A test description",
		ErweiterteEigenschaften: map[string]any{"key1": "value1", "key2": float64(42)},
	}

	jsonData, err := json.Marshal(originalBook)
	require.NoError(t, err, "Should marshal Book to JSON without error")

	var unmarshaledBook Book
	err = json.Unmarshal(jsonData, &unmarshaledBook)
	require.NoError(t, err, "Should unmarshal JSON to Book without error")

	assert.Equal(t, originalBook, unmarshaledBook, "Unmarshaled Book should match the original")
}

func TestClassBookAssignment_JSONMarshaling(t *testing.T) {
	originalAssignment := ClassBookAssignment{
		ClassName: "Class 5A",
		BookID:    "book-123",
		Title:     "Test Book Title",
		Subject:   "Math",
		Track:     "A",
		CoverURL:  "https://example.com/cover.jpg",
	}

	jsonData, err := json.Marshal(originalAssignment)
	require.NoError(t, err, "Should marshal ClassBookAssignment to JSON without error")

	var unmarshaledAssignment ClassBookAssignment
	err = json.Unmarshal(jsonData, &unmarshaledAssignment)
	require.NoError(t, err, "Should unmarshal JSON to ClassBookAssignment without error")

	assert.Equal(t, originalAssignment, unmarshaledAssignment, "Unmarshaled ClassBookAssignment should match the original")
}
