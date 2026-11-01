package druck

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// etikettenOffenLimit deckelt die Liste. Wer mehr als 300 Etiketten am Stück nachdruckt,
// druckt keinen Nachzügler mehr nach, sondern den halben Bestand — dafür ist der Weg über
// die Titelsuche im Druck-Center gedacht.
const etikettenOffenLimit = 300

// ExemplarOhneEtikett ist eine Zeile der Nachdruck-Liste. Die Namen in JSON (barcode_id, titel,
// autor) sind die der Aufträge, die frontend/src/lib/stores/etiketten.svelte.js druckt, damit die
// Oberfläche die Liste unverändert übergeben kann.
type ExemplarOhneEtikett struct {
	BarcodeID string `json:"barcode_id"`
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	// ZugangAm ist der Tag, an dem das Exemplar in den Bestand kam (zugang_am, für Zeilen ohne
	// Zugangsdatum erworben_am).
	ZugangAm string `json:"zugang_am"`

	// EtikettGedruckt unterscheidet in den Ansichten status=erledigt und status=alle die
	// Zeilen, die noch ein Etikett brauchen.
	EtikettGedruckt bool `json:"etikett_gedruckt"`
}

// etikettenSuchtext liest ?q= für Liste und Zähler in der Form, in der Titeltexte gespeichert
// sind (repository.TiteltextNormalform): Beide vergleichen den Text als Wortlaut.
func etikettenSuchtext(r *http.Request) string {
	return repository.TiteltextNormalform(r.URL.Query().Get("q"))
}

// EtikettenOffen listet die Exemplare, deren Etikett nicht als gedruckt vermerkt ist, die
// neuesten zuerst: Gesucht wird meist die Lieferung von gestern.
//
// Sie führt zu einer Lieferung, die freigegeben wurde, ohne dass ihre Etiketten aus dem Drucker
// kamen, etwa weil der Hinweis nach dem Wareneingang weggeklickt wurde. Mit status=erledigt
// oder status=alle zeigt sie auch vermerkte Exemplare: Nach einem Papierstau tragen sie den
// Vermerk, aber kein Etikett.
func (t *Tueren) EtikettenOffen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		zeilen, err := repository.LadeEtikettenExemplare(r.Context(), t.DB.Pool,
			r.URL.Query().Get("status"), etikettenSuchtext(r), etikettenOffenLimit)
		if err != nil {
			if errors.Is(err, repository.ErrZeileUnlesbar) {
				return apierrors.Internal("Fehler beim Lesen der offenen Etiketten", err)
			}
			return apierrors.Internal("Fehler beim Laden der offenen Etiketten", err)
		}

		// Nie nil: Eine leere Liste kommt als [] an; an null bricht die Oberfläche bei .length
		// ab.
		liste := make([]ExemplarOhneEtikett, 0, len(zeilen))
		for _, z := range zeilen {
			liste = append(liste, ExemplarOhneEtikett(z))
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}

// EtikettenOffenAnzahl zählt mit denselben Filtern wie EtikettenOffen (status, q), aber ohne
// deren Grenze von 300 Zeilen.
//
// Eine eigene Tür, weil der Hinweis im Bestellwesen die Zahl bei jedem Öffnen lädt und die
// Liste ab 300 eine falsche Zahl nennte. Die Fußzeile der Liste sagt mit ihr, wie viele
// Exemplare es über die gezeigten hinaus gibt.
func (t *Tueren) EtikettenOffenAnzahl() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		anzahl, err := repository.ZaehleEtikettenExemplare(r.Context(), t.DB.Pool,
			r.URL.Query().Get("status"), etikettenSuchtext(r))
		if err != nil {
			return apierrors.Internal("Fehler beim Zählen der offenen Etiketten", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]int{"anzahl": anzahl})
		return nil
	})
}

// etikettenVermerkAnfrage nennt die Exemplare, deren Vermerk „Etikett gedruckt" gesetzt oder
// zurückgenommen wird.
type etikettenVermerkAnfrage struct {
	BarcodeIDs []string `json:"barcode_ids"`
}

// EtikettenGedrucktVermerken bucht den Druck gegen. Ohne den Vermerk bliebe jedes Exemplar in
// der Nachdruck-Liste, und ihr Haken „erledigt" bedeutete nichts.
func (t *Tueren) EtikettenGedrucktVermerken() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var req etikettenVermerkAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		if len(req.BarcodeIDs) == 0 {
			return apierrors.BadRequest("Keine Exemplare angegeben", nil)
		}

		markiert, err := repository.VermerkeEtikettenGedruckt(r.Context(), t.DB.Pool, req.BarcodeIDs)
		if err != nil {
			return apierrors.Internal("Fehler beim Vermerken der gedruckten Etiketten", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]int64{"markiert": markiert})
		return nil
	})
}

// EtikettenZuruecksetzen nimmt den Vermerk „Etikett gedruckt" wieder zurück.
//
// Ob ein Etikett aus dem Drucker kam, weiß das Programm nicht: Der Druck wird gegengebucht,
// sobald das PDF erzeugt ist (vermerkeGedruckt in frontend/src/lib/stores/etiketten.svelte.js).
// Nach einem Papierstau holt diese Tür die Exemplare in die Liste zurück; erst damit ist auch das
// Vermerken von Hand unbedenklich.
func (t *Tueren) EtikettenZuruecksetzen() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var req etikettenVermerkAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		if len(req.BarcodeIDs) == 0 {
			return apierrors.BadRequest("Keine Exemplare angegeben", nil)
		}

		zurueckgesetzt, err := repository.NimmEtikettenVermerk(r.Context(), t.DB.Pool, req.BarcodeIDs)
		if err != nil {
			return apierrors.Internal("Fehler beim Zurücksetzen der Etiketten", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]int64{"zurueckgesetzt": zurueckgesetzt})
		return nil
	})
}
