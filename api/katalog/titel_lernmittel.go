package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// titelLernmittelAnfrage ist die Eingabe für TitelLernmittelSetzen.
type titelLernmittelAnfrage struct {
	IstLernmittel bool `json:"ist_lernmittel"`
}

// TitelLernmittelSetzen setzt das Kennzeichen „Lernmittel" eines Titels; die Schwester von
// TitelSignaturSetzen, aus demselben Grund schlank.
//
// Gebraucht im Fenster der Bestellsuche, in dem ein Treffer geprüft wird, bevor er in den
// Warenkorb kommt (OrderStaging.svelte): Ein über die DNB neu angelegter Titel (/aus-isbn)
// entsteht ohne Kennzeichen. Ohne die Frage dort wäre ein neues Schulbuch ein Titel der
// Bücherei, mit falscher Frist, falschem Katalog, falscher Löschfrist, unsichtbar im
// Bestellbedarf und im falschen Topf der Bestellung. Die Tür schreibt die Antwort, auch für
// einen vorhandenen Titel, sobald jemand im Fenster das Kästchen umlegt.
//
// Fällt das Kennzeichen, fällt der Mehrjahresband mit, wie im Buchformular: Den Schalter gibt
// es nur an einem Lernmittel. Setzte die Tür allein ist_lernmittel, lehnte
// chk_mehrjahresband_spanne ab, und das Fenster meldete nur „konnte nicht gespeichert
// werden".
//
// @Summary      Kennzeichen „Lernmittel" eines Titels setzen
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string                        true  "Kennung des Titels"
// @Param        body  body      titelLernmittelAnfrage  true  "Lernmittel ja oder nein"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /buecher/titel/{id}/lernmittel [put]
func (t *Tueren) TitelLernmittelSetzen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}

		var req titelLernmittelAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}

		istLernmittel, err := repository.SetzeTitelLernmittel(r.Context(), t.DB.Pool, id, req.IstLernmittel)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Titel nicht gefunden", err)
			}
			return apierrors.Internal("Lernmittel-Kennzeichen konnte nicht gespeichert werden", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"id":             id,
			"ist_lernmittel": istLernmittel,
		})
		return nil
	})
}
