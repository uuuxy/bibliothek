package service

import (
	"context"
	"log"
	"time"

	"bibliothek/db"
	"bibliothek/inventur"
)

// CoverService handles fetching book covers asynchronously.
type CoverService struct {
	db db.PgxPoolIface
}

// NewCoverService creates a new CoverService.
func NewCoverService(dbPool db.PgxPoolIface) *CoverService {
	return &CoverService{
		db: dbPool,
	}
}

// SyncMissingCoversAsync finds all titles without a cover URL and fetches them sequentially,
// with a 500ms delay to prevent rate-limiting.
func (s *CoverService) SyncMissingCoversAsync() {
	ctx := context.Background()

	// Hole alle Titel, die kein Cover haben, aber eine ISBN besitzen
	query := `SELECT id, isbn FROM buecher_titel WHERE (cover_url IS NULL OR cover_url = '') AND isbn IS NOT NULL AND isbn != ''`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		log.Printf("Cover Sync: Fehler beim Abrufen der fehlenden Cover: %v", err)
		return
	}

	type missingCover struct {
		ID   string
		ISBN string
	}

	var missing []missingCover
	for rows.Next() {
		var mc missingCover
		if err := rows.Scan(&mc.ID, &mc.ISBN); err == nil {
			missing = append(missing, mc)
		}
	}
	rows.Close()

	if len(missing) == 0 {
		log.Println("Cover Sync: Keine fehlenden Cover gefunden.")
		return
	}

	log.Printf("Cover Sync: Starte Download für %d fehlende Cover...", len(missing))

	client := inventur.NeuerMetadatenClient()

	for _, mc := range missing {
		// Timeout für einzelne API Requests
		reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		res, err := client.SucheNachISBN(reqCtx, mc.ISBN)
		cancel()

		if err != nil {
			log.Printf("Cover Sync: Fehler bei ISBN %s: %v", mc.ISBN, err)
		} else if res.CoverURL != "" {
			// Update DB
			updateQuery := `UPDATE buecher_titel SET cover_url = $1 WHERE id = $2`
			if _, err := s.db.Exec(ctx, updateQuery, res.CoverURL, mc.ID); err != nil {
				log.Printf("Cover Sync: DB-Update für Titel %s fehlgeschlagen: %v", mc.ID, err)
			} else {
				log.Printf("Cover Sync: Cover für ISBN %s aktualisiert.", mc.ISBN)
			}
		}

		// Zwingendes Rate-Limiting
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("Cover Sync: Abgeschlossen.")
}
