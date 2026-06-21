package service

import (
	"context"
	"encoding/xml"
	"io"
	"log"
	"strconv"
	"strings"

	"bibliothek/repository"
)

// Katalogisate repräsentiert die Wurzel des MAB2 XML-Exports
type Katalogisate struct {
	XMLName xml.Name      `xml:"Katalogisate"`
	Items   []Katalogisat `xml:"Katalogisat"`
}

// Katalogisat bündelt die Felder für einen Datensatz (ein Buch/Medium).
type Katalogisat struct {
	Felder []Feld `xml:"Feld"`
}

// Feld repräsentiert ein einzelnes XML-Feld aus Littera mit MAB-Code und Wert.
type Feld struct {
	MAB   string `xml:"MAB,attr"`
	Value string `xml:",chardata"`
}

// ImportService stellt Geschäftslogik für Datenimporte bereit.
type ImportService struct {
	bookRepo repository.BookRepository
}

// NewImportService erstellt einen neuen ImportService.
func NewImportService(bookRepo repository.BookRepository) *ImportService {
	return &ImportService{bookRepo: bookRepo}
}

// ParseLitteraXML liest die MAB2-XML-Datei ein und speichert die Bücher in der Datenbank.
func (s *ImportService) ParseLitteraXML(ctx context.Context, xmlData io.Reader) (int, error) {
	decoder := xml.NewDecoder(xmlData)
	var root Katalogisate

	// Komplette XML-Struktur einlesen
	if err := decoder.Decode(&root); err != nil {
		return 0, err
	}

	importedCount := 0

	for _, kat := range root.Items {
		var autor, titel, ort, verlag, isbn, jahrStr string

		// Parsing-Logik & Mapping
		for _, feld := range kat.Felder {
			mab := strings.TrimSpace(feld.MAB)
			val := strings.TrimSpace(feld.Value)

			switch mab {
			case "100":
				autor = val
			case "310":
				// Sortierzeichen '¬' entfernen
				titel = strings.ReplaceAll(val, "¬", "")
			case "410":
				ort = val
			case "412":
				verlag = val
			case "425":
				jahrStr = val
			case "540":
				isbn = val
			}
		}

		if titel == "" {
			continue // Ein Titel ist zwingend erforderlich
		}

		erscheinungsjahr := 0
		if len(jahrStr) >= 4 {
			// Versuche die ersten 4 Zeichen als Jahr zu parsen
			if y, err := strconv.Atoi(jahrStr[:4]); err == nil {
				erscheinungsjahr = y
			}
		}

		// Optional: Wenn Ort vorhanden ist, diesen mit dem Verlag kombinieren
		if ort != "" && verlag != "" {
			verlag = ort + " : " + verlag
		} else if ort != "" {
			verlag = ort
		}

		book := repository.BookTitle{
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			Verlag:           verlag,
			Erscheinungsjahr: erscheinungsjahr,
		}

		// Speichere die bereinigten Buch-Objekte über unser Repository
		if err := s.bookRepo.UpsertBookTitle(ctx, book); err != nil {
			log.Printf("Fehler beim Speichern des Buches %q (ISBN: %s): %v", titel, isbn, err)
			continue
		}
		
		importedCount++
	}

	return importedCount, nil
}
