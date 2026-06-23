// Package httpresp bündelt das Schreiben von HTTP-Antwortkörpern. Sind Status und
// Header erst einmal gesendet, lässt sich ein fehlgeschlagener Schreibvorgang nicht
// mehr in einen Fehlerstatus umwandeln – die einzig sinnvolle Reaktion ist Logging.
package httpresp

import (
	"encoding/json"
	"io"
	"log"
)

// Write schreibt b nach w und protokolliert einen etwaigen Schreibfehler.
func Write(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		log.Printf("httpresp: writing response body failed: %v", err)
	}
}

// Encode serialisiert payload als JSON nach w und protokolliert einen etwaigen Fehler.
func Encode(w io.Writer, payload any) {
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("httpresp: encoding JSON response failed: %v", err)
	}
}
