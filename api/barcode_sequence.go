package api

import (
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// NextBarcodeHandler returns the next available internal B-XXXXX barcode as JSON.
//
// Speist den Knopf „Interne ID generieren" in der Exemplarkarte — der Weg, auf dem ein
// Altbestands-Exemplar mit Platzhalter (SYS-/AUTO-) seine echte Nummer bekommt.
//
// Die Nummer kommt aus derselben Sequenz wie die Vorab-Barcodes des Bestellwesens
// (repository/barcode_vergabe.go). Früher rechnete dieser Handler stattdessen
// MAX(Nummer)+1 aus dem Bestand. Das hatte zwei Folgen: Die so vergebene Nummer zog die
// nächste Bestellung erneut (UNIQUE-Verletzung, die die komplette Bestellung kippte),
// und nach einem harten Löschen — Titel löschen, Massenlöschung, „endgültig löschen" im
// Fehlbestand — kam die freigewordene Nummer zurück, obwohl ihr Etikett noch irgendwo
// klebte.
func (s *Server) NextBarcodeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nextBarcode, err := repository.NaechsterFreierExemplarBarcode(r.Context(), s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{
			"next_barcode": nextBarcode,
		})
	}
}
