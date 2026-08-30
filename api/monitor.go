package api

import (
	"errors"
	"log"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// GetMonitorSlidesHandler bedient GET /api/monitor/slides — den einzigen Endpunkt des
// Flur-Monitors: öffentlich, nur Titeldaten (docs/PII_MATRIX.de.md, Stufe 0).
//
// Was der Monitor zeigen darf, entscheidet repository.OeffentlichSichtbar — dieselbe
// Regel wie im Katalog (api/opac.go). Die Folien selbst baut repository.MonitorRepository;
// die Abfragen standen bis zum 30.08.2026 hier im Handler, ohne diese Regel.
func (s *Server) GetMonitorSlidesHandler() http.HandlerFunc {
	repo := repository.NewMonitorRepository(s.DB.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		slides, err := repo.LadeSlides(r.Context())
		if err != nil {
			// Echter Grund ins Log, nach außen nur die Gattung: Der Endpunkt ist ohne
			// Anmeldung erreichbar, ein Postgres-Fehlertext gehört nicht auf den Flur.
			log.Printf("DB Error in Monitor: %v", err)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("internal server error"))
			return
		}
		RespondJSON(w, http.StatusOK, slides)
	}
}
