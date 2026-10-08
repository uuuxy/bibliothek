package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// KlassenLehrerMapping associates a class with the class teacher's e-mail address.
type KlassenLehrerMapping struct {
	Klasse      string `json:"klasse"`
	LehrerEmail string `json:"lehrer_email"`
	ErstelltAm  string `json:"erstellt_am,omitempty"`
}

// GetKlassenMappingHandler returns all class → teacher-e-mail mappings.
// GET /api/klassen-mapping
func (s *Server) GetKlassenMappingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		zeilen, err := repository.ListeKlassenleitungen(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		mappings := []KlassenLehrerMapping{}
		for _, z := range zeilen {
			mappings = append(mappings, KlassenLehrerMapping{
				Klasse: z.Klasse, LehrerEmail: z.LehrerEmail, ErstelltAm: z.ErstelltAm.Format("2006-01-02"),
			})
		}

		RespondJSON(w, http.StatusOK, mappings)
	}
}

// UpsertKlassenMappingHandler creates or updates a class → teacher-e-mail mapping.
// POST /api/klassen-mapping  { "klasse": "5b", "lehrer_email": "..." }
func (s *Server) UpsertKlassenMappingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req KlassenLehrerMapping
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		// Getrimmt speichern: Ein aus der Zwischenablage mitgeschlepptes Leerzeichen ist
		// unsichtbar, macht das Kürzel aber zu einem anderen — die Klasse bliebe ohne
		// Adresse, ohne dass jemand sieht warum.
		req.Klasse = strings.TrimSpace(req.Klasse)
		req.LehrerEmail = strings.TrimSpace(req.LehrerEmail)

		if req.Klasse == "" || req.LehrerEmail == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse und lehrer_email sind erforderlich"))
			return
		}

		ctx := r.Context()

		// Der Stand davor entscheidet über den Protokolleintrag: neu eingetragen, geändert
		// oder dieselbe Adresse noch einmal gespeichert.
		neu, unveraendert, err := repository.SetzeKlassenleitung(ctx, s.DB.Pool, req.Klasse, req.LehrerEmail)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		// An diese Adresse gehen die Mahnlisten der Klasse mit Namen und Titeln. Der Eintrag
		// nennt die Klasse, nicht die Adresse.
		if !unveraendert {
			art := "geaendert"
			if neu {
				art = "eingetragen"
			}
			s.protokolliereVerwaltung(ctx, auditKlassenleitungGeaendert,
				map[string]any{"klasse": req.Klasse, "art": art})
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// DeleteKlassenMappingHandler removes a class → teacher-e-mail mapping.
// DELETE /api/klassen-mapping/{klasse}
func (s *Server) DeleteKlassenMappingHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		klasse := r.PathValue("klasse")
		if klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse erforderlich"))
			return
		}

		ctx := r.Context()

		getroffen, err := repository.LoescheKlassenleitung(ctx, s.DB.Pool, klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		// 204 für etwas, das nie existierte, wäre ein Phantom-Erfolg (Sweep 31.08.2026).
		if getroffen == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("kein Eintrag für diese Klasse"))
			return
		}
		s.protokolliereVerwaltung(ctx, auditKlassenleitungEntfernt, map[string]any{"klasse": klasse})
		w.WriteHeader(http.StatusNoContent)
	}
}
