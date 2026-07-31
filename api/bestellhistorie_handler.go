package api

import (
	"context"
	"net/http"
	"time"

	"bibliothek/apierrors"
)

type BestellPositionResponse struct {
	TitelName   string  `json:"titel_name"`
	ISBN        string  `json:"isbn"`
	Menge       int     `json:"menge"`
	Einzelpreis float64 `json:"einzelpreis"`
	Gesamtpreis float64 `json:"gesamtpreis"`
	// TitelID verweist auf den Titelsatz. Sie kann leer sein: Die Fremdschlüssel-Regel
	// ist ON DELETE SET NULL — eine Bestellung bleibt als Beleg bestehen, auch wenn der
	// Titel später aus dem Katalog verschwindet. Die Oberfläche darf daraus also keinen
	// Verweis bauen, ohne vorher zu prüfen.
	TitelID string `json:"titel_id,omitempty"`
	// EtikettenOffen zählt die Exemplare DIESES TITELS ohne gedrucktes Etikett — nicht
	// die dieser Lieferung. Eine Bestellposition kennt nur den Titel; welches Exemplar
	// aus welcher Lieferung stammt, steht nirgends. Die Zahl ist damit ehrlich das, was
	// die Nachdruck-Liste beim Filtern auf diesen Titel auch zeigen wird.
	//
	// Sie trägt den Verweis: Ohne sie müsste die Oberfläche blind verlinken und öffnete
	// in der Hälfte der Fälle eine leere Liste — ein Verweis, der ins Leere führt,
	// entwertet alle anderen gleich mit.
	EtikettenOffen int `json:"etiketten_offen"`
}

type BestellVerlaufResponse struct {
	ID              string                    `json:"id"`
	LieferantName   string                    `json:"lieferant_name"`
	LieferantEmail  string                    `json:"lieferant_email"`
	Kundennummer    string                    `json:"kundennummer"`
	Bestelldatum    time.Time                 `json:"bestelldatum"`
	Gesamtbetrag    float64                   `json:"gesamtbetrag"`
	AnzahlExemplare int                       `json:"anzahl_exemplare"`
	Positionen      []BestellPositionResponse `json:"positionen"`
}

// GetBestellhistorieHandler returns all past orders with their line items, newest first.
func (s *Server) GetBestellhistorieHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		orders, orderIndex, err := s.ladeBestellhistorie(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(orders) == 0 {
			RespondJSON(w, http.StatusOK, orders)
			return
		}

		if err := s.ladeBestellhistoriePositionen(ctx, orders, orderIndex); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, orders)
	}
}

// ladeBestellhistorie lädt alle Bestellköpfe (neueste zuerst) und einen Index
// Bestell-ID → Position im Slice für das spätere Zuordnen der Positionen.
func (s *Server) ladeBestellhistorie(ctx context.Context) ([]BestellVerlaufResponse, map[string]int, error) {
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT id, lieferant_name, lieferant_email, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare
		FROM bestellungen_verlauf
		ORDER BY bestelldatum DESC
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	orders := make([]BestellVerlaufResponse, 0)
	orderIndex := map[string]int{}

	for rows.Next() {
		var o BestellVerlaufResponse
		if err := rows.Scan(&o.ID, &o.LieferantName, &o.LieferantEmail, &o.Kundennummer,
			&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare); err != nil {
			return nil, nil, err
		}
		o.Positionen = []BestellPositionResponse{}
		orderIndex[o.ID] = len(orders)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return orders, orderIndex, nil
}

// ladeBestellhistoriePositionen lädt alle Positionen in einer Query und ordnet sie den
// Bestellungen über orderIndex zu (Gesamtpreis wird pro Position berechnet).
func (s *Server) ladeBestellhistoriePositionen(ctx context.Context, orders []BestellVerlaufResponse, orderIndex map[string]int) error {
	// etikettenOffenBedingung ist dieselbe Definition, die auch die Nachdruck-Liste und
	// ihr Zähler verwenden (etiketten_offen.go). Wäre sie hier abgeschrieben, zeigte der
	// Verweis irgendwann eine andere Zahl als die Liste, die er öffnet.
	posRows, err := s.DB.Pool.Query(ctx, `
		SELECT p.bestellung_id, p.titel_name, p.isbn, p.menge, p.einzelpreis,
		       coalesce(p.titel_id::text, ''),
		       (SELECT count(*) FROM buecher_exemplare e
		         WHERE e.titel_id = p.titel_id AND `+etikettenOffenBedingung+`)
		FROM bestellungen_positionen p
		WHERE p.bestellung_id = ANY(
			SELECT id FROM bestellungen_verlauf ORDER BY bestelldatum DESC
		)
		ORDER BY p.bestellung_id, p.titel_name
	`)
	if err != nil {
		return err
	}
	defer posRows.Close()

	for posRows.Next() {
		var bestellungID string
		var pos BestellPositionResponse
		if err := posRows.Scan(&bestellungID, &pos.TitelName, &pos.ISBN, &pos.Menge, &pos.Einzelpreis,
			&pos.TitelID, &pos.EtikettenOffen); err != nil {
			return err
		}
		pos.Gesamtpreis = float64(pos.Menge) * pos.Einzelpreis
		if idx, ok := orderIndex[bestellungID]; ok {
			orders[idx].Positionen = append(orders[idx].Positionen, pos)
		}
	}
	return posRows.Err()
}
