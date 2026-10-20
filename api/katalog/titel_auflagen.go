package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Die Auflagen eines Schulbuchs. Die Regeln stehen in repository/auflagen.go; die Titelmaske
// erreicht sie über die Türen hier (edit_books), der Bestellbedarf über eine eigene Route mit
// create_orders (…/neue-auflage). Beide gehen über dieselben Funktionen, wie bei den
// Schlagworten.

// titelAuflagenAntwort ist die Antwort aller drei Türen: die Auflagen des Buchs, zu dem der
// Titel gehört, die neueste zuerst. Ein Titel ohne andere Auflage steht allein darin.
type titelAuflagenAntwort struct {
	ID       string               `json:"id"`
	Auflagen []repository.Auflage `json:"auflagen"`
}

// auflageZuordnenAnfrage nennt den Titel, der eine andere Auflage desselben Buchs ist.
type auflageZuordnenAnfrage struct {
	TitelID string `json:"titel_id" validate:"required,uuid_oder_leer"`
}

// TitelAuflagen liefert die Auflagen des Buchs, zu dem ein Titel gehört, mit ihrem Bestand.
func (t *Tueren) TitelAuflagen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id, err := titelIDAusPfad(r)
		if err != nil {
			return err
		}
		auflagen, err := repository.LadeAuflagenDesTitels(r.Context(), t.DB.Pool, id)
		if err != nil {
			return auflagenFehler(err, "gelesen")
		}
		tuer.AntworteJSON(w, http.StatusOK, titelAuflagenAntwort{ID: id, Auflagen: auflagen})
		return nil
	})
}

// AuflagenZusammenfassen fasst einen Titel mit einer anderen Auflage desselben Buchs
// zusammen. Gehört einer der beiden schon zu einem Buch mit weiteren Auflagen, kommen alle
// zusammen. Zwei Routen, eine Tür: die Titelmaske (…/auflagen, edit_books) und der
// Bestellbedarf (…/neue-auflage, create_orders), denn wer bestellt, legt das Werk an.
func (t *Tueren) AuflagenZusammenfassen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id, err := titelIDAusPfad(r)
		if err != nil {
			return err
		}
		var req auflageZuordnenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		auflagen, err := repository.FasseAuflagenZusammen(r.Context(), t.DB.Pool, id, req.TitelID)
		if err != nil {
			return auflagenFehler(err, "gespeichert")
		}
		tuer.AntworteJSON(w, http.StatusOK, titelAuflagenAntwort{ID: id, Auflagen: auflagen})
		return nil
	})
}

// AuflageLoesen nimmt einen Titel aus seinem Buch; die anderen Auflagen bleiben zusammen,
// solange es mindestens zwei sind.
func (t *Tueren) AuflageLoesen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id, err := titelIDAusPfad(r)
		if err != nil {
			return err
		}
		auflagen, err := repository.LoeseAuflage(r.Context(), t.DB.Pool, id)
		if err != nil {
			return auflagenFehler(err, "gespeichert")
		}
		tuer.AntworteJSON(w, http.StatusOK, titelAuflagenAntwort{ID: id, Auflagen: auflagen})
		return nil
	})
}

// titelIDAusPfad liest die Titel-ID aus dem Pfad; keine UUID ist 400.
func titelIDAusPfad(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !kennung.IstUUID(id) {
		return "", apierrors.BadRequest("Die Kennung des Titels ist ungültig", errors.New("keine UUID"))
	}
	return id, nil
}

// auflagenFehler übersetzt die fachlichen Fehler aus repository/auflagen.go: Eine verletzte
// Regel ist 400 mit ihrem Grund, ein unbekannter Titel 404, alles andere 500.
func auflagenFehler(err error, handlung string) error {
	switch {
	case errors.Is(err, repository.ErrAuflageUngueltig):
		return apierrors.BadRequest(repository.MitGrund(meldungAuflagenNichtGeaendert, err), err)
	case errors.Is(err, repository.ErrTitelNichtGefunden):
		return apierrors.NotFound(tuer.MeldungTitelNichtGefunden, err)
	default:
		return apierrors.Internal("Auflagen konnten nicht "+handlung+" werden", err)
	}
}
