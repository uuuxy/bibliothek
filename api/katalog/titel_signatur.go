package katalog

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// titelSignaturAnfrage ist die Eingabe für TitelSignaturSetzen.
type titelSignaturAnfrage struct {
	Signatur string `json:"signatur"`
}

// TitelSignaturSetzen ändert nur die Signatur eines Titels. Der Bestellkorb braucht dafür
// nicht das ganze Buchformular (PUT /api/books/{id}), wie bei ExemplarBarcodeSetzen und
// ExemplarStatusAendern.
//
// Ein Titel, der beim Bestellen neu aus der DNB entsteht, hat keine Signatur; sie wird im
// Bestellkorb eingetragen, mit der Vorschlagsliste aus dem Bestand (Signaturen). Ein
// vorhandener Titel behält seine Signatur: Der Bestellkorb ruft die Tür nur, wenn dort jemand
// die Signatur ändert.
func (t *Tueren) TitelSignaturSetzen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}

		var req titelSignaturAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		signatur := strings.TrimSpace(req.Signatur)

		neueSignatur, err := repository.SetzeTitelSignatur(r.Context(), t.DB.Pool, id, signatur)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Titel nicht gefunden", err)
			}
			return apierrors.Internal("Signatur konnte nicht gespeichert werden", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{
			"id":       id,
			"signatur": neueSignatur,
		})
		return nil
	})
}
