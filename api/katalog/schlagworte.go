package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/kennung"
	"bibliothek/repository"
)

// Die Schlagworte am Titel. Die Regeln stehen in repository.SetzeSchlagworte; das Buchformular
// erreicht sie über PUT /api/books/{id}, der Bestellkorb über die Tür hier, beide über dieselbe
// Funktion.

// titelSchlagworteAnfrage ist die Eingabe für TitelSchlagworteSetzen.
type titelSchlagworteAnfrage struct {
	// Zeiger: Ein Körper ohne das Feld ist ein Fehler, keine leere Liste. Diese Tür hat
	// keinen anderen Zweck, als die Schlagworte zu setzen; ein vergessenes Feld darf
	// nicht still alle entfernen.
	Schlagworte *[]string `json:"schlagworte"`
}

// titelSchlagworteAntwort ist die Antwort beider Titel-Türen: die Schlagworte in der
// gespeicherten Schreibweise, alphabetisch.
type titelSchlagworteAntwort struct {
	ID          string   `json:"id"`
	Schlagworte []string `json:"schlagworte"`
}

// SchlagwortVorschlaege liefert die Schlagworte, die im Bestand vorkommen: ohne ?suche= die
// häufigsten zuerst (höchstens 500), mit ?suche= die Wörter, die den Text enthalten, über alle
// Wörter (höchstens 50; repository.FindeSchlagwortVorschlaege). Es ist die Vorschlagsliste der
// Eingabefelder, wie GET /api/signaturen für die Signatur; das Feld fragt beim Tippen nach
// (utils/schlagwortVorschlaege.svelte.js).
func (t *Tueren) SchlagwortVorschlaege() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		vorschlaege, err := repository.FindeSchlagwortVorschlaege(r.Context(), t.DB.Pool, r.URL.Query().Get("suche"))
		if err != nil {
			return apierrors.Internal("Schlagworte konnten nicht geladen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, vorschlaege)
		return nil
	})
}

// TitelSchlagworte liefert die Schlagworte eines Titels. Der Bestellkorb
// braucht sie, bevor er ändert: Er setzt die Menge als Ganzes, und wer die vorhandenen
// nicht kennt, würde sie mit seinem ersten Wort überschreiben.
func (t *Tueren) TitelSchlagworte() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if !kennung.IstUUID(id) {
			return apierrors.BadRequest("Die Kennung des Titels ist ungültig", errors.New("keine UUID"))
		}
		woerter, err := repository.LadeSchlagworteDesTitels(r.Context(), t.DB.Pool, id)
		if err != nil {
			return schlagwortFehler(err)
		}
		tuer.AntworteJSON(w, http.StatusOK, titelSchlagworteAntwort{ID: id, Schlagworte: woerter})
		return nil
	})
}

// TitelSchlagworteSetzen ersetzt die Schlagworte eines Titels; die schlanke Tür
// des Bestellkorbs, Schwester von TitelSignaturSetzen. Eine leere Liste entfernt
// alle; ein Wort, das es in anderer Schreibweise schon gibt, wird nicht neu angelegt.
func (t *Tueren) TitelSchlagworteSetzen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if !kennung.IstUUID(id) {
			return apierrors.BadRequest("Die Kennung des Titels ist ungültig", errors.New("keine UUID"))
		}
		var req titelSchlagworteAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		if req.Schlagworte == nil {
			return apierrors.BadRequest("Die Schlagworte fehlen. Zum Entfernen aller Schlagworte eine leere Liste schicken.",
				errors.New("feld schlagworte fehlt"))
		}
		woerter, err := repository.SetzeSchlagworte(r.Context(), t.DB.Pool, id, *req.Schlagworte)
		if err != nil {
			return schlagwortFehler(err)
		}
		tuer.AntworteJSON(w, http.StatusOK, titelSchlagworteAntwort{ID: id, Schlagworte: woerter})
		return nil
	})
}

// schlagwortFehler übersetzt die fachlichen Fehler aus repository: Eine verletzte Grenze ist
// 400 mit ihrem Grund, ein unbekannter Titel 404, alles andere 500.
func schlagwortFehler(err error) error {
	switch {
	case errors.Is(err, repository.ErrSchlagwortUngueltig):
		return apierrors.BadRequest(repository.MitGrund(meldungSchlagworteNichtGespeichert, err), err)
	case errors.Is(err, repository.ErrTitelNichtGefunden):
		return apierrors.NotFound(tuer.MeldungTitelNichtGefunden, err)
	default:
		return apierrors.Internal("Schlagworte konnten nicht gespeichert werden", err)
	}
}
