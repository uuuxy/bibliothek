package inventur

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// BearbeiteBuecherLoeschen verarbeitet DELETE-Anfragen zum Löschen mehrerer Bücher.
// Es erwartet ein JSON-Array mit IDs und löscht diese sicher über das Repository.
func (handler *APIHandler) BearbeiteBuecherLoeschen(antwort http.ResponseWriter, anfrage *http.Request) {
	var eingabe struct {
		IDs []string `json:"ids"`
	}
	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		writeError(antwort, http.StatusBadRequest, "ungültiges request body")
		return
	}

	if len(eingabe.IDs) == 0 {
		writeError(antwort, http.StatusBadRequest, "keine IDs übergeben")
		return
	}

	if fehler := handler.repo.DeleteBooks(anfrage.Context(), eingabe.IDs); fehler != nil {
		if errors.Is(fehler, ErrBookNotFound) {
			writeError(antwort, http.StatusNotFound, "keines der ausgewählten bücher wurde gefunden")
			return
		}
		if strings.Contains(fehler.Error(), "Löschen abgebrochen") {
			writeError(antwort, http.StatusBadRequest, fehler.Error())
			return
		}
		log.Printf("Fehler beim Löschen von Büchern: %v", fehler)
		writeError(antwort, http.StatusInternalServerError, "Interner Serverfehler beim Löschen der Bücher")
		return
	}

	writeJSON(antwort, http.StatusOK, map[string]string{"message": "bücher gelöscht"})
}

// BearbeiteBuchErstellen verarbeitet POST-Anfragen zum Erstellen eines neuen Buches.
// Fehlende Metadaten (Titel, Autor, Cover) werden, falls ISBN vorhanden, automatisch
// über den MetadataClient via OpenLibrary-API im Hintergrund ergänzt, um Arbeit zu sparen.
func (handler *APIHandler) BearbeiteBuchErstellen(antwort http.ResponseWriter, anfrage *http.Request) {
	if anfrage.Method != http.MethodPost {
		writeError(antwort, http.StatusMethodNotAllowed, "nur post-anfragen erlaubt")
		return
	}

	var eingabe struct {
		ISBN                    string         `json:"isbn"`
		Fach                    string         `json:"subject"`
		KlassenStufe            int16          `json:"gradeLevel"`
		Schulzweig              string         `json:"track"`
		Bestand                 int            `json:"stock"`
		Titel                   string         `json:"title"`
		Autor                   string         `json:"author"`
		CoverURL                string         `json:"coverUrl"`
		ZaehlDatum              *string        `json:"lastCounted"`
		Medientyp               string         `json:"medientyp"`
		Signatur                string         `json:"signatur"`
		ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften"`
	}

	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		writeError(antwort, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	if eingabe.ISBN == "" {
		writeError(antwort, http.StatusBadRequest, "isbn ist erforderlich")
		return
	}
	if !validiereISBN(eingabe.ISBN) {
		writeError(antwort, http.StatusBadRequest, "ungültiges ISBN-Format")
		return
	}
	if eingabe.KlassenStufe < 0 || eingabe.KlassenStufe > 13 {
		writeError(antwort, http.StatusBadRequest, "gradeLevel muss zwischen 0 und 13 sein")
		return
	}

	buch := Book{
		ISBN:                    strings.TrimSpace(eingabe.ISBN),
		Subject:                 strings.TrimSpace(eingabe.Fach),
		GradeLevel:              eingabe.KlassenStufe,
		Track:                   strings.TrimSpace(eingabe.Schulzweig),
		Stock:                   eingabe.Bestand,
		LastCounted:             eingabe.ZaehlDatum,
		Medientyp:               strings.TrimSpace(eingabe.Medientyp),
		Signatur:                strings.TrimSpace(eingabe.Signatur),
		ErweiterteEigenschaften: eingabe.ErweiterteEigenschaften,
	}
	buch.Title = strings.TrimSpace(eingabe.Titel)
	buch.Author = strings.TrimSpace(eingabe.Autor)
	buch.CoverURL = strings.TrimSpace(eingabe.CoverURL)

	if buch.Title == "" || buch.Author == "" || buch.CoverURL == "" {
		nachschlagen, _ := handler.metadaten.SucheNachISBN(anfrage.Context(), buch.ISBN)
		if nachschlagen != nil {
			if buch.Title == "" {
				buch.Title = strings.TrimSpace(nachschlagen.Titel)
			}
			if buch.Author == "" {
				buch.Author = strings.TrimSpace(nachschlagen.Autor)
			}
			if buch.CoverURL == "" {
				buch.CoverURL = strings.TrimSpace(nachschlagen.CoverURL)
			}
		}
	}
	if buch.Title == "" {
		buch.Title = "Unbekannter Titel"
	}
	if buch.Author == "" {
		buch.Author = "Unbekannter Autor"
	}

	erstellteID, fehler := handler.repo.CreateBook(anfrage.Context(), buch)
	if fehler != nil {
		if errors.Is(fehler, ErrDuplicateISBN) {
			writeError(antwort, http.StatusConflict, "Ein Buch mit dieser ISBN existiert bereits in der Datenbank.")
			return
		}
		log.Printf("Fehler beim Erstellen von Buch ISBN %s: %v", buch.ISBN, fehler)
		writeError(antwort, http.StatusBadRequest, "buch konnte nicht erstellt werden")
		return
	}
	buch.ID = erstellteID

	writeJSON(antwort, http.StatusCreated, map[string]any{"message": "buch erstellt", "data": buch})
}
