package leser

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/service"
)

// fotoHochladenAnfrage ist das Foto als Base64 in einer data-URL.
type fotoHochladenAnfrage struct {
	FotoDaten string `json:"photo_data"` // data:image/jpeg;base64,...
}

// FotoHochladen nimmt das Ausweisfoto aus der Kamera an und speichert es verschlüsselt am
// Leser (service.SchreibeLeserFoto).
func (t *Tueren) FotoHochladen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing student ID parameter"))
			return
		}

		var req fotoHochladenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if !strings.HasPrefix(req.FotoDaten, "data:image/jpeg;base64,") {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("payload must be a base64 encoded JPEG data URL"))
			return
		}

		ctx := r.Context()

		fotoURL, err := service.SchreibeLeserFoto(ctx, t.DB.Pool, id, req.FotoDaten)
		if err != nil {
			if errors.Is(err, service.ErrFotoLeserUnbekannt) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			if errors.Is(err, service.ErrFotoUnlesbar) {
				apierrors.SendHTTPErrorMitMeldung(w, http.StatusBadRequest,
					"Das Bild ließ sich nicht lesen. Bitte die Aufnahme wiederholen.", err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Über den JSON-Schreiber, nicht von Hand zusammengesetzt: Die Ausweisnummer steht in
		// der Adresse und darf Zeichen tragen, die in JSON maskiert werden müssen.
		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "success", "url": fotoURL})
	}
}
