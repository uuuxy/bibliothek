package katalog

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/internal/katalogdienste"
	"bibliothek/pkg/logger"
	"bibliothek/repository"
)

// CoverAktualisieren ist die Tür POST /api/books/{id}/refresh-cover: Sie lädt das Cover eines
// Titels neu von den Katalogdiensten.
func (t *Tueren) CoverAktualisieren() http.HandlerFunc {
	return t.holeCoverNeu
}

// holeCoverNeu ist der Rumpf von CoverAktualisieren. Er steht auf der obersten Ebene, weil eine
// Closure für die Komplexitätsmessung als eigene Ebene zählt.
func (t *Tueren) holeCoverNeu(w http.ResponseWriter, r *http.Request) {
	id, ok := titelKennungAusPfad(w, r)
	if !ok {
		return
	}

	eintrag, err := t.titelAbfragen().LadeTitel(r.Context(), id)
	if errors.Is(err, repository.ErrTitelNichtGefunden) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Titel nicht gefunden", nil)
		return
	}
	if err != nil {
		slog.Error("cover-refresh: buch laden fehlgeschlagen", "titel", logger.OhneUmbruch(id), "fehler", err)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "buch konnte nicht geladen werden", nil)
		return
	}

	nachschlagen, err := t.katalogdienste().SucheNachISBN(r.Context(), eintrag.ISBN)
	if err != nil || nachschlagen == nil {
		if errors.Is(err, katalogdienste.ErrKatalogdiensteNichtErreichbar) {
			// Netzausfall ist kein Nicht-Treffer — sonst sieht „Cover neu laden" bei
			// WLAN-Störung so aus, als gäbe es schlicht kein besseres Cover.
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadGateway, "Die Katalogdienste (DNB, Google, OpenLibrary) sind nicht erreichbar. Bitte später erneut versuchen.", err)
			return
		}
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Kein Cover: Zu dieser ISBN kennen DNB, Google Books und OpenLibrary keinen Titel.", nil)
		return
	}

	// Nur das Cover. Schriebe die Tür auch Titel und Autor aus der Suche zurück, überschriebe der
	// Knopf „Cover neu holen" in der Titel-Akte unbemerkt Korrekturen von Hand: Wer „Mathe 7,
	// Ausgabe Hessen" eingetippt hat, bekäme den Titel aus dem Katalog zurück, ohne es zu sehen.
	// Der Knopf verspricht ein Bild, also ändert die Tür nur das Bild.
	coverURL := strings.TrimSpace(nachschlagen.CoverURL)
	if coverURL == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Bei DNB, Google Books und OpenLibrary gibt es kein Cover zu dieser ISBN", nil)
		return
	}

	err = t.titelAbfragen().AendereTitelMetadaten(r.Context(), id, "", "", coverURL)
	if err != nil {
		slog.Error("cover-refresh: update fehlgeschlagen", "titel", logger.OhneUmbruch(id), "fehler", err)
		apierrors.AntworteFehlerMitMeldung(w, http.StatusInternalServerError, "cover konnte nicht gespeichert werden", nil)
		return
	}

	eintrag.CoverURL = coverURL
	tuer.AntworteJSON(w, http.StatusOK, eintrag)
}
