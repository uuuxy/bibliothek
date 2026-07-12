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
		book, ok := bookTitleAusFelder(parseKatalogisat(kat))
		if !ok {
			continue
		}
		books = append(books, book)
	}

	importedCount, err := s.bookRepo.BulkUpsertBookTitles(ctx, books)
	if err != nil {
		return 0, err
	}

	return importedCount, nil
}

// litteraFelder bündelt die aus einem Katalogisat extrahierten MAB-Rohfelder.
type litteraFelder struct {
	autor, titel, ort, verlag, isbn, jahrStr, signatur, standort string
}

// parseKatalogisat extrahiert die relevanten MAB-Felder eines Datensatzes.
func parseKatalogisat(kat Katalogisat) litteraFelder {
	var f litteraFelder
	for _, feld := range kat.Felder {
		mab := strings.TrimSpace(feld.MAB)
		val := strings.TrimSpace(feld.Value)
		val = strings.ReplaceAll(val, "¬", "")

		switch mab {
		case "100":
			f.autor = val
		case "108a":
			f.standort = val
		case "310":
			f.titel = val
		case "410":
			f.ort = val
		case "412":
			f.verlag = val
		case "425":
			f.jahrStr = val
		case "540":
			f.isbn = strings.ReplaceAll(strings.ReplaceAll(val, "-", ""), " ", "")
		case "700":
			if feld.Reihung == "1" {
				f.signatur = val
			}
		}
	}
	return f
}

// bookTitleAusFelder wandelt die Rohfelder in einen BookTitle um. ok=false bedeutet:
// der Datensatz hat keinen Titel und wird übersprungen.
func bookTitleAusFelder(f litteraFelder) (repository.BookTitle, bool) {
	if f.titel == "" {
		return repository.BookTitle{}, false // Ein Titel ist zwingend erforderlich
	}

	// Lernmittelfreiheit: Kennung aus Signatur ("LMF Bio 7") oder Standort-Feld 108a
	// ("LMF", "LMF/Bibliothek") → reine Fach-Signatur behalten und den Titel per
	// Projekt-Konvention "LMF-" flaggen.
	titel := f.titel
	signatur := f.signatur
	if hatLMFKennung(signatur) || hatLMFKennung(f.standort) {
		signatur = entferneLMFToken(signatur)
		titel = flaggeAlsSchulbuch(titel)
	}

	erscheinungsjahr := 0
	if len(f.jahrStr) >= 4 {
		// Versuche die ersten 4 Zeichen als Jahr zu parsen
		if y, err := strconv.Atoi(f.jahrStr[:4]); err == nil {
			erscheinungsjahr = y
		}
	}

	// Optional: Wenn Ort vorhanden ist, diesen mit dem Verlag kombinieren
	verlag := f.verlag
	if f.ort != "" && verlag != "" {
		verlag = f.ort + " : " + verlag
	} else if f.ort != "" {
		verlag = f.ort
	}

	return repository.BookTitle{
		Titel:            titel,
		Autor:            f.autor,
		ISBN:             f.isbn,
		Verlag:           verlag,
		Erscheinungsjahr: erscheinungsjahr,
		Signatur:         signatur,
	}, true
}
