package api

import (
	"context"
	"log"
	"strings"
	"time"
	"net/http"

	"bibliothek/db"
	"bibliothek/internal/service"
	"bibliothek/inventur"
	"bibliothek/pkg/safego"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchAndSaveCoverURL queries metadata APIs asynchronously for an ISBN and updates the book's cover_url.
func FetchAndSaveCoverURL(db *pgxpool.Pool, titleID string, isbn string) {
	isbn = strings.TrimSpace(isbn)
	if isbn == "" {
		return
	}

	go func() {
		defer safego.Guard("cover-fetch")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := inventur.NeuerMetadatenClient()
		res, err := client.SucheNachISBN(ctx, isbn)
		if err != nil {
			log.Printf("Cover Fetch: Failed to fetch metadata for ISBN %s: %v", isbn, err)
			return
		}

		if res.CoverURL == "" {
			log.Printf("Cover Fetch: No cover URL found for ISBN %s", isbn)
			return
		}

		// Update the buecher_titel table
		query := `UPDATE buecher_titel SET cover_url = $1 WHERE id = $2`
		_, err = db.Exec(ctx, query, res.CoverURL, titleID)
		if err != nil {
			log.Printf("Cover Fetch: Failed to update cover_url in database for title ID %s: %v", titleID, err)
		} else {
			log.Printf("Cover Fetch: Successfully updated cover_url for title ID %s to %s", titleID, res.CoverURL)
		}
	}()
}

// SyncCoversHandler is the HTTP handler that triggers the asynchronous cover download.
func SyncCoversHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coverSvc := service.NewCoverService(dbPool)
		go coverSvc.SyncMissingCoversAsync()

		RespondJSON(w, http.StatusOK, map[string]string{
			"status": "success",
			"message": "Der Hintergrund-Job zum Herunterladen fehlender Cover wurde gestartet.",
		})
	}
}
