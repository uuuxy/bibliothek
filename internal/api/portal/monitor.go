package portal

import (
	"fmt"
	"net/http"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/repository"
)

// MonitorFolien bedient GET /api/monitor/slides — den einzigen Endpunkt des
// Flur-Monitors: öffentlich, nur Titeldaten und die heutigen Zeilen des veröffentlichten
// LMF-Plans (docs/PII_MATRIX.de.md, Stufe 0).
//
// Was der Monitor zeigen darf, entscheidet repository.OeffentlichSichtbar, dieselbe Regel wie
// im Katalog (internal/api/portal/opac.go). Die Folien baut repository.MonitorAbfragen.
func (t *Tueren) MonitorFolien() http.HandlerFunc {
	abfragen := repository.NewMonitorAbfragen(t.DB.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		folien, err := abfragen.LadeFolien(r.Context(), t.Jetzt())
		if err != nil {
			// Echter Grund ins Log, nach außen nur die Gattung: Der Endpunkt ist ohne
			// Anmeldung erreichbar, ein Postgres-Fehlertext gehört nicht auf den Flur.
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("monitor laden: %w", err))
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, folien)
	}
}
