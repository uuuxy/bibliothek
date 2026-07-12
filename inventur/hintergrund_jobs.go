package inventur

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// StarteHintergrundAktualisierung lädt automatisch fehlende Cover und Kategorien
// für bestehende Bücher im Hintergrund nach. Wird beim Server-Start einmalig
// als Goroutine gestartet und respektiert den übergebenen Context für Abbruch.
func StarteHintergrundAktualisierung(ctx context.Context, repo *BookRepository, metadatenClient *MetadatenClient) {
	time.Sleep(5 * time.Second) // Server starten lassen
	log.Println("Starte automatische Cover- und Kategorie-Aktualisierung im Hintergrund...")

	books, err := repo.ListBooks(ctx, "", nil, "")
	if err != nil {
		log.Printf("Fehler beim Laden der Bücher für Update: %v", err)
		return
	}

	updatedCovers := 0
	updatedCategories := 0
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	var mu sync.Mutex

	var cache sync.Map

	for _, b := range books {
		if ctx.Err() != nil {
			break
		}

		if !brauchtAktualisierung(b) {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(b Book) {
			defer wg.Done()
			defer func() { <-sem }()

			coverUpdated, catUpdated := aktualisiereEinzelnesBuch(ctx, repo, metadatenClient, b, &cache)
			mu.Lock()
			updatedCovers += coverUpdated
			updatedCategories += catUpdated
			mu.Unlock()

			time.Sleep(1 * time.Second) // Rate limiting
		}(b)
	}

	wg.Wait()
	log.Printf("Automatische Aktualisierung abgeschlossen. %d Cover und %d Kategorien aktualisiert.", updatedCovers, updatedCategories)
}

const openLibraryLeeresCover = "https://covers.openlibrary.org/b/isbn/-L.jpg"

// istPlatzhalterCover erkennt ein fehlendes oder generisches OpenLibrary-Platzhaltercover.
func istPlatzhalterCover(coverURL, isbn string) bool {
	return coverURL == "" ||
		coverURL == openLibraryLeeresCover ||
		coverURL == fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", isbn)
}

// brauchtAktualisierung prüft, ob ein Buch fehlende Cover oder Kategorien hat.
func brauchtAktualisierung(b Book) bool {
	if istPlatzhalterCover(b.CoverURL, b.ISBN) {
		return true
	}
	if b.Subject == "" || b.Subject == "Kein Fach" || b.GradeLevel == 0 {
		return true
	}
	return false
}

// ladeMetadaten liefert das (ggf. gecachte) Nachschlage-Ergebnis für eine ISBN;
// nil bedeutet: kein Treffer bzw. Fehler beim Nachschlagen.
func ladeMetadaten(ctx context.Context, client *MetadatenClient, isbn string, cache *sync.Map) *MetadatenErgebnis {
	if cachedVal, ok := cache.Load(isbn); ok {
		return cachedVal.(*MetadatenErgebnis)  //nolint:errcheck
	}
	nachschlagen, err := client.SucheNachISBN(ctx, isbn)
	if err != nil || nachschlagen == nil {
		return nil
	}
	cache.Store(isbn, nachschlagen)
	return nachschlagen
}

// aktualisiereCover schreibt das nachgeschlagene Cover, sofern das bisherige fehlt oder
// generisch ist. Liefert 1 bei erfolgtem Update.
func aktualisiereCover(ctx context.Context, repo *BookRepository, b Book, nachschlagen *MetadatenErgebnis) int {
	if nachschlagen.CoverURL == "" || !istPlatzhalterCover(b.CoverURL, b.ISBN) {
		return 0
	}
	if err := repo.UpdateBookMetadata(ctx, b.ID, b.Title, b.Author, nachschlagen.CoverURL); err != nil {
		return 0
	}
	log.Printf("Cover für ISBN %s aktualisiert", b.ISBN)
	return 1
}

// aktualisiereKategorie ergänzt fehlendes Fach/fehlende Klassenstufe aus dem
// Nachschlage-Ergebnis. Liefert 1 bei erfolgtem Update.
func aktualisiereKategorie(ctx context.Context, repo *BookRepository, b Book, nachschlagen *MetadatenErgebnis) int {
	if nachschlagen.Fach == "" && nachschlagen.KlassenStufe == "" {
		return 0
	}

	newSubject := b.Subject
	if b.Subject == "" || b.Subject == "Kein Fach" {
		newSubject = nachschlagen.Fach
	}

	newGrade := b.GradeLevel
	if b.GradeLevel == 0 && nachschlagen.KlassenStufe != "" {
		var parsedGrade int
		_, _ = fmt.Sscanf(nachschlagen.KlassenStufe, "%d", &parsedGrade)  //nolint:errcheck
		if parsedGrade >= 5 && parsedGrade <= 13 {
			newGrade = int16(parsedGrade)
		}
	}

	if newSubject == b.Subject && newGrade == b.GradeLevel {
		return 0
	}
	if err := repo.UpdateBookCategory(ctx, b.ID, newSubject, newGrade); err != nil {
		return 0
	}
	log.Printf("Kategorie/Klasse für ISBN %s aktualisiert (%s, Kl. %d)", b.ISBN, newSubject, newGrade)
	return 1
}

// aktualisiereEinzelnesBuch aktualisiert Cover und Kategorie für ein einzelnes Buch.
func aktualisiereEinzelnesBuch(ctx context.Context, repo *BookRepository, client *MetadatenClient, b Book, cache *sync.Map) (coverUpdated int, catUpdated int) {
	nachschlagen := ladeMetadaten(ctx, client, b.ISBN, cache)
	if nachschlagen == nil {
		return 0, 0
	}
	return aktualisiereCover(ctx, repo, b, nachschlagen), aktualisiereKategorie(ctx, repo, b, nachschlagen)
}
