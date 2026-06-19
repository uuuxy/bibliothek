package pdf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRechnung_Success(t *testing.T) {
	schueler := Schueler{
		Vorname:    "Max",
		Nachname:   "Mustermann",
		Strasse:    "Musterstraße",
		Hausnummer: "1a",
		PLZ:        "12345",
		Ort:        "Musterstadt",
	}

	items := []RechnungItem{
		{
			Titel:        "Mathematik 1",
			Barcode:      "B-12345",
			Ausleihdatum: time.Now().Add(-30 * 24 * time.Hour),
			Ersatzpreis:  25.50,
		},
		{
			Titel:        "Deutsch 1",
			Barcode:      "B-67890",
			Ausleihdatum: time.Now().Add(-40 * 24 * time.Hour),
			Ersatzpreis:  15.00,
		},
	}

	pdfBytes, err := GenerateRechnung(schueler, items)

	assert.NoError(t, err)
	assert.NotNil(t, pdfBytes)
	assert.Greater(t, len(pdfBytes), 0, "PDF byte slice should not be empty")
}

func TestGenerateRechnung_EmptyItems(t *testing.T) {
	schueler := Schueler{
		Vorname:    "Anna",
		Nachname:   "Musterfrau",
		Strasse:    "Hauptstraße",
		Hausnummer: "10",
		PLZ:        "54321",
		Ort:        "Testdorf",
	}

	var items []RechnungItem // Empty items

	pdfBytes, err := GenerateRechnung(schueler, items)

	assert.NoError(t, err)
	assert.NotNil(t, pdfBytes)
	assert.Greater(t, len(pdfBytes), 0, "PDF byte slice should not be empty even without items")
}
