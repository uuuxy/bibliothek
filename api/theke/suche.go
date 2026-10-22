package theke

import (
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// suchErgebnis sind die Treffer der unscharfen Suche nach Lesern und Titeln. LeserGesamt und
// TitelGesamt nennen die Zahl der Treffer vor der Grenze, damit die Omnibox eine gekürzte
// Liste als gekürzt ausweisen kann.
type suchErgebnis struct {
	Leser       []thekenLeser      `json:"students"`
	Titel       []repository.Titel `json:"books"`
	LeserGesamt int                `json:"students_total"`
	TitelGesamt int                `json:"books_total"`
}

// Suche sucht unscharf nach Lesern und Titeln, ohne Vorsilbe (GET /api/search). Einen Scan
// bucht die Theke über POST /api/action; diese Suche bucht nie.
func (t *Tueren) Suche(leserAbfragen *repository.LeserAbfragen, katalogAbfragen *repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		suchtext := strings.TrimSpace(r.URL.Query().Get("q"))
		if suchtext == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte einen Suchbegriff eingeben", nil)
			return
		}

		ctx := r.Context()

		grenze := 10

		// Die zwei Abfragen laufen nacheinander: Mit der Grenze ist das gegen die Datenbank
		// des Servers schnell genug und einfacher als Nebenläufigkeit.
		leser, leserGesamt, err := leserAbfragen.FindeUnscharf(ctx, suchtext, grenze)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		titel, titelGesamt, err := katalogAbfragen.FindeTitelUnscharf(ctx, suchtext, grenze)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		if titel == nil {
			titel = []repository.Titel{}
		}

		// Reduktion auf die Theken-Sicht: /api/search hängt an perform_actions
		// (auch Helfer-Rolle) — Adresse, Eltern-Mail und Geburtsdatum gehören
		// nicht in diese Antwort (bewertung/sicherheitsbefund-kiosk-suche.md).
		// zuThekenLesern liefert nie nil, damit das JSON ein leeres Array trägt.
		ergebnis := suchErgebnis{
			Leser:       zuThekenLesern(leser),
			Titel:       titel,
			LeserGesamt: leserGesamt,
			TitelGesamt: titelGesamt,
		}

		tuer.AntworteJSON(w, http.StatusOK, ergebnis)
	}
}
