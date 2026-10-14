package system

// Das Protokoll der Vorgänge (audit_log) zum Lesen, die jüngsten Einträge zuerst.

import (
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// protokollEintrag ist eine Zeile des Protokolls der Vorgänge mit dem Namen des Bearbeiters.
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

// protokollMaxZeilen begrenzt das Logbuch auf die jüngsten Einträge: Auf dem Prüfstand mit
// 247.000 Zeilen wäre die ganze Tabelle eine Antwort von 72 MB JSON, die der Browser nicht mehr
// darstellt, und das Logbuch wächst im Betrieb unbegrenzt. 1000 wie bei VerwaltungsProtokoll.
const protokollMaxZeilen = 1000

// Protokoll liefert die jüngsten 1000 Einträge des Protokolls der Vorgänge.
// @Summary      Protokoll
// @Description  Die jüngsten 1000 Einträge des Protokolls der Vorgänge, die neuesten zuerst.
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
