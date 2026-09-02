package inventur

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

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

	var eingabe BuchEingabe
	if fehler := json.NewDecoder(anfrage.Body).Decode(&eingabe); fehler != nil {
		writeError(antwort, http.StatusBadRequest, "ungültiges JSON")
		return
	}

	if validierungsFehler := bereinigeUndValidiereBuchEingabe(&eingabe); validierungsFehler != nil {
		writeError(antwort, http.StatusBadRequest, validierungsFehler.Error())
		return
	}

	ergaenzeFehlendeMetadatenFuerAktualisierung(anfrage.Context(), handler, &eingabe)

	// Beim ÄNDERN ist ein leerer Titel ein Fehler, kein Anlass für einen Platzhalter.
	//
	// Bis zum 23.08.2026 wurde daraus still "Unbekannter Titel" bzw. "Unbekannter Autor" —
	// dieselbe Regel wie beim Anlegen. Dort ist sie richtig: Ein per Scan angelegtes Buch
	// ohne Fund im Katalog braucht irgendeinen Namen. Beim Ändern steht davor aber ein
	// Mensch, der ein Feld geleert hat oder dessen Formular es nie befüllt hat — und der
	// bekam die Bescheinigung "buch aktualisiert", während der Titel seines Buchs
	// verschwand. Der Nachschlag oben darf weiter füllen; nur der Platzhalter ist weg.
	if eingabe.Titel == "" {
		writeError(antwort, http.StatusBadRequest,
			"titel darf nicht leer sein (beim Ändern wird kein Platzhalter eingesetzt)")
		return
	}
	if eingabe.Autor == "" {
		writeError(antwort, http.StatusBadRequest,
			"autor darf nicht leer sein (beim Ändern wird kein Platzhalter eingesetzt)")
		return
	}

	buch := Book{
		ISBN:                    eingabe.ISBN,
		Title:                   eingabe.Titel,
		Author:                  eingabe.Autor,
		CoverURL:                eingabe.CoverURL,
		Subject:                 eingabe.Fach,
		GradeLevel:              eingabe.KlassenStufe,
		Track:                   eingabe.Schulzweig,
		IstLernmittel:           eingabe.IstLernmittel,
		LastCounted:             eingabe.ZaehlDatum,
		Medientyp:               eingabe.Medientyp,
		JahrgangVon:             eingabe.JahrgangVon,
		JahrgangBis:             eingabe.JahrgangBis,
		Untertitel:              eingabe.Untertitel,
		Verlag:                  eingabe.Verlag,
		Erscheinungsjahr:        eingabe.Erscheinungsjahr,
		Beschreibung:            eingabe.Beschreibung,
		Signatur:                strings.TrimSpace(eingabe.Signatur),
		ErweiterteEigenschaften: eingabe.ErweiterteEigenschaften,
	}

	if fehler := handler.repo.UpdateBook(anfrage.Context(), id, buch, eingabe.Bestand); fehler != nil {
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
func bereinigeUndValidiereBuchEingabe(eingabe *BuchEingabe) error {
	eingabe.ISBN = strings.TrimSpace(eingabe.ISBN)
	eingabe.Titel = strings.TrimSpace(eingabe.Titel)
	eingabe.Autor = strings.TrimSpace(eingabe.Autor)
	eingabe.CoverURL = strings.TrimSpace(eingabe.CoverURL)
	eingabe.Fach = strings.TrimSpace(eingabe.Fach)
	eingabe.Schulzweig = strings.TrimSpace(eingabe.Schulzweig)
	eingabe.Medientyp = strings.TrimSpace(eingabe.Medientyp)
	eingabe.Untertitel = strings.TrimSpace(eingabe.Untertitel)
	eingabe.Verlag = strings.TrimSpace(eingabe.Verlag)
	eingabe.Beschreibung = strings.TrimSpace(eingabe.Beschreibung)

	if eingabe.ISBN == "" {
		return errors.New("isbn darf nicht leer sein")
	}
	if !validiereISBN(eingabe.ISBN) {
		return errors.New("ungültiges ISBN-Format")
	}
	if eingabe.KlassenStufe < 0 || eingabe.KlassenStufe > 13 {
		return errors.New("gradeLevel muss zwischen 0 und 13 sein")
	}
	if eingabe.Bestand != nil && *eingabe.Bestand < 0 {
		return errors.New("stock muss >= 0 sein")
	}

	return nil
}

// ergaenzeFehlendeMetadatenFuerAktualisierung sucht fehlende Buchinformationen über den
// Metadaten-Handler nach. Standardwerte setzt sie NICHT mehr: Bleibt der Titel danach
// leer, ist das beim Ändern ein Fehler (400), kein Platzhalter — siehe den Aufrufer.
func ergaenzeFehlendeMetadatenFuerAktualisierung(ctx context.Context, handler *APIHandler, eingabe *BuchEingabe) {
	if eingabe.Titel == "" || eingabe.Autor == "" || eingabe.CoverURL == "" {
		nachschlagen, _ := handler.metadaten.SucheNachISBN(ctx, eingabe.ISBN) //nolint:errcheck
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

}
