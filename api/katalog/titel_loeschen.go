package katalog

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// TitelLoeschen löscht einen Titel mit seinen Exemplaren und schreibt den Vorgang ins
// Protokoll (KatalogAbfragen.LoescheTitel). Ist ein Exemplar verliehen, lehnt die Tür mit
// 400 ab.
func (t *Tueren) TitelLoeschen(katalogAbfragen *repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Titels fehlt", nil)
			return
		}

		ctx := r.Context()

		err := katalogAbfragen.LoescheTitel(ctx, id, claims.KontoID)
		if err != nil {
			if errors.Is(err, repository.ErrTitelNichtGefunden) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, tuer.MeldungTitelNichtGefunden, err)
				return
			}
			var mitAusleihen *repository.TitelHatAusleihenFehler
			if errors.As(err, &mitAusleihen) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf(
					"Der Titel lässt sich nicht löschen. Diese Exemplare sind noch verliehen: %s.",
					strings.Join(mitAusleihen.Barcodes, ", ")), err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteErfolg(w)
	}
}
