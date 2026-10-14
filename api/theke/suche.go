package theke

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// suchErgebnis defines the combined payload for the fuzzy search.
// LeserGesamt/TitelGesamt nennen die Gesamttrefferzahl vor dem Limit, damit die
// Omnibox eine gekürzte Liste als gekürzt ausweisen kann.
type suchErgebnis struct {
	Leser       []thekenLeser      `json:"students"`
	Titel       []repository.Titel `json:"books"`
	LeserGesamt int                `json:"students_total"`
	TitelGesamt int                `json:"books_total"`
	// Treffer: exakte Auflösung eines Scans (Exemplar-Barcode, Littera-EAN, Ausweis) —
	// die globale Suchleiste springt damit direkt zur Akte (03.09.2026). nil = kein Scan.
	Treffer *service.ScanTreffer `json:"treffer,omitempty"`
}

// Suche provides a unified fuzzy search for students and books without requiring prefixes.
func (t *Tueren) Suche(leserAbfragen repository.LeserAbfragen, katalogAbfragen repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		suchtext := strings.TrimSpace(r.URL.Query().Get("q"))
		if suchtext == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("search query 'q' cannot be empty"))
			return
		}

		ctx := r.Context()

		grenze := 10

		// Using channels or just sequential is fine, but sequential is easier and perfectly fast enough
		// for a local postgres database with limits.
		leser, leserGesamt, err := leserAbfragen.FindeUnscharf(ctx, suchtext, grenze)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		titel, titelGesamt, err := katalogAbfragen.FindeTitelUnscharf(ctx, suchtext, grenze)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		treffer, err := service.ErkenneScan(ctx, katalogAbfragen, leserAbfragen, suchtext)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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
			Treffer:     treffer,
		}

		tuer.AntworteJSON(w, http.StatusOK, ergebnis)
	}
}
