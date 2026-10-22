package druck

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/buchetikett"
	"bibliothek/repository"
)

// langeNummernAntwort lehnt einen Druckauftrag ab, der eine Nummer enthält, die kein Etikett
// trägt, und nennt diese Nummern; das Druck-Center fragt dann nach neuen (VergebeNeueNummern).
type langeNummernAntwort struct {
	Error   string   `json:"error"`
	Nummern []string `json:"lange_nummern"`
}

// meldungLangeNummern steht in der Ablehnung eines Druckauftrags mit LMF-Nummer.
const meldungLangeNummern = "Eine LMF-Nummer passt nicht auf das Etikett; das Exemplar braucht dafür eine kurze Nummer"

// neueNummernAntwort nennt zu jeder alten Nummer die neue.
type neueNummernAntwort struct {
	Nummern map[string]string `json:"nummern"`
}

// VergebeNeueNummern gibt den Exemplaren mit den genannten LMF-Nummern je eine kurze Nummer
// (repository.VergebeNeueNummern). Das Druck-Center ruft sie nach einer Rückfrage, wenn
// Etikettenbogen einen Auftrag mit solchen Nummern ablehnt. Eine Nummer, die ein Etikett tragen
// darf, lehnt sie ab: Diese Tür ersetzt keine Nummer, die schon an einem Buch klebt und passt.
func (t *Tueren) VergebeNeueNummern() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			return apierrors.Unauthorized(auth.MeldungNichtAngemeldet, nil)
		}
		var req etikettenVermerkAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		for _, n := range req.BarcodeIDs {
			if buchetikett.IstDruckbar(n) {
				return apierrors.BadRequest("Eine neue Nummer bekommt nur ein Exemplar mit LMF-Nummer", nil)
			}
		}
		nummern := buchetikett.NichtDruckbare(req.BarcodeIDs)
		if len(nummern) == 0 {
			return apierrors.BadRequest("Keine Exemplare angegeben", nil)
		}

		zuordnung, err := repository.VergebeNeueNummern(r.Context(), t.DB.Pool, nummern, claims.KontoID)
		if errors.Is(err, repository.ErrExemplarNichtGefunden) {
			return apierrors.NotFound("Eine der Nummern trägt kein Exemplar mehr; die Seite neu laden zeigt den Stand", err)
		}
		if err != nil {
			return apierrors.Internal("Fehler beim Vergeben der neuen Nummern", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, neueNummernAntwort{Nummern: zuordnung})
		return nil
	})
}
