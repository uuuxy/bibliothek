package api

import (
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

		rows, err := s.DB.Pool.Query(ctx, `
			SELECT id, lieferant_name, lieferant_email, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare
			FROM bestellungen_verlauf
			ORDER BY bestelldatum DESC
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		orders := make([]BestellVerlaufResponse, 0)
		orderIndex := map[string]int{}

		for rows.Next() {
			var o BestellVerlaufResponse
			if err := rows.Scan(&o.ID, &o.LieferantName, &o.LieferantEmail, &o.Kundennummer,
				&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			o.Positionen = []BestellPositionResponse{}
			orderIndex[o.ID] = len(orders)
			orders = append(orders, o)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if len(orders) == 0 {
			RespondJSON(w, http.StatusOK, orders)
			return
		}

		// Load all line items for the returned orders in one query
		posRows, err := s.DB.Pool.Query(ctx, `
			SELECT bestellung_id, titel_name, isbn, menge, einzelpreis
			FROM bestellungen_positionen
			WHERE bestellung_id = ANY(
				SELECT id FROM bestellungen_verlauf ORDER BY bestelldatum DESC
			)
			ORDER BY bestellung_id, titel_name
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer posRows.Close()

		for posRows.Next() {
			var bestellungID string
			var pos BestellPositionResponse
			if err := posRows.Scan(&bestellungID, &pos.TitelName, &pos.ISBN, &pos.Menge, &pos.Einzelpreis); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			pos.Gesamtpreis = float64(pos.Menge) * pos.Einzelpreis
			if idx, ok := orderIndex[bestellungID]; ok {
				orders[idx].Positionen = append(orders[idx].Positionen, pos)
			}
		}
		if err := posRows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, orders)
	}
}
