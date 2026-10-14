package katalog

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/service"
)

// SyncCoversHandler is the HTTP handler that triggers the asynchronous cover download.
func SyncCoversHandler(dbPool db.PgxPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coverSvc := service.NewCoverDienst(dbPool)
		go coverSvc.GleicheFehlendeCoverAb()

		tuer.RespondJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Der Hintergrund-Job zum Herunterladen fehlender Cover wurde gestartet.",
		})
	}
}
