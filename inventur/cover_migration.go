package inventur

import (
	"bibliothek/db"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// RunCoverMigration lädt alle Cover, die noch als externe HTTP-Links (z.B. OpenLibrary) in der DB stehen,
// einzeln herunter und aktualisiert die Datenbank auf den lokalen "/uploads/..." Pfad.
func RunCoverMigration(db db.PgxPoolIface) {
	log.Println("=== Starte automatische Cover-Migration ===")
	ctx := context.Background()

	// Holt alle Bücher, deren Cover-URL mit "http" beginnt (also nicht lokal "/uploads/..." ist)
	// Holt alle Bücher, deren Cover-URL mit "http" beginnt (also nicht lokal "/uploads/..." ist)
	rows, err := db.Query(ctx, "SELECT id, isbn, cover_url, titel AS title FROM buecher_titel WHERE cover_url LIKE 'http%'")
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der Bücher: %v", err)
	}

	type BookToMigrate struct {
		ID       string
		ISBN     string
		CoverURL string
		Title    string
	}

	var books []BookToMigrate
	for rows.Next() {
		var b BookToMigrate
		if err := rows.Scan(&b.ID, &b.ISBN, &b.CoverURL, &b.Title); err != nil {
			log.Printf("Fehler beim Lesen einer Zeile: %v", err)
			continue
		}
		books = append(books, b)
	}
	rows.Close() // Explizit schließen bevor wir andere Queries machen

	log.Printf("Es wurden %d Bücher mit externen Covern gefunden. Starte Download...", len(books))

	client := &http.Client{Timeout: 10 * time.Second}
	erfolgreich := 0
	fehlerhaft := 0

	var updatedIDs []string
	var updatedURLs []string

	for i, b := range books {
		log.Printf("[%d/%d] Bearbeite Buch ID %s '%s' (ISBN: %s)", i+1, len(books), b.ID, b.Title, b.ISBN)

		if b.CoverURL == "" {
			continue
		}
		// Versuch 1: Generiere DNB Cover-URL, wenn keine vorhanden ist
		// (Die DNB liefert sehr hochauflösende Cover, oft besser als Buchverlage selbst)
		isbn13 := konvertiereISBN10zu13(b.ISBN)
		dnbCoverURL := fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", isbn13)

		lokalerPfad := ""

		// Wenn kein Cover vorhanden ist ODER der Link zu einem kleinen OpenLibrary-Thumbnail zeigt, probieren wir DNB
		if b.CoverURL == "" || strings.HasPrefix(b.CoverURL, "http") {
			// Wir checken zuerst unverbindlich, ob die DNB das Bild überhaupt hat, bevor wir herunterladen
			testReq, err := http.NewRequestWithContext(ctx, http.MethodHead, dnbCoverURL, nil)
			if err == nil {
				testReq.Header.Set("User-Agent", "Mozilla/5.0 (Inventur/1.0)")
				if testResp, err := client.Do(testReq); err == nil && testResp.StatusCode == http.StatusOK {
					b.CoverURL = dnbCoverURL
				}
			}
		}

		// Falls OpenLibrary-Fallback (klein -> groß tauschen)
		if strings.HasSuffix(b.CoverURL, "-S.jpg") && strings.Contains(b.CoverURL, "openlibrary") {
			b.CoverURL = strings.Replace(b.CoverURL, "-S.jpg", "-L.jpg", 1)
		}

		// Jetzt laden wir entweder von DNB, OpenLibrary, dem Verlag (aus b.CoverURL) herunter
		lokalerPfad = downloadAndSaveCoverLocally(ctx, client, b.CoverURL, b.ISBN)

		if lokalerPfad == "" || strings.HasPrefix(lokalerPfad, "http") || lokalerPfad == b.CoverURL {
			log.Printf("  -> Konnte Cover nicht umwandeln (bleibt extern).")
			fehlerhaft++
			continue
		}

		updatedIDs = append(updatedIDs, b.ID)
		updatedURLs = append(updatedURLs, lokalerPfad)
		log.Printf("  -> Erfolgreich heruntergeladen als: %s", lokalerPfad)

		// Wir pausieren ganz kurz, um die Server von OpenLibrary/DNB nicht zu überlasten (Rate Limiting)
		time.Sleep(300 * time.Millisecond)
	}

	if len(updatedIDs) > 0 {
		query := `
			UPDATE buecher_titel
			SET cover_url = data.cover_url
			FROM (SELECT UNNEST($1::text[])::uuid AS id, UNNEST($2::text[]) AS cover_url) AS data
			WHERE buecher_titel.id = data.id
		`
		_, err := db.Exec(ctx, query, updatedIDs, updatedURLs)
		if err != nil {
			log.Printf("Fehler beim Batch-Speichern der neuen Pfade in der DB: %v", err)
			fehlerhaft += len(updatedIDs)
		} else {
			erfolgreich += len(updatedIDs)
		}
	}

	log.Println("=== Cover-Migration abgeschlossen ===")
	log.Printf("Erfolgreich umgewandelt: %d, Fehlerhaft (oder leer): %d", erfolgreich, fehlerhaft)
}
