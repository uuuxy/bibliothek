package inventur

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// TitelAendern verarbeitet PUT-Anfragen für ein bestehendes Buch. Geschrieben
// werden die Felder, die der Rumpf nennt; die Maske schickt die, die sie geändert hat.
func (t *Tueren) TitelAendern(antwort http.ResponseWriter, anfrage *http.Request) {
	id, ok := titelIDAusPfad(antwort, anfrage)
	if !ok {
		return
	}

	eingabe, felder, fehler := leseAenderung(anfrage)
	if fehler != nil {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, fehler)
		return
	}

	if validierungsFehler := bereinigeTitelEingabe(&eingabe); validierungsFehler != nil {
		apierrors.AntworteFehler(antwort, http.StatusBadRequest, validierungsFehler)
		return
	}
	schlagworte, fehler := schlagworteAusEingabe(eingabe.Schlagworte)
	if fehler != nil {
		schreibeFehler(antwort, http.StatusBadRequest, repository.MitGrund(meldungSchlagworteNichtGespeichert, fehler))
		return
	}

	// Beim Ändern ist ein leerer Titel ein Fehler, kein Anlass für einen Platzhalter: Davor
	// steht ein Mensch, der das Feld geleert hat, oder ein Formular, das es nie befüllt hat.
	// Die Katalogdienste füllen hier nichts nach — sonst wartete das Speichern auf sie, und
	// in der Akte stünde, was niemand eingetragen und niemand gesehen hat.
	if slices.Contains(felder, "title") && eingabe.Titel == "" {
		schreibeFehler(antwort, http.StatusBadRequest,
			"Der Titel darf nicht leer sein. Beim Ändern setzt das Programm keinen Platzhalter ein.")
		return
	}

	buch := Titel{
		ISBN:                    eingabe.ISBN,
		Titel:                   eingabe.Titel,
		Autor:                   eingabe.Autor,
		CoverURL:                eingabe.CoverURL,
		Fach:                    eingabe.Fach,
		Zweig:                   eingabe.Schulzweig,
		IstLernmittel:           eingabe.IstLernmittel,
		ZuletztGezaehlt:         eingabe.ZaehlDatum,
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

	if fehler := t.abfragen.AendereTitel(anfrage.Context(), id, buch, felder, bestandsangabe(anfrage, eingabe)); fehler != nil {
		antworteAufAenderungsfehler(antwort, id, fehler)
		return
	}

	buch.ID = id
	schreibeJSON(antwort, http.StatusOK, map[string]any{"message": "Titel gespeichert", "data": t.gespeichert(anfrage.Context(), buch)})
}

// leseAenderung liest den Rumpf einer Änderung: die Werte und die Namen der Felder, die er
// nennt. Ein Feld, das fehlt, bleibt am Titel, wie es ist; eines, das titelEingabe nicht kennt,
// ist ein Fehler, sonst hielte der Aufrufer ein vertipptes Feld für gespeichert. Groß- und
// Kleinschreibung der Namen zählt nicht, wie beim Dekodieren in titelEingabe.
func leseAenderung(anfrage *http.Request) (titelEingabe, []string, error) {
	var eingabe titelEingabe
	var roh json.RawMessage
	var genannt map[string]json.RawMessage
	if fehler := json.NewDecoder(anfrage.Body).Decode(&roh); fehler != nil {
		return eingabe, nil, fehler
	}
	streng := json.NewDecoder(bytes.NewReader(roh))
	streng.DisallowUnknownFields()
	if fehler := streng.Decode(&eingabe); fehler != nil {
		return eingabe, nil, fehler
	}
	if fehler := json.Unmarshal(roh, &genannt); fehler != nil {
		return eingabe, nil, fehler
	}
	felder := []string{}
	for _, feld := range titelFelder {
		for schluessel := range genannt {
			if strings.EqualFold(schluessel, feld.name) {
				felder = append(felder, feld.name)
				break
			}
		}
	}
	return eingabe, felder, nil
}

