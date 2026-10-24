package inventur

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// ISBNSuche ist die Tür GET /api/lookup/{isbn}: Sie fragt die Katalogdienste nach den Angaben
// zu einer ISBN.
func (t *Tueren) ISBNSuche(w http.ResponseWriter, r *http.Request) {
	isbn := strings.TrimSpace(r.PathValue("isbn"))
	if isbn == "" {
		schreibeFehler(w, http.StatusBadRequest, "Die ISBN fehlt")
		return
	}

	if !validiereISBN(isbn) {
		schreibeFehler(w, http.StatusBadRequest, "Die ISBN ist ungültig")
		return
	}

	ergebnis, err := t.metadaten.SucheNachISBN(r.Context(), isbn)
	if err != nil {
		if errors.Is(err, ErrKatalogdiensteNichtErreichbar) {
			// Netzausfall ist kein Nicht-Treffer: Bei 404 katalogisiert die Theke
			// während einer WLAN-Störung Bücher von Hand, die längst in der DNB stehen.
			slog.Warn("isbn-lookup: Katalogdienste nicht erreichbar", "isbn", isbn, "fehler", err)
			schreibeFehler(w, http.StatusBadGateway, "Die Katalogdienste (DNB, Google, OpenLibrary) sind nicht erreichbar. Bitte später erneut versuchen.")
			return
		}
		slog.Info("isbn-lookup: keine Angaben gefunden", "isbn", isbn, "fehler", err)
		schreibeFehler(w, http.StatusNotFound, "Keine Angaben zu dieser ISBN gefunden")
		return
	}

	// Der Preis ist der Ladenpreis der DNB und ein Vorschlag für den Listenpreis; 0 heißt,
	// es ließ sich keiner ermitteln. Die Maske zeigt ihn, bevor gespeichert wird. Die
	// Jahrgangsspanne kommt aus dem Titel; 0 und 0 heißt, er nennt keine.
	schreibeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"title":       ergebnis.Titel,
			"subtitle":    ergebnis.Untertitel,
			"author":      ergebnis.Autor,
			"coverUrl":    ergebnis.CoverURL,
			"subject":     ergebnis.Fach,
			"jahrgangVon": ergebnis.JahrgangVon,
			"jahrgangBis": ergebnis.JahrgangBis,
			"verlag":      ergebnis.Verlag,
			"jahr":        ergebnis.Jahr,
			"zielgruppe":  ergebnis.Zielgruppe,
			"preis":       ergebnis.Preis,
		},
	})
}
