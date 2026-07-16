package inventur

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
)

// writeJSON delegiert an httpresp.Encode statt direkt zu kodieren: Dort sitzt die
// zentrale nil-Slice-Normalisierung ("eine Liste ist immer [], nie null"). Ein eigener
// Encoder hier umginge sie — genau so blieb dieses Paket beim Schuelerdatei-Fix
// zunaechst ungeschuetzt.
func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	httpresp.Encode(writer, payload)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	apierrors.SendHTTPError(writer, status, errors.New(message))
}
