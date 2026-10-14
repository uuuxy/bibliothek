package mahnwesen

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// klassenleitung ordnet einer Klasse die Adresse ihrer Klassenleitung zu.
type klassenleitung struct {
	Klasse      string `json:"klasse"`
	LehrerEmail string `json:"lehrer_email"`
	ErstelltAm  string `json:"erstellt_am,omitempty"`
}

// Klassenleitungen liefert alle Klassen mit der Adresse ihrer Klassenleitung.
// GET /api/klassen-mapping
func (t *Tueren) Klassenleitungen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		zeilen, err := repository.LadeKlassenleitungen(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := []klassenleitung{}
		for _, z := range zeilen {
			liste = append(liste, klassenleitung{
				Klasse: z.Klasse, LehrerEmail: z.LehrerEmail, ErstelltAm: z.ErstelltAm.Format("2006-01-02"),
			})
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// KlassenleitungSetzen trägt die Adresse der Klassenleitung einer Klasse ein oder ändert sie.
// POST /api/klassen-mapping  { "klasse": "5b", "lehrer_email": "..." }
func (t *Tueren) KlassenleitungSetzen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req klassenleitung
		if !tuer.LeseAnfrage(w, r, &req) {
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
		neu, unveraendert, err := repository.SetzeKlassenleitung(ctx, t.DB.Pool, req.Klasse, req.LehrerEmail)
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
			t.ProtokolliereVerwaltung(ctx, tuer.AktionKlassenleitungGeaendert,
				map[string]any{"klasse": req.Klasse, "art": art})
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// KlassenleitungLoeschen nimmt die Adresse der Klassenleitung einer Klasse heraus.
// DELETE /api/klassen-mapping/{klasse}
func (t *Tueren) KlassenleitungLoeschen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		klasse := r.PathValue("klasse")
		if klasse == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("klasse erforderlich"))
			return
		}

		ctx := r.Context()

		getroffen, err := repository.LoescheKlassenleitung(ctx, t.DB.Pool, klasse)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		// 204 für etwas, das nie existierte, wäre ein Erfolg, den es nicht gab.
		if getroffen == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("kein Eintrag für diese Klasse"))
			return
		}
		t.ProtokolliereVerwaltung(ctx, tuer.AktionKlassenleitungEntfernt, map[string]any{"klasse": klasse})
		w.WriteHeader(http.StatusNoContent)
	}
}
