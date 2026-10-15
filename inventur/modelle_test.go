package inventur

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTitel_JSON(t *testing.T) {
	zuletztGezaehlt := "2023-10-27T10:00:00Z"
	original := Titel{
		ID:                      "book-123",
		ISBN:                    "978-3-16-148410-0",
		Titel:                   "Test Book Title",
		Autor:                   "John Doe",
		Signatur:                "Test-Sig",
		CoverURL:                "https://example.com/cover.jpg",
		Fach:                    "Math",
		Zweig:                   "A",
		Bestand:                 10,
		Verfuegbar:              8,
		Gesamt:                  10,
		ZuletztGezaehlt:         &zuletztGezaehlt,
		Reihenfolge:             1,
		Medientyp:               "Book",
		JahrgangVon:             5,
		JahrgangBis:             6,
		Untertitel:              "An Introduction",
		Verlag:                  "Test Publisher",
		Erscheinungsjahr:        2023,
		ErweiterteEigenschaften: map[string]any{"key1": "value1", "key2": float64(42)},
	}

	jsonDaten, err := json.Marshal(original)
	require.NoError(t, err, "Should marshal Book to JSON without error")

	var gelesen Titel
	err = json.Unmarshal(jsonDaten, &gelesen)
	require.NoError(t, err, "Should unmarshal JSON to Book without error")

	assert.Equal(t, original, gelesen, "Unmarshaled Book should match the original")
}

// ClassBookAssignment ist entfernt: Das Struct hatte keinen
// einzigen produktiven Verwender — die Klassen-Buchlisten laufen über ClassBook/
// ClassGroup (klassensaetze.go). Sein Round-Trip-Test hier hielt es nur
// künstlich am Leben.
