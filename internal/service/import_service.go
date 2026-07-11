package service

import (
	"context"
	"encoding/xml"
	"io"
	"strconv"
	"strings"

	"bibliothek/db"
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
	MAB     string `xml:"MAB,attr"`
	Reihung string `xml:"Reihung,attr"`
	Value   string `xml:",chardata"`
}

// ImportService stellt Geschäftslogik für Datenimporte bereit.
type ImportService struct {
	bookRepo repository.BookRepository
	db       db.PgxPoolIface
}

// NewImportService erstellt einen neuen ImportService.
func NewImportService(bookRepo repository.BookRepository, dbPool db.PgxPoolIface) *ImportService {
	return &ImportService{bookRepo: bookRepo, db: dbPool}
}

// ParseLitteraXML liest die MAB2-XML-Datei ein und speichert die Bücher in der Datenbank.
func (s *ImportService) ParseLitteraXML(ctx context.Context, xmlData io.Reader) (int, error) {
	decoder := xml.NewDecoder(xmlData)
	var root Katalogisate

	// Komplette XML-Struktur einlesen
	if err := decoder.Decode(&root); err != nil {
		return 0, err
	}

	// Titel sammeln und in EINEM gepipelineten Batch schreiben (statt je Titel
	// eine eigene DB-Rundreise) — der frühere N+1 ließ den Import mit ~15.000
	// Titeln gegen eine nicht-lokale DB in Client-Timeouts laufen.
	books := make([]repository.BookTitle, 0, len(root.Items))

	for _, kat := range root.Items {
		var autor, titel, ort, verlag, isbn, jahrStr, signatur, standort string

		// Parsing-Logik & Mapping
		for _, feld := range kat.Felder {
			mab := strings.TrimSpace(feld.MAB)
			val := strings.TrimSpace(feld.Value)
			val = strings.ReplaceAll(val, "¬", "")

			switch mab {
			case "100":
				autor = val
			case "108a":
				standort = val
			case "310":
				titel = val
			case "410":
				ort = val
			case "412":
				verlag = val
			case "425":
				jahrStr = val
			case "540":
				isbn = strings.ReplaceAll(strings.ReplaceAll(val, "-", ""), " ", "")
			case "700":
				if feld.Reihung == "1" {
					signatur = val
				}
			}
		}

		if titel == "" {
			continue // Ein Titel ist zwingend erforderlich
		}

		// Lernmittelfreiheit: Kennung aus Signatur ("LMF Bio 7") oder
		// Standort-Feld 108a ("LMF", "LMF/Bibliothek") → reine Fach-Signatur
		// behalten und den Titel per Projekt-Konvention "LMF-" flaggen.
		if hatLMFKennung(signatur) || hatLMFKennung(standort) {
			signatur = entferneLMFToken(signatur)
			titel = flaggeAlsSchulbuch(titel)
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

		books = append(books, repository.BookTitle{
			Titel:            titel,
			Autor:            autor,
			ISBN:             isbn,
			Verlag:           verlag,
			Erscheinungsjahr: erscheinungsjahr,
			Signatur:         signatur,
		})
	}

	importedCount, err := s.bookRepo.BulkUpsertBookTitles(ctx, books)
	if err != nil {
		return 0, err
	}

	return importedCount, nil
}
