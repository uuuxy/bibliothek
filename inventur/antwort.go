package inventur

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
)

// schreibeJSON delegiert an httpresp.SchreibeJSON statt direkt zu kodieren: Dort sitzt die
// zentrale nil-Slice-Normalisierung ("eine Liste ist immer [], nie null"). Ein eigener
// Encoder hier umginge sie — genau so blieb dieses Paket beim Schuelerdatei-Fix
// zunaechst ungeschuetzt.
func schreibeJSON(w http.ResponseWriter, status int, nutzlast any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	httpresp.SchreibeJSON(w, nutzlast)
}

func schreibeFehler(w http.ResponseWriter, status int, meldung string) {
	apierrors.AntworteFehler(w, status, errors.New(meldung))
}
