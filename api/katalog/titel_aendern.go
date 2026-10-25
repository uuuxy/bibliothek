package katalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// TitelAendern verarbeitet PUT-Anfragen für ein bestehendes Buch. Geschrieben
// werden die Felder, die der Rumpf nennt; die Maske schickt die, die sie geändert hat.
func (t *Tueren) TitelAendern() http.HandlerFunc {
	return func(antwort http.ResponseWriter, anfrage *http.Request) {
		id, ok := titelKennungAusPfad(antwort, anfrage)
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
		schlagworte, fehler := schlagworteAusAnfrage(eingabe.Schlagworte)
		if fehler != nil {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, repository.MitGrund(meldungSchlagworteNichtGespeichert, fehler), nil)
			return
		}

		// Beim Ändern ist ein leerer Titel ein Fehler, kein Anlass für einen Platzhalter: Davor
		// steht ein Mensch, der das Feld geleert hat, oder ein Formular, das es nie befüllt hat.
		// Die Katalogdienste füllen hier nichts nach — sonst wartete das Speichern auf sie, und
		// in der Akte stünde, was niemand eingetragen und niemand gesehen hat.
		if slices.Contains(felder, "title") && eingabe.Titel == "" {
			apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Der Titel darf nicht leer sein. Beim Ändern setzt das Programm keinen Platzhalter ein.", nil)

			return
		}

		buch := repository.KatalogTitel{
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

		if fehler := t.titelAbfragen().AendereTitel(anfrage.Context(), id, buch, felder, bestandsangabe(anfrage, eingabe)); fehler != nil {
			antworteAufAenderungsfehler(antwort, id, fehler)
			return
		}

		buch.ID = id
		tuer.AntworteJSON(antwort, http.StatusOK, t.gespeichert(anfrage.Context(), buch))
	}
}

// leseAenderung liest den Rumpf einer Änderung: die Werte und die Namen der Felder, die er
// nennt. Ein Feld, das fehlt, bleibt am Titel, wie es ist; eines, das titelAnfrage nicht kennt,
// ist ein Fehler, sonst hielte der Aufrufer ein vertipptes Feld für gespeichert. Groß- und
// Kleinschreibung der Namen zählt nicht, wie beim Dekodieren in titelAnfrage.
func leseAenderung(anfrage *http.Request) (titelAnfrage, []string, error) {
	var eingabe titelAnfrage
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
	for _, name := range repository.TitelFeldNamen() {
		for schluessel := range genannt {
			if strings.EqualFold(schluessel, name) {
				felder = append(felder, name)
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
	if errors.Is(fehler, repository.ErrISBNDoppelt) {
		schreibeDubletteISBN(antwort, fehler)
		return
	}
	if meldung, ok := meldungZurSpanne(fehler); ok {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, meldung, nil)
		return
	}
	var veraltet *repository.BestandVeraltet
	if errors.As(fehler, &veraltet) {
		// Mit dem Stand in der Antwort stellt die Maske ihr Feld nach, ohne dass die
		// übrigen Eingaben verloren gehen.
		tuer.AntworteJSON(antwort, http.StatusConflict, map[string]any{
			"error":   veraltet.Meldung(),
			"bestand": veraltet.Aktuell,
		})

		return
	}
	if errors.Is(fehler, repository.ErrTitelNichtGefunden) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusNotFound, "Buch nicht gefunden", nil)
		return
	}
	if errors.Is(fehler, repository.ErrAutorGeleert) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Der Autor darf nicht leer sein. Beim Ändern setzt das Programm keinen Platzhalter ein.", nil)

		return
	}
	if errors.Is(fehler, repository.ErrISBNFormat) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusBadRequest, "Die ISBN ist ungültig", nil)
		return
	}
	if errors.Is(fehler, repository.ErrAussonderungOhneBearbeiter) {
		apierrors.AntworteFehlerMitMeldung(antwort, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}
	slog.Error("Fehler beim Aktualisieren eines Titels", "titel", id, "fehler", fehler)
	apierrors.AntworteFehlerMitMeldung(antwort, http.StatusInternalServerError, "buch konnte nicht aktualisiert werden", nil)
}

// bestandsangabe liest aus der Eingabe, was sie zum Bestand sagt. Ohne das Feld „stock" sagt
// sie nichts (nil), und die Exemplare bleiben unangetastet. Wer speichert, nennt die Sitzung
// der Anfrage; ohne sie lehnt der Schreibpfad einen kleineren Bestand ab.
func bestandsangabe(anfrage *http.Request, eingabe titelAnfrage) *repository.Bestandsangabe {
	if eingabe.Bestand == nil {
		return nil
	}
	angabe := &repository.Bestandsangabe{Soll: *eingabe.Bestand, Gesehen: eingabe.BestandGesehen}
	if claims, ok := auth.ClaimsAusKontext(anfrage.Context()); ok {
		angabe.BearbeiterID = claims.KontoID
	}
	return angabe
}

// bereinigeTitelEingabe entfernt Leerzeichen an den Rändern der Felder und prüft sie; der
// Fehler ist als Meldung für die Oberfläche geschrieben.
// ISBN und Mehrjahresband prüft sie nicht: Was sich ändert und was am Titel steht, weiß erst
// der Schreibpfad, der den gespeicherten Titel liest (pruefeAenderung).
func bereinigeTitelEingabe(eingabe *titelAnfrage) error {
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

// Sätze zu den beiden Regeln für die Oberfläche; die Fehlerwerte oben stehen im Log.
const (
	meldungJahrgangsSpanne = "Beim Jahrgang gehören „von“ und „bis“ zusammen oder bleiben beide leer. Beide liegen " +
		"zwischen 1 und 13, „von“ nicht über „bis“."
	meldungMehrjahresband = "Mehrjahresband nur bei einem Lernmittel und nur über mehr als einen Jahrgang, mit „bis“ " +
		"über „von“ und beiden zwischen 1 und 13"
)

// meldungZurSpanne nennt den Satz zu einem Verstoß gegen die Spanne oder den Mehrjahresband;
// ok ist false für jeden anderen Fehler.
func meldungZurSpanne(err error) (meldung string, ok bool) {
	switch {
	case errors.Is(err, repository.ErrJahrgangsSpanne):
		return meldungJahrgangsSpanne, true
	case errors.Is(err, repository.ErrMehrjahresband):
		return meldungMehrjahresband, true
	}
	return "", false
}
