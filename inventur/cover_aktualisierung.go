package inventur

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/pkg/logger"
)

// CoverAktualisieren ist die Tür POST /api/books/{id}/refresh-cover: Sie lädt das Cover eines
// Titels neu von den Katalogdiensten.
func (t *Tueren) CoverAktualisieren(w http.ResponseWriter, r *http.Request) {
	id, ok := titelIDAusPfad(w, r)
	if !ok {
		return
	}

	eintrag, err := t.abfragen.LadeTitel(r.Context(), id)
	if errors.Is(err, ErrTitelNichtGefunden) {
		schreibeFehler(w, http.StatusNotFound, "Titel nicht gefunden")
		return
	}
	if err != nil {
		log.Printf("cover-refresh: buch %s laden: %v", logger.OhneUmbruch(id), err)
		schreibeFehler(w, http.StatusInternalServerError, "buch konnte nicht geladen werden")
		return
	}

	nachschlagen, err := t.metadaten.SucheNachISBN(r.Context(), eintrag.ISBN)
	if err != nil || nachschlagen == nil {
		if errors.Is(err, ErrKatalogdiensteNichtErreichbar) {
			// Netzausfall ist kein Nicht-Treffer — sonst sieht „Cover neu laden" bei
			// WLAN-Störung so aus, als gäbe es schlicht kein besseres Cover.
			schreibeFehler(w, http.StatusBadGateway, "Die Katalogdienste (DNB, Google, OpenLibrary) sind nicht erreichbar. Bitte später erneut versuchen.")
			return
		}
		schreibeFehler(w, http.StatusNotFound, "Kein Cover: Zu dieser ISBN kennen DNB, Google Books und OpenLibrary keinen Titel.")
		return
	}

	// Nur das Cover. Schriebe die Tür auch Titel und Autor aus der Suche zurück, überschriebe der
	// Knopf „Cover neu holen" in der Titel-Akte unbemerkt Korrekturen von Hand: Wer „Mathe 7,
	// Ausgabe Hessen" eingetippt hat, bekäme den Titel aus dem Katalog zurück, ohne es zu sehen.
	// Der Knopf verspricht ein Bild, also ändert die Tür nur das Bild.
	coverURL := strings.TrimSpace(nachschlagen.CoverURL)
	if coverURL == "" {
		schreibeFehler(w, http.StatusNotFound, "Bei DNB, Google Books und OpenLibrary gibt es kein Cover zu dieser ISBN")
		return
	}

	err = t.abfragen.AendereTitelMetadaten(r.Context(), id, "", "", coverURL)
	if err != nil {
		log.Printf("cover-refresh: update fehlgeschlagen für buch %s: %v", logger.OhneUmbruch(id), err)
		schreibeFehler(w, http.StatusInternalServerError, "cover konnte nicht gespeichert werden")
		return
	}

	eintrag.CoverURL = coverURL
	schreibeJSON(w, http.StatusOK, map[string]any{"message": "Cover aktualisiert", "data": eintrag})
}
