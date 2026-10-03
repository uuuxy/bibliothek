package inventur

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// BearbeiteBuchAktualisieren verarbeitet PUT-Anfragen für ein bestehendes Buch.
func (handler *APIHandler) BearbeiteBuchAktualisieren(antwort http.ResponseWriter, anfrage *http.Request) {
	id, ok := buchIDAusPfad(antwort, anfrage)
	if !ok {
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
	schlagworte, fehler := schlagworteAusEingabe(eingabe.Schlagworte)
	if fehler != nil {
		writeError(antwort, http.StatusBadRequest, fehler.Error())
		return
	}

	// Beim Ändern ist ein leerer Titel ein Fehler, kein Anlass für einen Platzhalter: Davor
	// steht ein Mensch, der das Feld geleert hat, oder ein Formular, das es nie befüllt hat.
	// Die Katalogdienste füllen hier nichts nach — sonst wartete das Speichern auf sie, und
	// in der Akte stünde, was niemand eingetragen und niemand gesehen hat.
	if eingabe.Titel == "" {
		writeError(antwort, http.StatusBadRequest,
			"titel darf nicht leer sein (beim Ändern wird kein Platzhalter eingesetzt)")
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
		Mehrjahresband:          eingabe.Mehrjahresband,
		Untertitel:              eingabe.Untertitel,
		Auflage:                 strings.TrimSpace(eingabe.Auflage),
		Listenpreis:             eingabe.Listenpreis,
		Verlag:                  eingabe.Verlag,
		Erscheinungsjahr:        eingabe.Erscheinungsjahr,
		Signatur:                strings.TrimSpace(eingabe.Signatur),
		ErweiterteEigenschaften: eingabe.ErweiterteEigenschaften,
		Schlagworte:             schlagworte,
	}

	if fehler := handler.repo.UpdateBook(anfrage.Context(), id, buch, bestandsangabe(eingabe)); fehler != nil {
		if errors.Is(fehler, ErrDuplicateISBN) {
			schreibeDubletteISBN(antwort, fehler)
			return
		}
		var veraltet *BestandVeraltet
		if errors.As(fehler, &veraltet) {
			// Mit dem Stand in der Antwort stellt die Maske ihr Feld nach, ohne dass die
			// übrigen Eingaben verloren gehen.
			writeJSON(antwort, http.StatusConflict, map[string]any{
				"error":   veraltet.Meldung(),
				"bestand": veraltet.Aktuell,
			})
			return
		}
		if errors.Is(fehler, ErrBookNotFound) {
			writeError(antwort, http.StatusNotFound, "Buch nicht gefunden")
			return
		}
		if errors.Is(fehler, ErrAutorGeleert) {
			writeError(antwort, http.StatusBadRequest,
				"autor darf nicht leer sein (beim Ändern wird kein Platzhalter eingesetzt)")
			return
		}
		if errors.Is(fehler, ErrISBNFormat) {
			writeError(antwort, http.StatusBadRequest, "ungültiges ISBN-Format")
			return
		}
		log.Printf("Fehler beim Aktualisieren von Buch ID %s: %v", id, fehler)
		writeError(antwort, http.StatusInternalServerError, "buch konnte nicht aktualisiert werden")
		return
	}

	buch.ID = id
	writeJSON(antwort, http.StatusOK, map[string]any{"message": "buch aktualisiert", "data": handler.gespeichert(anfrage.Context(), buch)})
}

// bestandsangabe liest aus der Eingabe, was sie zum Bestand sagt. Ohne das Feld „stock" sagt
// sie nichts (nil), und die Exemplare bleiben unangetastet.
func bestandsangabe(eingabe BuchEingabe) *Bestandsangabe {
	if eingabe.Bestand == nil {
		return nil
	}
	return &Bestandsangabe{Soll: *eingabe.Bestand, Gesehen: eingabe.BestandGesehen}
}

// bereinigeUndValidiereBuchEingabe trimmt Leerzeichen der Eingabefelder und prüft auf Gültigkeit.
// Es gibt einen Fehler zurück, der als HTTP-Fehlermeldung an den Client gesendet werden kann.
// Die ISBN prüft sie nicht: Ob sie sich geändert hat, weiß erst der Schreibpfad, der den
// gespeicherten Titel liest (pruefeAenderung).
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

	if eingabe.KlassenStufe < 0 || eingabe.KlassenStufe > 13 {
		return errors.New("gradeLevel muss zwischen 0 und 13 sein")
	}
	if eingabe.Bestand != nil && *eingabe.Bestand < 0 {
		return errors.New("stock muss >= 0 sein")
	}
	if eingabe.BestandGesehen != nil && *eingabe.BestandGesehen < 0 {
		return errors.New("stockGesehen muss >= 0 sein")
	}
	if fehler := pruefeListenpreis(eingabe.Listenpreis); fehler != nil {
		return fehler
	}
	return pruefeMehrjahresband(eingabe.IstLernmittel, eingabe.Mehrjahresband, eingabe.JahrgangVon, eingabe.JahrgangBis)
}

// pruefeListenpreis nennt den erlaubten Bereich, bevor die Datenbank ablehnt
// (chk_listenpreis_nonneg, Migration 127): Beim Ändern kam dort eine 500, beim Anlegen
// „buch konnte nicht erstellt werden" — beides ohne zu sagen, was erlaubt ist. Leer (nil)
// heißt „nicht erfasst" und ist etwas anderes als 0.
func pruefeListenpreis(preis *float64) error {
	if preis != nil && *preis < 0 {
		return errors.New("listenpreis muss >= 0 sein (leer lassen, wenn unbekannt)")
	}
	return nil
}
