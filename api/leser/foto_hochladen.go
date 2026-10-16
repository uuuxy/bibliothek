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
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Lesers fehlt", nil)
			return
		}

		var req fotoHochladenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if !strings.HasPrefix(req.FotoDaten, "data:image/jpeg;base64,") {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Das Foto muss ein JPEG-Bild sein", nil)
			return
		}

		ctx := r.Context()

		fotoURL, err := service.SchreibeLeserFoto(ctx, t.DB.Pool, id, req.FotoDaten)
		if err != nil {
			if errors.Is(err, service.ErrFotoLeserUnbekannt) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungLeserNichtGefunden, err)
				return
			}
			if errors.Is(err, service.ErrFotoUnlesbar) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest,
					"Das Bild ließ sich nicht lesen. Bitte die Aufnahme wiederholen.", err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Über den JSON-Schreiber, nicht von Hand zusammengesetzt: Die Ausweisnummer steht in
		// der Adresse und darf Zeichen tragen, die in JSON maskiert werden müssen.
		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "success", "url": fotoURL})
	}
}
