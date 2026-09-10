package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"

	"github.com/jackc/pgx/v5"
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

// UpdateTitelLernmittelRequest ist die Eingabe für UpdateTitelLernmittelHandler.
type UpdateTitelLernmittelRequest struct {
	IstLernmittel bool `json:"ist_lernmittel"`
}

// UpdateTitelLernmittelHandler setzt das Lernmittel-Kennzeichen eines Titels — die
// Schwester von UpdateTitelSignaturHandler, aus demselben Grund schlank.
//
// Gebraucht im Staging-Fenster der Bestellsuche: Ein über die DNB neu angelegter Titel
// (/aus-isbn) entsteht ohne Kennzeichen, und bis zum 10.09.2026 blieb er so — ein neues
// Schulbuch war damit dauerhaft ein Bücherei-Titel: falsche Frist, falscher Katalog,
// falsche Löschfrist, unsichtbar im Bestellbedarf, und seit Migration 109 der falsche
// Topf auf der Bestellung. Das Fenster fragt jetzt nach; dieser Endpunkt schreibt die
// Antwort. Für vorhandene Titel gilt das Buchformular (PUT /api/books/{id}) wie bisher.
//
// @Summary      Update a title's Lernmittel flag
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string                        true  "Title ID"
// @Param        body  body      UpdateTitelLernmittelRequest  true  "Lernmittel yes/no"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /buecher/titel/{id}/lernmittel [put]
func (s *Server) UpdateTitelLernmittelHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("id darf nicht leer sein", errors.New("missing id"))
		}

		var req UpdateTitelLernmittelRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}

		var istLernmittel bool
		err := s.DB.Pool.QueryRow(r.Context(), `
			UPDATE buecher_titel SET ist_lernmittel = $2, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
			RETURNING ist_lernmittel
		`, id, req.IstLernmittel).Scan(&istLernmittel)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Titel nicht gefunden", err)
			}
			return apierrors.Internal("Lernmittel-Kennzeichen konnte nicht gespeichert werden", err)
		}

		RespondJSON(w, http.StatusOK, map[string]any{
			"id":             id,
			"ist_lernmittel": istLernmittel,
		})
		return nil
	})
}
