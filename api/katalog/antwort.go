package katalog

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
)

// schreibeJSON gibt an httpresp.SchreibeJSON weiter, statt selbst zu kodieren: Dort sitzt die
// eine Regel für leere Listen („eine Liste ist immer [], nie null"). Ein eigener Encoder hier
// umginge sie.
func schreibeJSON(w http.ResponseWriter, status int, nutzlast any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	httpresp.SchreibeJSON(w, nutzlast)
}

// schreibeFehler schickt eine Meldung für die Oberfläche; bei 500 bleibt sie neutral, wie überall
// (apierrors.AntworteFehlerMitMeldung).
func schreibeFehler(w http.ResponseWriter, status int, meldung string) {
	apierrors.AntworteFehlerMitMeldung(w, status, meldung, nil)
}
