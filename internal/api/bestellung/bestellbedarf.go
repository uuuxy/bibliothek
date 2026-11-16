package bestellung

import (
	"context"
	"net/http"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"
)

// bestellbedarfAntwort ist die Antwort von GET /api/bestellungen: die Zeilen und die Grundlage
// der Rechnung. Ohne die Grundlage hieße eine leere Liste „es fehlt nichts", auch wenn in diesem
// Schuljahr noch kein Schulbuch ausgegeben ist oder die Warnung aus ist.
type bestellbedarfAntwort struct {
	Zeilen []repository.Bestellbedarf `json:"zeilen"`
	// Kinder zählt die aktiven Kinder mit Jahrgang, KinderMitSchulbuch die davon, die in diesem
	// Schuljahr ein Schulbuch bekommen haben.
	Kinder             int  `json:"kinder"`
	KinderMitSchulbuch int  `json:"kinder_mit_schulbuch"`
	Ausgeschaltet      bool `json:"ausgeschaltet"`
}

// Bestellbedarf liefert die Schulbücher, deren Bestand nicht für die Kinder im nächsten
// Schuljahr reicht (repository.LadeBestellbedarf), mit der Grundlage der Rechnung.
func (t *Tueren) Bestellbedarf() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		heute := schulzeit.Jetzt()
		zeilen, ausgeschaltet, err := t.bestellbedarfMitEinstellungen(r.Context(), heute)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}
		antwort := bestellbedarfAntwort{Zeilen: zeilen, Ausgeschaltet: ausgeschaltet}
		if !ausgeschaltet {
			antwort.Kinder, antwort.KinderMitSchulbuch, err = repository.LadeBestellbedarfGrundlage(r.Context(), t.DB.Pool, heute)
			if err != nil {
				apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
				return
			}
		}
		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}

// bestellbedarfMitEinstellungen liefert den Bestellbedarf gemäß Einstellung: Ist die Warnung
// aus, keine Zeilen und ausgeschaltet. Eine Quelle für Ansicht und PDF-Export, damit beide nie
// auseinanderlaufen.
func (t *Tueren) bestellbedarfMitEinstellungen(ctx context.Context, heute time.Time) (zeilen []repository.Bestellbedarf, ausgeschaltet bool, err error) {
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return nil, false, err
	}
	if !einstellungen.BestellbedarfWarnungAktiv {
		return []repository.Bestellbedarf{}, true, nil
	}
	zeilen, err = repository.LadeBestellbedarf(ctx, t.DB.Pool, heute)
	return zeilen, false, err
}
