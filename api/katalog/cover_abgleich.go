package katalog

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/internal/service"
)

// CoverAbgleichen is the HTTP handler that triggers the asynchronous cover download.
func (t *Tueren) CoverAbgleichen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dienst := service.NewCoverDienst(t.DB.Pool)
		go dienst.GleicheFehlendeCoverAb()

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Der Hintergrund-Job zum Herunterladen fehlender Cover wurde gestartet.",
		})
	}
}
