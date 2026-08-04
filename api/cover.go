package api

import (
	"net/http"

	"bibliothek/db"
	"bibliothek/internal/service"
)

// SyncCoversHandler is the HTTP handler that triggers the asynchronous cover download.
func SyncCoversHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coverSvc := service.NewCoverService(dbPool)
		go coverSvc.SyncMissingCoversAsync()

		RespondJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Der Hintergrund-Job zum Herunterladen fehlender Cover wurde gestartet.",
		})
	}
}
