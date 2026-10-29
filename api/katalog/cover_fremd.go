package katalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"
)

// FremdeCoverListe ist die Tür GET /api/admin/books/external-covers: die Titel, deren Cover
// noch von außen geladen wird.
func (t *Tueren) FremdeCoverListe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eintraege, err := t.titelAbfragen().LadeTitelMitFremdemCover(r.Context(), 300)
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "externe cover konnten nicht geladen werden", fmt.Errorf("bücher mit fremdem cover laden: %w", err))
			return
		}

		tuer.AntworteJSON(w, http.StatusOK, eintraege)
	}
}

// FremdeCoverNeuLaden ist die Tür POST /api/admin/books/retry-covers: Sie versucht die Cover
// von außen noch einmal zu laden.
func (t *Tueren) FremdeCoverNeuLaden() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var eingabe struct {
			IDs   []string `json:"ids"`
			Limit int      `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&eingabe); err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Anfrage ließ sich nicht lesen", nil)
			return
		}
		if !alleUUIDs(eingabe.IDs) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Eine der Kennungen ist ungültig", nil)
			return
		}

		var (
			eintraege []repository.KatalogTitel
			err       error
		)
		if len(eingabe.IDs) > 0 {
			eintraege, err = t.titelAbfragen().LadeTitelNachIDs(r.Context(), eingabe.IDs)
		} else {
			eintraege, err = t.titelAbfragen().LadeTitelMitFremdemCover(r.Context(), eingabe.Limit)
		}
		if err != nil {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "cover-retry konnte nicht gestartet werden", fmt.Errorf("fremde cover für den neuen versuch laden: %w", err))
			return
		}

		zahlen := map[string]int{"retried": 0, coverAktualisiert: 0, coverUebersprungen: 0, coverGescheitert: 0}
		for _, eintrag := range eintraege {
			zahlen["retried"]++
			zahlen[t.ladeCoverErneut(r.Context(), eintrag)]++
		}

		tuer.AntworteJSON(w, http.StatusOK, zahlen)

	}
}

// Die Zahlen der Antwort, unter denen ein Titel nach dem erneuten Laden zählt.
const (
	coverAktualisiert  = "updated"
	coverUebersprungen = "skipped"
	coverGescheitert   = "failed"
)

// ladeCoverErneut holt das Cover eines Titels über die Katalogdienste, trägt den lokalen Pfad
// des heruntergeladenen Bilds ein und nennt, unter welcher Zahl der Titel in der Antwort zählt.
//
// Nachgeladen wird nur, was noch eine Adresse im Netz trägt: Ein Cover, das lokal liegt, ist
// schon da oder von Hand hochgeladen, und das ersetzt dieser Lauf nicht. Die Suche liefert
// immer den lokalen Pfad (MetadatenClient.aufloeseCover), nie eine Adresse im Netz.
func (t *Tueren) ladeCoverErneut(ctx context.Context, eintrag repository.KatalogTitel) string {
	if !isbnutil.HatForm(eintrag.ISBN) || !strings.HasPrefix(eintrag.CoverURL, "http") {
		return coverUebersprungen
	}

	nachschlagen, nachschlageFehler := t.katalogdienste().SucheNachISBN(ctx, eintrag.ISBN)
	if nachschlageFehler != nil || nachschlagen == nil || nachschlagen.CoverURL == "" {
		return coverGescheitert
	}

	if aenderFehler := t.titelAbfragen().AendereTitelMetadaten(ctx, eintrag.ID, "", "", nachschlagen.CoverURL); aenderFehler != nil {
		return coverGescheitert
	}
	return coverAktualisiert
}
