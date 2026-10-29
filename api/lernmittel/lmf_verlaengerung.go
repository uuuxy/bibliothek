package lernmittel

import (
	"fmt"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// verlaengerungAnfrage nennt die Klasse und das neue Rückgabedatum ihrer Lernmittel.
type verlaengerungAnfrage struct {
	Klasse              string `json:"klasse"`
	NeuesRueckgabeDatum string `json:"neues_rueckgabe_datum"` // JJJJ-MM-TT
}

// LernmittelDerKlasseVerlaengern verlängert alle Lernmittel einer Klasse auf ein Datum, in einer
// Transaktion.
func (t *Tueren) LernmittelDerKlasseVerlaengern() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verlaengerungAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		if req.Klasse == "" || req.NeuesRueckgabeDatum == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte Klasse und neues Rückgabedatum angeben", nil)
			return
		}

		frist, err := time.Parse("2006-01-02", req.NeuesRueckgabeDatum)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Das Datum muss die Form JJJJ-MM-TT haben", nil)
			return
		}
		// Tagesende in der Schulzeitzone (Berlin) — dieselbe Definition wie jede andere
		// Frist, nicht roh 23:59:59 UTC.
		frist = schulzeit.TagesEnde(frist)

		ctx := r.Context()
		tx, err := t.DB.Pool.Begin(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("transaktion der lmf-verlängerung beginnen: %w", err))
			return
		}
		defer db.Verwerfe(ctx, tx)

		angepasst, err := repository.VerlaengereLernmittelDerKlasse(ctx, tx, req.Klasse, frist)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("lernmittel der klasse verlängern: %w", err))
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("commit der lmf-verlängerung: %w", err))
			return
		}

		// Wie bei der Frist einer einzelnen Ausleihe (FRIST_OVERRIDE): Wer die Fristen einer
		// ganzen Klasse verschiebt, steht im Protokoll. Bewegt sich keine Frist, steht nichts.
		if angepasst > 0 {
			t.ProtokolliereVerwaltung(ctx, tuer.AktionFristKlasseGeaendert, map[string]any{
				"klasse":            req.Klasse,
				"neue_frist":        frist.Format(time.RFC3339),
				"fristen_angepasst": angepasst,
			})
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"success":       true,
			"updated_count": angepasst,
		})
	}
}
