package inventur

import (
	"context"
	"math"
	"strconv"
	"strings"

	"bibliothek/pkg/lmf"
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
	// Die Fach-Spalte ist Freitext, buecher_titel.subject aber ein Fremdschlüssel auf die
	// Systematik (Migration 078): Was hier durchgeht, wird eine Kategorie und steht danach
	// in der Fach-Auswahl der Maske und als Fach im Portal. Deshalb dieselbe Prüfung wie
	// im CSV-Bestandsimport seit dem 03.09.2026 (fachDerZeile): nur ein echter Fachname
	// oder eine bekannte Schreibvariante, sonst leer.
	//
	// Der frühere Rückfall auf das wörtliche Fach „Unbekannt" ist damit entfallen — er
	// legte eine Systematik-Kategorie dieses Namens an, die kein Fach ist. Leer heißt
	// überall sonst „ohne Fach", und genau das ist hier gemeint.
	subject := lmf.FachExakt(getCol("fach"))

	book := Book{
		ISBN:        isbn,
		Title:       title,
		Author:      author,
		Subject:     subject,
		GradeLevel:  parseKlassenStufe(getCol("klasse")),
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

// parseKlassenStufe liest die Klassenstufe aus der Spalte „klasse". Gültig ist 5–13
// (kooperative Gesamtschule inkl. Oberstufe). Fehlt die Spalte, ist sie leer oder liegt der
// Wert daneben, ist die Klasse unbekannt (0); beide Upserts schreiben dafür NULL — dieselbe
// Regel wie Littera-Übernahme und Sammelimport (NULLIF(…, 0)).
//
// Bis zum 22.09.2026 riet der Import stattdessen: erst die erste Zahl im Titel („Die 13½
// Leben des Käpt'n Blaubär" bekam Klasse 13), sonst die Vorgabe 5. Eine geratene Klasse ist
// in der Datenbank von einer gepflegten nicht zu unterscheiden, und das Upsert behält eine
// vorhandene Klasse ungleich 0 — eine spätere Liste mit der echten Klasse kam gegen die
// geratene nicht mehr an (docs/OFFEN.md 5.5).
//
// Early Return statt Clamp-Zuweisung: die int16-Konvertierung muss auf
// einem Pfad liegen, den der Bounds-Check exklusiv kontrolliert — nach einem Merge
// mit dem Default-Zweig gilt der Check statisch nicht mehr als Guard
// (go/incorrect-integer-conversion).
func parseKlassenStufe(gradeStr string) int16 {
	gradeLevel := 0
	if g, err := strconv.Atoi(gradeStr); err == nil {
		gradeLevel = g
	}
	if gradeLevel < 5 || gradeLevel > 13 {
		return 0
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

	lookup, _ := metadaten.SucheNachISBN(ctx, book.ISBN) //nolint:errcheck
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
