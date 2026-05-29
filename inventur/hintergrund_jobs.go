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

// brauchtAktualisierung prüft, ob ein Buch fehlende Cover oder Kategorien hat.
func brauchtAktualisierung(b Book) bool {
	if b.CoverURL == "" || b.CoverURL == "https://covers.openlibrary.org/b/isbn/-L.jpg" ||
		b.CoverURL == fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", b.ISBN) {
		return true
	}
	if b.Subject == "" || b.Subject == "Kein Fach" || b.GradeLevel == 0 {
		return true
	}
	return false
}

// aktualisiereEinzelnesBuch aktualisiert Cover und Kategorie für ein einzelnes Buch.
func aktualisiereEinzelnesBuch(ctx context.Context, repo *BookRepository, client *MetadatenClient, b Book, cache *sync.Map) (coverUpdated int, catUpdated int) {
	var nachschlagen *MetadatenErgebnis

	if cachedVal, ok := cache.Load(b.ISBN); ok {
		nachschlagen = cachedVal.(*MetadatenErgebnis)
	} else {
		var err error
		nachschlagen, err = client.SucheNachISBN(ctx, b.ISBN)
		if err != nil || nachschlagen == nil {
			return 0, 0
		}
		cache.Store(b.ISBN, nachschlagen)
	}

	// Cover aktualisieren
	if nachschlagen.CoverURL != "" && (b.CoverURL == "" ||
		b.CoverURL == "https://covers.openlibrary.org/b/isbn/-L.jpg" ||
		b.CoverURL == fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", b.ISBN)) {
		if err := repo.UpdateBookMetadata(ctx, b.ID, b.Title, b.Author, nachschlagen.CoverURL); err == nil {
			coverUpdated = 1
			log.Printf("Cover für ISBN %s aktualisiert", b.ISBN)
		}
	}

	// Kategorie/Klasse aktualisieren
	if nachschlagen.Fach != "" || nachschlagen.KlassenStufe != "" {
		newSubject := b.Subject
		if b.Subject == "" || b.Subject == "Kein Fach" {
			newSubject = nachschlagen.Fach
		}

		newGrade := b.GradeLevel
		if b.GradeLevel == 0 && nachschlagen.KlassenStufe != "" {
			var parsedGrade int
			fmt.Sscanf(nachschlagen.KlassenStufe, "%d", &parsedGrade)
			if parsedGrade >= 5 && parsedGrade <= 13 {
				newGrade = int16(parsedGrade)
			}
		}

		if newSubject != b.Subject || newGrade != b.GradeLevel {
			if err := repo.UpdateBookCategory(ctx, b.ID, newSubject, newGrade); err == nil {
				catUpdated = 1
				log.Printf("Kategorie/Klasse für ISBN %s aktualisiert (%s, Kl. %d)", b.ISBN, newSubject, newGrade)
			}
		}
	}

	return coverUpdated, catUpdated
}
