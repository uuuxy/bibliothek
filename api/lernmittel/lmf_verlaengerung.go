package lernmittel

import (
	"errors"
	"log"
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
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("klasse und neues_rueckgabe_datum sind erforderlich"))
			return
		}

		frist, err := time.Parse("2006-01-02", req.NeuesRueckgabeDatum)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, errors.New("ungültiges Datumsformat (erwartet YYYY-MM-DD)"))
			return
		}
		// Tagesende in der Schulzeitzone (Berlin) — dieselbe Definition wie jede andere
		// Frist, nicht roh 23:59:59 UTC.
		frist = schulzeit.TagesEnde(frist)

		ctx := r.Context()
		tx, err := t.DB.Pool.Begin(ctx)
		if err != nil {
			log.Printf("Fehler beim Starten der Transaktion: %v", err)
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
			return
		}
		defer db.SafeRollback(ctx, tx)

		angepasst, err := repository.VerlaengereLernmittelDerKlasse(ctx, tx, req.Klasse, frist)
		if err != nil {
			log.Printf("Fehler beim globalen Verlängern: %v", err)
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("fehler beim Ausführen des Updates"))
			return
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("Fehler beim Commit der Transaktion: %v", err)
			apierrors.AntworteFehler(w, http.StatusInternalServerError, errors.New("interner Serverfehler"))
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

		tuer.AntworteJSON(w, http.StatusOK, map[string]interface{}{
			"success":       true,
			"updated_count": angepasst,
		})
	}
}
