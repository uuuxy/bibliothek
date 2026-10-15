package druck

import (
	"errors"
	"net/http"
	"strings"
	"time"

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
// oder status=alle zeigt sie auch vermerkte Exemplare: Nach einem Papierstau oder einem zu
// späten Stichtag tragen sie den Vermerk, aber kein Etikett.
//
// @Summary      Exemplare ohne gedrucktes Etikett
// @Tags         books
// @Produce      json
// @Param        q       query   string  false  "Filter über Titel oder Barcode"
// @Param        status  query   string  false  "offen (Vorgabe) | erledigt | alle"
// @Success      200  {array}  ExemplarOhneEtikett
// @Router       /exemplare/etiketten-offen [get]
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
//
// @Summary      Anzahl der Exemplare ohne gedrucktes Etikett
// @Tags         books
// @Produce      json
// @Param        q       query   string  false  "Filter über Titel oder Barcode"
// @Param        status  query   string  false  "offen (Vorgabe) | erledigt | alle"
// @Param        bis     query   string  false  "Stichtag JJJJ-MM-TT (nur mit status=offen sinnvoll)"
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-offen/anzahl [get]
func (t *Tueren) EtikettenOffenAnzahl() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		// Mit Stichtag nennt dieselbe Tür, wie viele Exemplare das Aufräumen des Altbestands
		// träfe: die Zahl, die der Betreiber vor dem Bestätigen sieht.
		bisText := strings.TrimSpace(r.URL.Query().Get("bis"))
		var bis *time.Time
		if bisText != "" {
			geparst, err := time.Parse(tuer.DatumsformatISO, bisText)
			if err != nil {
				return apierrors.BadRequest("Stichtag muss im Format JJJJ-MM-TT angegeben werden", err)
			}
			bis = &geparst
		}

		anzahl, err := repository.ZaehleEtikettenExemplare(r.Context(), t.DB.Pool,
			r.URL.Query().Get("status"), bis, etikettenSuchtext(r))
		if err != nil {
			return apierrors.Internal("Fehler beim Zählen der offenen Etiketten", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]int{"anzahl": anzahl})
		return nil
	})
}

// etikettenAltbestandAnfrage nennt den Stichtag, bis zu dem aufgeräumt wird.
type etikettenAltbestandAnfrage struct {
	Bis string `json:"bis"` // JJJJ-MM-TT, einschließlich
}

// EtikettenAltbestandVermerken vermerkt alle Exemplare bis zu einem Stichtag als gedruckt.
//
// Für den Altbestand hat niemand ein Etikett vermerkt, und ohne Vermerk zeigt die
// Nachdruck-Liste den ganzen Bestand. Den Stichtag wählt der Betreiber, weil nur er weiß, ab
// wann sein Regal beklebt ist; eine Migration träfe auch die Lieferung, die wirklich kein
// Etikett hat. Die Oberfläche nennt vorher die Zahl und verlangt eine Bestätigung;
// zurücknehmen lässt sich der Vermerk mit EtikettenZuruecksetzen.
//
// @Summary      Etiketten des Altbestands als gedruckt vermerken
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-altbestand [post]
func (t *Tueren) EtikettenAltbestandVermerken() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var req etikettenAltbestandAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		bis, err := time.Parse(tuer.DatumsformatISO, req.Bis)
		if err != nil {
			return apierrors.BadRequest("Stichtag muss im Format JJJJ-MM-TT angegeben werden", err)
		}

		markiert, err := repository.VermerkeAltbestandEtiketten(r.Context(), t.DB.Pool, bis)
		if err != nil {
			return apierrors.Internal("Fehler beim Vermerken des Altbestands", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]int64{"markiert": markiert})
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
//
// @Summary      Etiketten als gedruckt markieren
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-gedruckt [post]
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
// sobald das PDF erzeugt ist (vermerkeGedruckt in frontend/src/lib/stores/etiketten.svelte.js),
// und ein zu später Stichtag beim Altbestand trifft auch die frische Lieferung. Nach einem
// Papierstau oder einem solchen Stichtag holt diese Tür die Exemplare in die Liste zurück; erst
// damit ist auch das Vermerken von Hand unbedenklich.
//
// @Summary      Etiketten wieder als offen markieren
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-zuruecksetzen [post]
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
