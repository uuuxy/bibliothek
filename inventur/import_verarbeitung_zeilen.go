package inventur

import (
	"context"
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

	gradeLevel := parseKlassenStufe(getCol("klasse"), title)

	stock := 0
	if s, err := strconv.Atoi(getCol("bestand")); err == nil {
		stock = s
	}

	book := Book{
		ISBN:    isbn,
		Title:   title,
		Author:  author,
		Subject: subject,
		// #nosec G115 - gradeLevel is guaranteed to be 5-10 by parseKlassenStufe
		GradeLevel:  int16(gradeLevel),
		Stock:       stock,
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
func parseKlassenStufe(gradeStr string, title string) int {
	gradeLevel := 0
	if g, err := strconv.Atoi(gradeStr); err == nil {
		gradeLevel = g
	}
	if gradeLevel == 0 {
		gradeLevel = inferGradeLevelFromTitle(title)
	}
	if gradeLevel < 5 || gradeLevel > 10 {
		gradeLevel = 5
	}
	return gradeLevel
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
