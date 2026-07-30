package api

import (
	"net/http"
	"strings"

	"bibliothek/apierrors"
)

// etikettenOffenLimit deckelt die Liste. Wer mehr als 300 Etiketten am Stück nachdruckt,
// druckt keinen Nachzügler mehr nach, sondern den halben Bestand — dafür ist der Weg über
// die Titelsuche im Druck-Center gedacht.
const etikettenOffenLimit = 300

// ExemplarOhneEtikett ist eine Zeile der Nachdruck-Liste. Die Feldnamen barcode_id/titel/
// autor sind KEIN Zufall: In genau dieser Form nimmt der Etikettendruck seine Aufträge
// entgegen (printQueue → labels.svelte.js), die Liste kann also direkt übergeben werden.
type ExemplarOhneEtikett struct {
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	ErworbenAm string `json:"erworben_am"`
}

// EtikettenOffenHandler listet Exemplare, deren Barcode-Etikett noch nicht gedruckt wurde.
//
// Der Anlass: Eine Lieferung kann im System freigegeben sein, ohne dass die Etiketten je
// aus dem Drucker kamen (z. B. weil der Hinweis nach dem Wareneingang weggeklickt wurde).
// Danach gab es keinen Weg mehr zu genau diesen Exemplaren zurück — man hätte jeden Titel
// einzeln suchen müssen, ohne zu wissen, welche es sind.
//
// Sortiert wird NEUESTE ZUERST. Das ist der praktische Fall: Gesucht wird die Lieferung von
// gestern, nicht ein Exemplar von 2019.
//
// @Summary      Exemplare ohne gedrucktes Etikett
// @Tags         books
// @Produce      json
// @Param        q  query     string  false  "Filter über Titel oder Barcode"
// @Success      200  {array}  ExemplarOhneEtikett
// @Router       /exemplare/etiketten-offen [get]
func (s *Server) EtikettenOffenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		suche := strings.TrimSpace(r.URL.Query().Get("q"))

		// Ausgesonderte Exemplare stehen nicht mehr im Regal — für sie ein Etikett zu
		// drucken wäre immer falsch.
		rows, err := s.DB.Pool.Query(r.Context(), `
			SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), to_char(e.erworben_am, 'YYYY-MM-DD')
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE e.etikett_gedruckt = false
			  AND e.ist_ausgesondert = false
			  AND ($1 = '' OR t.titel ILIKE '%' || $1 || '%' OR e.barcode_id ILIKE '%' || $1 || '%')
			ORDER BY e.erworben_am DESC, e.erstellt_am DESC, e.barcode_id
			LIMIT $2
		`, suche, etikettenOffenLimit)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der offenen Etiketten", err)
		}
		defer rows.Close()

		// Nie nil: Eine leere Liste muss als [] beim Client ankommen, sonst bricht dort
		// .length ab (siehe TestListStudentsLeereListeIstArray).
		liste := make([]ExemplarOhneEtikett, 0)
		for rows.Next() {
			var e ExemplarOhneEtikett
			if err := rows.Scan(&e.BarcodeID, &e.Titel, &e.Autor, &e.ErworbenAm); err != nil {
				return apierrors.Internal("Fehler beim Lesen der offenen Etiketten", err)
			}
			liste = append(liste, e)
		}
		if err := rows.Err(); err != nil {
			return apierrors.Internal("Fehler beim Lesen der offenen Etiketten", err)
		}

		RespondJSON(w, http.StatusOK, liste)
		return nil
	})
}

// EtikettenGedrucktRequest nennt die Exemplare, deren Etiketten gedruckt wurden.
type EtikettenGedrucktRequest struct {
	BarcodeIDs []string `json:"barcode_ids"`
}

// EtikettenGedrucktHandler bucht den Druck gegen.
//
// Ohne diesen Schritt wäre die Nachdruck-Liste wertlos: etikett_gedruckt wurde bis hierher
// NIRGENDS auf true gesetzt — der Wert stand seit dem Anlegen der Tabelle auf false und
// blieb es. Die Liste hätte also dauerhaft den kompletten Bestand angezeigt statt der
// Nachzügler, und der Haken "erledigt" wäre eine Anzeige ohne Bedeutung gewesen.
//
// @Summary      Etiketten als gedruckt markieren
// @Tags         books
// @Accept       json
// @Success      200  {object}  map[string]int
// @Router       /exemplare/etiketten-gedruckt [post]
func (s *Server) EtikettenGedrucktHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req EtikettenGedrucktRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		if len(req.BarcodeIDs) == 0 {
			return apierrors.BadRequest("keine Exemplare angegeben", nil)
		}

		tag, err := s.DB.Pool.Exec(r.Context(), `
			UPDATE buecher_exemplare SET etikett_gedruckt = true, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE barcode_id = ANY($1) AND etikett_gedruckt = false
		`, req.BarcodeIDs)
		if err != nil {
			return apierrors.Internal("Fehler beim Vermerken der gedruckten Etiketten", err)
		}

		RespondJSON(w, http.StatusOK, map[string]int64{"markiert": tag.RowsAffected()})
		return nil
	})
}