// antworteAufAenderungsfehler ordnet ein, warum ein Titel sich nicht ändern ließ: doppelte
// ISBN, veralteter Bestand, unbekannter Titel, geleerter Autor, ISBN-Format, halbe Spanne,
// Mehrjahresband ohne Spanne, kleinerer Bestand ohne Sitzung; alles andere 500.
func antworteAufAenderungsfehler(antwort http.ResponseWriter, id string, fehler error) {
	if errors.Is(fehler, errISBNDoppelt) {
		schreibeDubletteISBN(antwort, fehler)
		return
	}
	if meldung, ok := meldungZurSpanne(fehler); ok {
		schreibeFehler(antwort, http.StatusBadRequest, meldung)
		return
	}
	var veraltet *bestandVeraltet
	if errors.As(fehler, &veraltet) {
		// Mit dem Stand in der Antwort stellt die Maske ihr Feld nach, ohne dass die
		// übrigen Eingaben verloren gehen.
		schreibeJSON(antwort, http.StatusConflict, map[string]any{
			"error":   veraltet.Meldung(),
			"bestand": veraltet.Aktuell,
		})
		return
	}
	if errors.Is(fehler, repository.ErrTitelNichtGefunden) {
		schreibeFehler(antwort, http.StatusNotFound, "Buch nicht gefunden")
		return
	}
	if errors.Is(fehler, errAutorGeleert) {
		schreibeFehler(antwort, http.StatusBadRequest,
			"Der Autor darf nicht leer sein. Beim Ändern setzt das Programm keinen Platzhalter ein.")
		return
	}
	if errors.Is(fehler, errISBNFormat) {
		schreibeFehler(antwort, http.StatusBadRequest, "Die ISBN ist ungültig")
		return
	}
	if errors.Is(fehler, repository.ErrAussonderungOhneBearbeiter) {
		schreibeFehler(antwort, http.StatusUnauthorized, auth.MeldungNichtAngemeldet)
		return
	}
	log.Printf("Fehler beim Aktualisieren von Buch ID %s: %v", id, fehler)
	schreibeFehler(antwort, http.StatusInternalServerError, "buch konnte nicht aktualisiert werden")
}

// bestandsangabe liest aus der Eingabe, was sie zum Bestand sagt. Ohne das Feld „stock" sagt
// sie nichts (nil), und die Exemplare bleiben unangetastet. Wer speichert, nennt die Sitzung
// der Anfrage; ohne sie lehnt der Schreibpfad einen kleineren Bestand ab.
func bestandsangabe(anfrage *http.Request, eingabe titelEingabe) *Bestandsangabe {
	if eingabe.Bestand == nil {
		return nil
	}
	angabe := &Bestandsangabe{Soll: *eingabe.Bestand, Gesehen: eingabe.BestandGesehen}
	if claims, ok := auth.ClaimsAusKontext(anfrage.Context()); ok {
		angabe.BearbeiterID = claims.KontoID
	}
	return angabe
}

// bereinigeTitelEingabe entfernt Leerzeichen an den Rändern der Felder und prüft sie; der
// Fehler ist als Meldung für die Oberfläche geschrieben.
// ISBN und Mehrjahresband prüft sie nicht: Was sich ändert und was am Titel steht, weiß erst
// der Schreibpfad, der den gespeicherten Titel liest (pruefeAenderung).
func bereinigeTitelEingabe(eingabe *titelEingabe) error {
	eingabe.ISBN = strings.TrimSpace(eingabe.ISBN)
	eingabe.Titel = strings.TrimSpace(eingabe.Titel)
	eingabe.Autor = strings.TrimSpace(eingabe.Autor)
	eingabe.CoverURL = strings.TrimSpace(eingabe.CoverURL)
	eingabe.Fach = strings.TrimSpace(eingabe.Fach)
	eingabe.Schulzweig = strings.TrimSpace(eingabe.Schulzweig)
	eingabe.Medientyp = strings.TrimSpace(eingabe.Medientyp)
	eingabe.Untertitel = strings.TrimSpace(eingabe.Untertitel)
	eingabe.Verlag = strings.TrimSpace(eingabe.Verlag)

	if eingabe.Bestand != nil && *eingabe.Bestand < 0 {
		return apierrors.BadRequest("Der Bestand darf nicht negativ sein", nil)
	}
	if eingabe.BestandGesehen != nil && *eingabe.BestandGesehen < 0 {
		return apierrors.BadRequest("Der gesehene Bestand darf nicht negativ sein", nil)
	}
	return pruefeListenpreis(eingabe.Listenpreis)
}

// pruefeListenpreis nennt den erlaubten Bereich, bevor die Datenbank ablehnt
// (chk_listenpreis_nonneg): Sonst käme beim Ändern eine 500 und beim Anlegen „buch konnte nicht
// erstellt werden", beides ohne zu sagen, was erlaubt ist. Leer (nil) heißt „nicht erfasst" und
// ist etwas anderes als 0.
func pruefeListenpreis(preis *float64) error {
	if preis != nil && *preis < 0 {
		return apierrors.BadRequest("Der Listenpreis darf nicht negativ sein. Leer lassen, wenn er unbekannt ist.", nil)
	}
	return nil
}
