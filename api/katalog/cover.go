package katalog

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/internal/service"
)

// SyncCoversHandler is the HTTP handler that triggers the asynchronous cover download.
func (s *Tueren) SyncCoversHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		coverSvc := service.NewCoverDienst(s.DB.Pool)
		go coverSvc.GleicheFehlendeCoverAb()

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Der Hintergrund-Job zum Herunterladen fehlender Cover wurde gestartet.",
		})
	}
}
