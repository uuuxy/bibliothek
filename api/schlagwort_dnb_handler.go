package api

import (
	"errors"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/inventur"
	"bibliothek/repository"
)

// Der Schlagwort-Vorschlag aus der DNB für einen Titel, den es schon gibt oder der im
// Buchformular entsteht (entschieden am 30.09.2026, docs/OFFEN.md 4.25). Beim Bestellen bekam
// ihn bis dahin nur ein Titel, der dabei neu aus der DNB entstand (POST /api/buecher/aus-isbn);
// ein Treffer aus dem eigenen Katalog fragte die DNB nicht, und das Buchformular holte zur ISBN
// Titel und Autor, aber keine Schlagworte. Gemessen am 30.09.2026 an Stichproben des
// Katalogisats vom Juni 2026: Von 50 Bücherei-Titeln ohne Schlagwort bekämen 26 einen
// Vorschlag; von 60 Titeln mit Littera-Schlagworten hätte die DNB für 26 weitere Wörter, im
// Median zwei bis drei je Titel.
//
// Dieselbe Regel wie beim Anlegen (repository.SchlagwortVorschlagAusDNB), und wie dort ein
// Vorschlag, kein Eintrag: Geschrieben wird nichts. Welche Wörter der Titel schon trägt, weiß
// das Feld, das fragt — es bietet nur die übrigen an (ui/ChipFeld), so gilt auch ein Wort, das
// eben erst getippt und noch nicht gespeichert ist.

// DnbSchlagwortVorschlag ist die Antwort von GET /api/schlagworte/dnb-vorschlag.
type DnbSchlagwortVorschlag struct {
	// DNBSatz: false = die DNB kennt die ISBN nicht; beide Listen sind dann leer.
	DNBSatz bool `json:"dnb_satz"`
	// SchlagwortVorschlaege: die Wörter der eigenen Liste, die der Satz als Gattung, Verlagswort
	// oder Normdatei-Schlagwort nennt, aufgelöst über Verweise; alphabetisch.
	SchlagwortVorschlaege []string `json:"schlagwort_vorschlaege"`
	// SchlagwortVorschlaegeNeu: Normdatei-Schlagwörter, die die Liste noch nicht kennt. Wer eines
	// anklickt, legt es mit dem Speichern in der Liste an.
	SchlagwortVorschlaegeNeu []string `json:"schlagwort_vorschlaege_neu"`
}

// DnbSchlagwortVorschlagHandler handles GET /api/schlagworte/dnb-vorschlag?isbn=.
//
// @Summary      Keyword suggestions from the DNB for an ISBN
// @Description  Asks only the German National Library (no Google Books, no OpenLibrary, no cover download) and matches its genre terms, publisher keywords and GND subject headings against the own keyword list — the same rule as POST /buecher/aus-isbn. Nothing is written.
// @Tags         books
// @Produce      json
// @Param        isbn  query     string  true  "ISBN-10 or ISBN-13"
// @Success      200   {object}  DnbSchlagwortVorschlag
// @Failure      400   {object}  map[string]string
// @Failure      502   {object}  map[string]string
// @Router       /schlagworte/dnb-vorschlag [get]
func (s *Server) DnbSchlagwortVorschlagHandler() http.HandlerFunc {
	return s.dnbSchlagwortVorschlag(inventur.NeuerMetadatenClient())
}

// dnbSchlagwortVorschlag ist die Tür mit ihrem DNB-Client als Parameter: Ein Test stellt die
// Antwort der DNB nach (MetadatenClient.SetzeHTTPClientFuerTest), wie bei isbnZuTitel.
func (s *Server) dnbSchlagwortVorschlag(client *inventur.MetadatenClient) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		antwort := DnbSchlagwortVorschlag{SchlagwortVorschlaege: []string{}, SchlagwortVorschlaegeNeu: []string{}}
		meta, err := client.SucheDNBNachISBN(r.Context(), r.URL.Query().Get("isbn"))
		switch {
		case errors.Is(err, inventur.ErrUngueltigeISBN):
			return apierrors.BadRequest("keine gültige ISBN", err)
		case errors.Is(err, inventur.ErrDNBNichtErreichbar):
			return apierrors.New(http.StatusBadGateway, "DNB nicht erreichbar — bitte später erneut versuchen", err)
		case errors.Is(err, inventur.ErrNichtInDerDNB):
			RespondJSON(w, http.StatusOK, antwort)
			return nil
		case err != nil:
			return apierrors.Internal("DNB-Abfrage fehlgeschlagen", err)
		}
		liste, neu, err := repository.SchlagwortVorschlagAusDNB(r.Context(), s.DB.Pool, meta.Stichwoerter, meta.Normdaten)
		if err != nil {
			return apierrors.Internal("Schlagwort-Vorschlag konnte nicht gelesen werden", err)
		}
		antwort.DNBSatz, antwort.SchlagwortVorschlaege, antwort.SchlagwortVorschlaegeNeu = true, liste, neu
		RespondJSON(w, http.StatusOK, antwort)
		return nil
	})
}
