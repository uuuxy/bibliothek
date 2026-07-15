package inventur

import (
	"context"
	"math"
	"strconv"
	"strings"
)

// ImportConfig bündelt die Parameter für die Verarbeitung einer Importzeile.
type ImportConfig struct {
	Ctx       context.Context
	Row       []string
	ColIdx    map[string]int
	Repo      *BookRepository
	Metadaten *MetadatenClient
}

// verarbeiteImportZeile verarbeitet eine einzelne Excel-Zeile und erstellt das
// entsprechende Book-Objekt für den Batch-Import.
func verarbeiteImportZeile(cfg ImportConfig) (*Book, error) {
	getCol := func(name string) string {
		idx := cfg.ColIdx[name]
		if idx >= 0 && idx < len(cfg.Row) {
			return strings.TrimSpace(cfg.Row[idx])
		}
		return ""
	}

	isbn := getCol("isbn")
	if isbn == "" {
		return nil, nil // Leere Zeile überspringen
	}

	title := getCol("titel")
	author := getCol("autor")
	subject := getCol("fach")
	if subject == "" {
		subject = "Unbekannt"
	}

	book := Book{
		ISBN:        isbn,
		Title:       title,
		Author:      author,
		Subject:     subject,
		GradeLevel:  parseKlassenStufe(getCol("klasse"), title),
		Stock:       parseBestand(getCol("bestand")),
		LastCounted: nil,
	}

	ergaenzeMetadaten(cfg.Ctx, cfg.Metadaten, &book)

	if book.Subject == "" || strings.EqualFold(book.Subject, "unbekannt") {
		if inferredSubject := inferSubjectFromTitle(book.Title); inferredSubject != "" {
			book.Subject = inferredSubject
		}
	}

	return &book, nil
}

// parseKlassenStufe versucht die Klassenstufe aus einem String zu extrahieren.
// Außerhalb von 5–10 gilt der Default 5. Early Return statt Clamp-Zuweisung:
// die int16-Konvertierung muss auf einem Pfad liegen, den der Bounds-Check
// exklusiv kontrolliert — nach einem Merge mit dem Default-Zweig gilt der
// Check statisch nicht mehr als Guard (go/incorrect-integer-conversion).
func parseKlassenStufe(gradeStr string, title string) int16 {
	gradeLevel := 0
	if g, err := strconv.Atoi(gradeStr); err == nil {
		gradeLevel = g
	}
	if gradeLevel == 0 {
		gradeLevel = inferGradeLevelFromTitle(title)
	}
	if gradeLevel < 5 || gradeLevel > 10 {
		return 5
	}
	return int16(gradeLevel)
}

// parseBestand liest den Bestand aus der Import-Spalte. Nur Werte aus [0, MaxInt32]
// werden übernommen: negative Bestände sind Datenfehler, und die DB-Spalte ist int4 —
// der Bulk-Upsert konvertiert nach int32. Ungültiges wird wie ein Parse-Fehler
// behandelt (Bestand 0).
func parseBestand(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > math.MaxInt32 {
		return 0
	}
	return n
}

// ergaenzeMetadaten ergänzt fehlende Buch-Metadaten über externe APIs.
func ergaenzeMetadaten(ctx context.Context, metadaten *MetadatenClient, book *Book) {
	if book.Title != "" && book.Author != "" && book.CoverURL != "" {
		return
	}

	lookup, _ := metadaten.SucheNachISBN(ctx, book.ISBN)  //nolint:errcheck
	if lookup == nil {
		return
	}

	if book.Title == "" || book.Title == "Unbekannter Titel" {
		book.Title = lookup.Titel
	}
	if book.Author == "" || book.Author == "Unbekannter Autor" {
		book.Author = lookup.Autor
	}
	if book.CoverURL == "" {
		book.CoverURL = lookup.CoverURL
	}

	if book.Title == "" {
		book.Title = "Unbekannter Titel"
	}
	if book.Author == "" {
		book.Author = "Unbekannter Autor"
	}
}
