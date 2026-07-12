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
	posRows, err := s.DB.Pool.Query(ctx, `
		SELECT bestellung_id, titel_name, isbn, menge, einzelpreis
		FROM bestellungen_positionen
		WHERE bestellung_id = ANY(
			SELECT id FROM bestellungen_verlauf ORDER BY bestelldatum DESC
		)
		ORDER BY bestellung_id, titel_name
	`)
	if err != nil {
		return err
	}
	defer posRows.Close()

	for posRows.Next() {
		var bestellungID string
		var pos BestellPositionResponse
		if err := posRows.Scan(&bestellungID, &pos.TitelName, &pos.ISBN, &pos.Menge, &pos.Einzelpreis); err != nil {
			return err
		}
		pos.Gesamtpreis = float64(pos.Menge) * pos.Einzelpreis
		if idx, ok := orderIndex[bestellungID]; ok {
			orders[idx].Positionen = append(orders[idx].Positionen, pos)
		}
	}
	return posRows.Err()
}
