package schaden

import (
	"context"
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// schadensMeldung bucht einen Verlust oder Schaden an einer Ausleihe und liefert die ID des
// Schadensfalls (repository.SchadensAbfragen). Hat sich die Lage geändert, seit die Akte offen
// steht, liefert es repository.ErrExemplarNeuVerliehen oder
// repository.ErrAusleiheInzwischenZurueck.
type schadensMeldung interface {
	Melde(ctx context.Context, exemplarID, ausleiheID, bearbeiterID, beschreibung string, art repository.SchadensArt, betrag float64) (string, error)
}

// SchadenMelden meldet einen Verlust oder Schaden (POST /api/damage/report): Das Exemplar wird
// ausgesondert, die Forderung angelegt und die Ausleihe beendet.
func (t *Tueren) SchadenMelden(schadensAbfragen schadensMeldung) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized(auth.MeldungNichtAngemeldet, nil)
		}

		var req struct {
			AusleiheID string `json:"loan_id" validate:"omitempty,uuid_oder_leer"`
			// SchuelerID schickt der Dialog mit; gelesen wird es nicht. Der Schuldner steht
			// an der Ausleihe (repository.meldeSchaden).
			SchuelerID   string `json:"schueler_id" validate:"omitempty,uuid_oder_leer"`
			ExemplarID   string `json:"copy_id" validate:"omitempty,uuid_oder_leer"`
			Beschreibung string `json:"beschreibung"`
			// Art: Fallgruppe des Bescheids, Pflicht und ohne stillen Vorgabewert: Mit der
			// Vorgabe 'beschaedigt' der Spalte nennte der Bescheid ein verlorenes Buch „beschädigt
			// zurückgegeben".
			Art    string  `json:"art"`
			Betrag float64 `json:"betrag"`
		}
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		art := repository.SchadensArt(req.Art)
		if !art.IstGueltig() {
			return apierrors.BadRequest("Bitte angeben, ob das Buch nicht zurückgegeben oder beschädigt zurückgegeben wurde",
				errors.New("art fehlt oder ist ungültig"))
		}

		schadensID, err := schadensAbfragen.Melde(r.Context(), req.ExemplarID, req.AusleiheID, claims.KontoID, req.Beschreibung, art, req.Betrag)
		if err != nil {
			// Inzwischen neu verliehen oder zurückgegeben: ein Konflikt (409), kein
			// Serverfehler. Die Akte zeigt einen Stand, den es nicht mehr gibt.
			if meldung, ok := lageGeaendert(err); ok {
				return apierrors.Conflict(meldung, err)
			}
			return apierrors.Internal("Fehler beim Melden des Schadens", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"status": "ok", "schadens_id": schadensID})
		return nil
	})
}
