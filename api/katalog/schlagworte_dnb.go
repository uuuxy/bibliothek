package katalog

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/katalogdienste"
	"bibliothek/repository"
)

// Der Schlagwort-Vorschlag aus der DNB für einen Titel, den es schon gibt oder der im
// Buchformular entsteht. Ohne diese Tür bekäme ihn nur ein Titel, der beim Bestellen neu aus der
// DNB entsteht (POST /api/buecher/aus-isbn): Ein Treffer aus dem eigenen Katalog fragt die DNB
// nicht, und das Buchformular holt zur ISBN Titel und Autor, aber keine Schlagworte. Wie viele
// Titel des Bestands einen Vorschlag bekämen, steht in docs/FACHKONZEPT.md (Schlagworte).
//
// Dieselbe Regel wie beim Anlegen (repository.LadeSchlagwortVorschlagAusDNB), und wie dort ein
// Vorschlag, kein Eintrag: Geschrieben wird nichts. Welche Wörter der Titel schon trägt, weiß
// das Feld, das fragt; es bietet nur die übrigen an (ui/ChipFeld), so gilt auch ein Wort, das
// eben erst getippt und noch nicht gespeichert ist.

// dnbSchlagwortVorschlagAntwort ist die Antwort von GET /api/schlagworte/dnb-vorschlag.
type dnbSchlagwortVorschlagAntwort struct {
	// DNBSatz: false = die DNB kennt die ISBN nicht; beide Listen sind dann leer.
	DNBSatz bool `json:"dnb_satz"`
	// SchlagwortVorschlaege: die Wörter der eigenen Liste, die der Satz als Gattung, Verlagswort
	// oder Normdatei-Schlagwort nennt, aufgelöst über Verweise; alphabetisch.
	SchlagwortVorschlaege []string `json:"schlagwort_vorschlaege"`
	// SchlagwortVorschlaegeNeu: Normdatei-Schlagwörter, die die Liste noch nicht kennt. Wer eines
	// anklickt, legt es mit dem Speichern in der Liste an.
	SchlagwortVorschlaegeNeu []string `json:"schlagwort_vorschlaege_neu"`
}

// DNBSchlagwortVorschlag liefert den Schlagwort-Vorschlag der DNB zu einer ISBN
// (GET /api/schlagworte/dnb-vorschlag?isbn=).
func (t *Tueren) DNBSchlagwortVorschlag() http.HandlerFunc {
	return t.dnbSchlagwortVorschlagTuer(katalogdienste.NewMetadatenClient())
}

// dnbSchlagwortVorschlagTuer ist die Tür mit den Katalogdiensten als Parameter: Ein Test stellt
// die Antwort der DNB nach (MetadatenClient.SetzeHTTPClientFuerTest), wie bei isbnZuTitelTuer.
func (t *Tueren) dnbSchlagwortVorschlagTuer(metadaten *katalogdienste.MetadatenClient) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		antwort := dnbSchlagwortVorschlagAntwort{SchlagwortVorschlaege: []string{}, SchlagwortVorschlaegeNeu: []string{}}
		meta, err := metadaten.SucheDNBNachISBN(r.Context(), r.URL.Query().Get("isbn"))
		switch {
		case errors.Is(err, katalogdienste.ErrUngueltigeISBN):
			return apierrors.BadRequest("Keine gültige ISBN", err)
		case errors.Is(err, katalogdienste.ErrDNBNichtErreichbar):
			return apierrors.New(http.StatusBadGateway, "Die DNB ist nicht erreichbar. Bitte später erneut versuchen.", err)
		case errors.Is(err, katalogdienste.ErrNichtInDerDNB):
			tuer.AntworteJSON(w, http.StatusOK, antwort)
			return nil
		case err != nil:
			return apierrors.Internal("DNB-Abfrage fehlgeschlagen", err)
		}
		liste, neu, err := repository.LadeSchlagwortVorschlagAusDNB(r.Context(), t.DB.Pool, meta.Stichwoerter, meta.Normdaten)
		if err != nil {
			return apierrors.Internal("Schlagwort-Vorschlag konnte nicht gelesen werden", err)
		}
		antwort.DNBSatz, antwort.SchlagwortVorschlaege, antwort.SchlagwortVorschlaegeNeu = true, liste, neu
		tuer.AntworteJSON(w, http.StatusOK, antwort)
		return nil
	})
}
