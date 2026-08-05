package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

// UpdateTitelSignaturRequest ist die Eingabe für UpdateTitelSignaturHandler.
type UpdateTitelSignaturRequest struct {
	Signatur string `json:"signatur"`
}

// UpdateTitelSignaturHandler ändert nur die Signatur eines Titels — schlank wie
// UpdateCopyBarcodeHandler/UpdateCopyStatusHandler, statt das große Buchformular
// (PUT /api/books/{id}) für eine einzelne Feldänderung im Bestellkorb zu bemühen.
//
// Genutzt beim Anlegen/Bestellen eines Buchs: Ein DNB-Treffer liefert nur einen
// Signatur-VORSCHLAG (siehe signaturVorschlagAusMetadaten), den das Sekretariat vor
// dem Bestellen noch korrigieren können muss. Ein bereits vorhandener Titel behält
// seine Signatur automatisch (dieser Endpunkt wird dafür nie automatisch aufgerufen,
// nur wenn im Bestellkorb tatsächlich editiert wird).
//
// @Summary      Update a title's signatur (shelf mark)
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string                      true  "Title ID"
// @Param        body  body      UpdateTitelSignaturRequest  true  "New signatur"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /buecher/titel/{id}/signatur [put]
func (s *Server) UpdateTitelSignaturHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}

		var req UpdateTitelSignaturRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		signatur := strings.TrimSpace(req.Signatur)

		var neueSignatur string
		err := s.DB.Pool.QueryRow(r.Context(), `
			UPDATE buecher_titel SET signatur = $2, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
			RETURNING coalesce(signatur, '')
		`, id, signatur).Scan(&neueSignatur)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				return apierrors.NotFound("Titel nicht gefunden", err)
			}
			return apierrors.Internal("Signatur konnte nicht gespeichert werden", err)
		}

		RespondJSON(w, http.StatusOK, map[string]string{
			"id":       id,
			"signatur": neueSignatur,
		})
		return nil
	})
}
