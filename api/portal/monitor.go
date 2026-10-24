package portal

import (
	"errors"
	"log/slog"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// MonitorFolien bedient GET /api/monitor/slides — den einzigen Endpunkt des
// Flur-Monitors: öffentlich, nur Titeldaten (docs/PII_MATRIX.de.md, Stufe 0).
//
// Was der Monitor zeigen darf, entscheidet repository.OeffentlichSichtbar, dieselbe Regel wie
// im Katalog (api/portal/opac.go). Die Folien baut repository.MonitorAbfragen.
func (t *Tueren) MonitorFolien() http.HandlerFunc {
	abfragen := repository.NewMonitorAbfragen(t.DB.Pool)
	return func(w http.ResponseWriter, r *http.Request) {
		folien, err := abfragen.LadeFolien(r.Context())
		if err != nil {
			// Echter Grund ins Log, nach außen nur die Gattung: Der Endpunkt ist ohne
			// Anmeldung erreichbar, ein Postgres-Fehlertext gehört nicht auf den Flur.
			slog.Error("DB Error in Monitor", "fehler", err)
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("internal server error"))
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, folien)
	}
}
