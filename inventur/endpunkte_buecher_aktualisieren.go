package inventur

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// BuchAktualisierenAnfrage repräsentiert die erwartete JSON-Struktur für das Aktualisieren eines Buches.
type BuchAktualisierenAnfrage struct {
	ISBN                    string         `json:"isbn"`
	Titel                   string         `json:"title"`
	Autor                   string         `json:"author"`
	CoverURL                string         `json:"coverUrl"`
	Fach                    string         `json:"subject"`
	KlassenStufe            int16          `json:"gradeLevel"`
	Schulzweig              string         `json:"track"`
	Bestand                 int            `json:"stock"`
	ZaehlDatum              *string        `json:"lastCounted"`
	Medientyp               string         `json:"medientyp"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften"`
}

// BearbeiteBuchAktualisieren verarbeitet PUT-Anfragen für ein bestehendes Buch.
func (handler *APIHandler) BearbeiteBuchAktualisieren(antwort http.ResponseWriter, anfrage *http.Request) {
	teile := strings.Split(strings.Trim(anfrage.URL.Path, "/"), "/")
	if len(teile) != 3 || teile[0] != "api" || teile[1] != "books" {
		writeError(antwort, http.StatusBadRequest, "ungültige route")
		return
	}

	id := teile[2]
	if id == "" {
		writeError(antwort, http.StatusBadRequest, "id darf nicht leer sein")
		return
	}

	var eingabe BuchAktualisierenAnfrage
	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		writeError(antwort, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	if validierungsFehler := bereinigeUndValidiereBuchEingabe(&eingabe); validierungsFehler != nil {
		writeError(antwort, http.StatusBadRequest, validierungsFehler.Error())
		return
	}

	ergaenzeFehlendeMetadatenFuerAktualisierung(anfrage.Context(), handler, &eingabe)

	buch := Book{
		ISBN:                    eingabe.ISBN,
		Title:                   eingabe.Titel,
		Author:                  eingabe.Autor,
		CoverURL:                eingabe.CoverURL,
		Subject:                 eingabe.Fach,
		GradeLevel:              eingabe.KlassenStufe,
		Track:                   eingabe.Schulzweig,
		Stock:                   eingabe.Bestand,
		LastCounted:             eingabe.ZaehlDatum,
		Medientyp:               eingabe.Medientyp,
		ErweiterteEigenschaften: eingabe.ErweiterteEigenschaften,
	}

	if fehler := handler.repo.UpdateBook(anfrage.Context(), id, buch); fehler != nil {
		if errors.Is(fehler, ErrDuplicateISBN) {
			writeError(antwort, http.StatusConflict, "Ein Buch mit dieser ISBN existiert bereits in der Datenbank.")
			return
		}
		if errors.Is(fehler, ErrBookNotFound) {
			writeError(antwort, http.StatusNotFound, "Buch nicht gefunden")
			return
		}
		log.Printf("Fehler beim Aktualisieren von Buch ID %s: %v", id, fehler)
		writeError(antwort, http.StatusInternalServerError, "buch konnte nicht aktualisiert werden")
		return
	}

	buch.ID = id
	writeJSON(antwort, http.StatusOK, map[string]any{"message": "buch aktualisiert", "data": buch})
}

// bereinigeUndValidiereBuchEingabe trimmt Leerzeichen der Eingabefelder und prüft auf Gültigkeit.
// Es gibt einen Fehler zurück, der als HTTP-Fehlermeldung an den Client gesendet werden kann.
func bereinigeUndValidiereBuchEingabe(eingabe *BuchAktualisierenAnfrage) error {
	eingabe.ISBN = strings.TrimSpace(eingabe.ISBN)
	eingabe.Titel = strings.TrimSpace(eingabe.Titel)
	eingabe.Autor = strings.TrimSpace(eingabe.Autor)
	eingabe.CoverURL = strings.TrimSpace(eingabe.CoverURL)
	eingabe.Fach = strings.TrimSpace(eingabe.Fach)
	eingabe.Schulzweig = strings.TrimSpace(eingabe.Schulzweig)
	eingabe.Medientyp = strings.TrimSpace(eingabe.Medientyp)

	if eingabe.ISBN == "" {
		return errors.New("isbn darf nicht leer sein")
	}
	if !validiereISBN(eingabe.ISBN) {
		return errors.New("ungültiges ISBN-Format")
	}
	if eingabe.KlassenStufe < 0 || eingabe.KlassenStufe > 13 {
		return errors.New("gradeLevel muss zwischen 0 und 13 sein")
	}
	if eingabe.Bestand < 0 {
		return errors.New("stock muss >= 0 sein")
	}

	if sig, ok := eingabe.ErweiterteEigenschaften["signatur"].(string); ok && sig != "" {
		if !validiereSignatur(sig, eingabe.Schulzweig) {
			return errors.New("ungültiges Format für Signatur/Systematik")
		}
	}

	return nil
}

// ergaenzeFehlendeMetadatenFuerAktualisierung sucht nach fehlenden Buchinformationen
// über den Metadaten-Handler und setzt Standardwerte ("Unbekannter Titel/Autor"), falls nichts gefunden wird.
func ergaenzeFehlendeMetadatenFuerAktualisierung(ctx context.Context, handler *APIHandler, eingabe *BuchAktualisierenAnfrage) {
	if eingabe.Titel == "" || eingabe.Autor == "" || eingabe.CoverURL == "" {
		nachschlagen, _ := handler.metadaten.SucheNachISBN(ctx, eingabe.ISBN)
		if nachschlagen != nil {
			if eingabe.Titel == "" {
				eingabe.Titel = strings.TrimSpace(nachschlagen.Titel)
			}
			if eingabe.Autor == "" {
				eingabe.Autor = strings.TrimSpace(nachschlagen.Autor)
			}
			if eingabe.CoverURL == "" {
				eingabe.CoverURL = strings.TrimSpace(nachschlagen.CoverURL)
			}
		}
	}

	if eingabe.Titel == "" {
		eingabe.Titel = "Unbekannter Titel"
	}
	if eingabe.Autor == "" {
		eingabe.Autor = "Unbekannter Autor"
	}
}
