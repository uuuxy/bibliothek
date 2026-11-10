package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// ExemplarAussondern ist „Exemplar löschen" in der Buchakte: Es sondert ein Exemplar mit dem Grund
// aus dem Körper ({"grund": "AUSSORTIERT"}) aus. Die Zeile bleibt und steht im Abgangsbuch, weil
// Ausleihen und Schadensfälle an ihr hängen (KatalogAbfragen.SondereExemplarAus). Ein verliehenes
// Exemplar lehnt die Tür mit 400 ab.
func (t *Tueren) ExemplarAussondern(katalogAbfragen *repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Exemplars fehlt", nil)
			return
		}

		grund, meldung := leseGrund(w, r)
		if meldung != "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldung, nil)
			return
		}

		ctx := r.Context()

		err := katalogAbfragen.SondereExemplarAus(ctx, id, grund, claims.KontoID)
		if err != nil {
			// Am Fehlerwert erkannt, nicht am Text der Meldung: Eine umformulierte Meldung machte
			// aus der Auskunft sonst einen Serverfehler.
			if errors.Is(err, repository.ErrExemplarNochVerliehen) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungExemplarVerliehen, err)
				return
			}
			if errors.Is(err, repository.ErrExemplarNichtGefunden) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungExemplarNichtGefunden, err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		tuer.AntworteErfolg(w)
	}
}

// leseGrund liest den Grund einer Aussonderung aus dem Körper {"grund": "AUSSORTIERT"} und liefert
// die Meldung, wenn der Körper nicht lesbar oder der Grund keiner der vier ist.
func leseGrund(w http.ResponseWriter, r *http.Request) (grund, meldung string) {
	var eingabe struct {
		Grund string `json:"grund"`
	}
	if err := tuer.DekodiereRumpf(w, r, &eingabe); err != nil {
		return "", "Die Anfrage ließ sich nicht lesen"
	}
	if !repository.IstAussonderungsgrund(eingabe.Grund) {
		return "", meldungGrundFehlt
	}
	return eingabe.Grund, ""
}
