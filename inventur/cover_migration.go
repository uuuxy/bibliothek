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

// coverMigrationBuch ist ein Titel-Datensatz, dessen Cover-URL noch extern (http…) ist.
type coverMigrationBuch struct {
	ID       string
	ISBN     string
	CoverURL string
	Title    string
}

// ladeExterneCoverBuecher lädt alle Titel mit externer Cover-URL (LIKE 'http%').
// ok=false bedeutet: Fehler beim Iterieren (bereits protokolliert). Ein Fehler bei der
// Query selbst bricht den Prozess (log.Fatalf) wie bisher hart ab.
func ladeExterneCoverBuecher(ctx context.Context, database db.PgxPoolIface) ([]coverMigrationBuch, bool) {
	rows, err := database.Query(ctx, "SELECT id, isbn, cover_url, titel AS title FROM buecher_titel WHERE cover_url LIKE 'http%'")
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der Bücher: %v", err)
	}

	var books []coverMigrationBuch
	for rows.Next() {
		var b coverMigrationBuch
		if err := rows.Scan(&b.ID, &b.ISBN, &b.CoverURL, &b.Title); err != nil {
			log.Printf("Fehler beim Lesen einer Zeile: %v", err)
			continue
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Printf("Cover-Migration: Fehler beim Lesen der Buchliste: %v", err)
		return nil, false
	}
	rows.Close() // Explizit schließen bevor wir andere Queries machen
	return books, true
}

// migriereEinzelnesCover lädt das (ggf. über DNB/OpenLibrary aufgewertete) Cover eines
// Buchs lokal herunter und aktualisiert die DB. Liefert (erfolgreich, fehlerhaft); ein
// leeres Cover wird stillschweigend übersprungen (beide false).
func migriereEinzelnesCover(ctx context.Context, client *http.Client, database db.PgxPoolIface, b coverMigrationBuch) (erfolgreich, fehlerhaft bool) {
	if b.CoverURL == "" {
		return false, false
	}

	// Versuch 1: Generiere DNB Cover-URL, wenn keine vorhanden ist
	// (Die DNB liefert sehr hochauflösende Cover, oft besser als Buchverlage selbst)
	isbn13 := konvertiereISBN10zu13(b.ISBN)
	dnbCoverURL := fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", isbn13)

	// Wenn kein Cover vorhanden ist ODER der Link zu einem kleinen OpenLibrary-Thumbnail zeigt, probieren wir DNB
	if b.CoverURL == "" || strings.HasPrefix(b.CoverURL, "http") {
		// Wir checken zuerst unverbindlich, ob die DNB das Bild überhaupt hat, bevor wir herunterladen
		// #nosec G107 - Hostname is hardcoded in dnbCoverURL
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
	lokalerPfad := downloadAndSaveCoverLocally(ctx, client, b.CoverURL, b.ISBN)

	if lokalerPfad == "" || strings.HasPrefix(lokalerPfad, "http") || lokalerPfad == b.CoverURL {
		log.Printf("  -> Konnte Cover nicht umwandeln (bleibt extern).")
		return false, true
	}

	// Update in der Datenbank
	if _, err := database.Exec(ctx, "UPDATE buecher_titel SET cover_url = $1 WHERE id = $2", lokalerPfad, b.ID); err != nil {
		log.Printf("  -> Fehler beim Speichern des neuen Pfades in der DB: %v", err)
		fehlerhaft = true
	} else {
		log.Printf("  -> Erfolgreich gespeichert als: %s", lokalerPfad)
		erfolgreich = true
	}

	// Wir pausieren ganz kurz, um die Server von OpenLibrary/DNB nicht zu überlasten (Rate Limiting)
	time.Sleep(300 * time.Millisecond)
	return erfolgreich, fehlerhaft
}

// RunCoverMigration lädt alle Cover, die noch als externe HTTP-Links (z.B. OpenLibrary) in der DB stehen,
// einzeln herunter und aktualisiert die Datenbank auf den lokalen "/uploads/..." Pfad.
func RunCoverMigration(db db.PgxPoolIface) {
	log.Println("=== Starte automatische Cover-Migration ===")
	ctx := context.Background()

	// Holt alle Bücher, deren Cover-URL mit "http" beginnt (also nicht lokal "/uploads/..." ist)
	books, ok := ladeExterneCoverBuecher(ctx, db)
	if !ok {
		return
	}

	log.Printf("Es wurden %d Bücher mit externen Covern gefunden. Starte Download...", len(books))

	client := &http.Client{Timeout: 10 * time.Second}
	erfolgreich := 0
	fehlerhaft := 0

	for i, b := range books {
		log.Printf("[%d/%d] Bearbeite Buch ID %s '%s' (ISBN: %s)", i+1, len(books), b.ID, b.Title, b.ISBN)

		erfolg, fehler := migriereEinzelnesCover(ctx, client, db, b)
		if erfolg {
			erfolgreich++
		}
		if fehler {
			fehlerhaft++
		}
	}

	log.Println("=== Cover-Migration abgeschlossen ===")
	log.Printf("Erfolgreich umgewandelt: %d, Fehlerhaft (oder leer): %d", erfolgreich, fehlerhaft)
}
