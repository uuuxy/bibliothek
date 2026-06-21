package service

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"

	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
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

	importedCount := 0

	for _, kat := range root.Items {
		var autor, titel, ort, verlag, isbn, jahrStr, signatur string

		// Parsing-Logik & Mapping
		for _, feld := range kat.Felder {
			mab := strings.TrimSpace(feld.MAB)
			val := strings.TrimSpace(feld.Value)
			val = strings.ReplaceAll(val, "¬", "")

			switch mab {
			case "100":
				autor = val
			case "310":
				titel = val
			case "410":
				ort = val
			case "412":
				verlag = val
			case "425":
				jahrStr = val
			case "540":
				isbn = val
			case "700 ":
				if feld.Reihung == "1" {
					signatur = val
				}
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
			Signatur:         signatur,
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

// ImportLitteraBestand liest eine finale Bestands-CSV (Trennzeichen ';') und importiert
// Titel sowie Exemplare über eine einzige SQL-Transaktion.
// Spalten: Titel;Autor;Verlag;ISBN;Jahr;Kategorie;Barcode;Zustand
func (s *ImportService) ImportLitteraBestand(ctx context.Context, csvData io.Reader) (int, int, error) {
	reader := csv.NewReader(csvData)
	reader.Comma = ';'
	reader.LazyQuotes = true

	rows, err := reader.ReadAll()
	if err != nil {
		return 0, 0, err
	}
	if len(rows) < 2 {
		return 0, 0, nil // Leer oder nur Header
	}

	// Kopfzeile prüfen (Index 0 bis 7)
	header := rows[0]
	if len(header) < 8 {
		return 0, 0, errors.New("CSV hat nicht die benötigten 8 Spalten")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	newTitlesCount := 0
	importedCopiesCount := 0

		batch := &pgx.Batch{}

		qCombined := `
			WITH t AS (
				INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject)
				VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), $6)
				ON CONFLICT (isbn) DO UPDATE SET 
					titel = EXCLUDED.titel, autor = EXCLUDED.autor, 
					verlag = EXCLUDED.verlag, erscheinungsjahr = EXCLUDED.erscheinungsjahr, 
					subject = EXCLUDED.subject
				RETURNING id, (xmax = 0) AS is_new_titel
			), fb_sel AS (
				SELECT id, false AS is_new_titel FROM buecher_titel 
				WHERE titel = $1 AND (isbn IS NULL OR isbn = '') 
				AND NOT EXISTS (SELECT 1 FROM t)
			), fb_ins AS (
				INSERT INTO buecher_titel (titel, autor, verlag, isbn, erscheinungsjahr, subject)
				SELECT $1, $2, $3, NULLIF($4, ''), NULLIF($5, 0), $6
				WHERE NOT EXISTS (SELECT 1 FROM t) AND NOT EXISTS (SELECT 1 FROM fb_sel)
				RETURNING id, true AS is_new_titel
			), final_titel AS (
				SELECT id, is_new_titel FROM t UNION ALL SELECT id, is_new_titel FROM fb_sel UNION ALL SELECT id, is_new_titel FROM fb_ins
			), ex_ins AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, ist_ausleihbar, zustand_notiz)
				SELECT id, $7, CURRENT_DATE, $8, $9
				FROM final_titel
				ON CONFLICT (barcode_id) DO UPDATE SET 
					zustand_notiz = EXCLUDED.zustand_notiz, 
					ist_ausleihbar = EXCLUDED.ist_ausleihbar, 
					aktualisiert_am = CURRENT_TIMESTAMP
				RETURNING id
			)
			SELECT 
				COALESCE((SELECT is_new_titel FROM final_titel LIMIT 1), false) AS titel_inserted,
				EXISTS(SELECT 1 FROM ex_ins) AS exemplar_inserted
		`

	for _, row := range rows[1:] {
		if len(row) < 8 {
			continue // Zeile ignorieren, wenn zu kurz
		}

		titel := strings.TrimSpace(row[0])
		autor := strings.TrimSpace(row[1])
		verlag := strings.TrimSpace(row[2])
		isbn := strings.TrimSpace(row[3])
		jahrStr := strings.TrimSpace(row[4])
		kategorie := strings.TrimSpace(row[5])
		barcode := strings.TrimSpace(row[6])
		zustandCSV := strings.TrimSpace(row[7])

		if titel == "" || barcode == "" {
			continue
		}

		jahr, _ := strconv.Atoi(jahrStr)

		// Zustand mappen (verfuegbar oder verliehen etc.)
		istAusleihbar := true
		if strings.ToLower(zustandCSV) == "verliehen" {
			istAusleihbar = false
		}

		batch.Queue(qCombined, titel, autor, verlag, isbn, jahr, kategorie, barcode, istAusleihbar, zustandCSV)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		var titelInserted, exemplarInserted bool
		err := br.QueryRow().Scan(&titelInserted, &exemplarInserted)
		if err == nil {
			if titelInserted {
				newTitlesCount++
			}
			if exemplarInserted {
				importedCopiesCount++
			}
		}
	}
	if err := br.Close(); err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}

	return newTitlesCount, importedCopiesCount, nil
}

