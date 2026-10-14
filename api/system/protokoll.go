package system

// audit_handler.go — Handler for the immutable audit log.
// The audit trail records all sensitive delete/cancel operations performed by staff.

import (
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// protokollEintrag represents a joined row in the audit log table.
type protokollEintrag struct {
	ID                 string    `json:"id"`
	Tabelle            string    `json:"tabelle"`
	Aktion             string    `json:"aktion"`
	DatensatzID        string    `json:"datensatz_id"`
	Zeitstempel        time.Time `json:"timestamp"`
	BearbeiterID       string    `json:"bearbeiter_id"`
	BearbeiterVorname  string    `json:"bearbeiter_vorname"`
	BearbeiterNachname string    `json:"bearbeiter_nachname"`
	// Akteur unterscheidet 'USER' von 'SYSTEM'. Ohne dieses Feld stünde eine
	// Systemaktion in der Anzeige als Eintrag ganz ohne Urheber da — nicht
	// unterscheidbar von einem Datenfehler.
	Akteur string `json:"akteur"`
}

// protokollMaxZeilen begrenzt das Logbuch auf die jüngsten Einträge.
//
// Vorher holte die Abfrage die GESAMTE Tabelle. Auf dem Prüfstand mit 247.000 Zeilen
// waren das 72 MB JSON in einer Antwort: Der Server lieferte sie in 0,6 s, aber der
// Browser musste sie parsen und ebenso viele Tabellenzeilen bauen — die Seite kam nicht
// mehr zum Vorschein, der E2E-Test lief in den Timeout. Das Logbuch wächst im Betrieb
// unbegrenzt weiter, der Fall tritt also zwangsläufig ein.
//
// 1000 wie beim Schwester-Endpunkt GetAdminAuditLogsHandler — der hatte die Grenze von
// Anfang an, dieser war der Ausreißer.
const protokollMaxZeilen = 1000

// Protokoll returns logs of immutable security events.
// @Summary      Get audit logs
// @Description  Retrieves the most recent 1000 records of the system's audit trail, newest first.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   protokollEintrag
// @Failure      500  {object}  map[string]string
// @Router       /audit [get]
func (t *Tueren) Protokoll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		zeilen, err := repository.LadeProtokoll(ctx, t.DB.Pool, protokollMaxZeilen)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := []protokollEintrag{}
		for _, z := range zeilen {
			liste = append(liste, protokollEintrag(z))
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}
