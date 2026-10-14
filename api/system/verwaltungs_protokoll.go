package system

import (
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// verwaltungsEintrag ist eine Zeile des Protokolls der Verwaltung, wie die Oberfläche sie
// anzeigt. AdminID ist bewusst nullbar: Systemgetriebene Einträge (Cron, Migration)
// haben keinen Bearbeiter, sollen aber trotzdem im Protokoll stehen.
type verwaltungsEintrag struct {
	ID          string    `json:"id"`
	AdminID     *string   `json:"admin_id"`
	AdminName   string    `json:"admin_name"`
	Aktion      string    `json:"aktion"`
	Details     any       `json:"details"`
	IPAdresse   string    `json:"ip_adresse"`
	Zeitstempel time.Time `json:"zeitstempel"`
}

// VerwaltungsProtokoll liefert die jüngsten 1000 Einträge des Protokolls der Verwaltung.
func (t *Tueren) VerwaltungsProtokoll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		zeilen, err := repository.LadeAdminProtokoll(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := []verwaltungsEintrag{}
		for _, z := range zeilen {
			liste = append(liste, verwaltungsEintrag(z))
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}
