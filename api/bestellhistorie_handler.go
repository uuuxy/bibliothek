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

	// BietetBestellbestaetigung: Lieferant dieser Bestellung bietet den externen
	// Bestätigungsschritt an (z. B. Naacher) — steuert, ob die Oberfläche den
	// Bestätigen-Schritt überhaupt anzeigt. false, wenn der Lieferant inzwischen
	// gelöscht wurde (COALESCE gegen lieferant_id IS NULL).
	BietetBestellbestaetigung bool `json:"bietet_bestellbestaetigung"`
	// BestaetigtAm: Zeitpunkt der externen Bestätigung, NULL solange unbestätigt.
	BestaetigtAm *time.Time `json:"bestaetigt_am,omitempty"`
	// EtikettenGroesse: beim Bestätigen gewählte Größe ('klein'/'gross'), NULL solange
	// unbestätigt.
	EtikettenGroesse *string `json:"etiketten_groesse,omitempty"`
	// BestaetigtDurch unterscheidet die beiden Wege: 'lieferant' = über den verschickten
	// Link selbst bestätigt, 'bibliothek' = hier von Hand nachgetragen. Ohne diese
	// Unterscheidung sähe eine Vermutung aus wie eine Rückmeldung.
	BestaetigtDurch *string `json:"bestaetigt_durch,omitempty"`
	// LinkAktiv: Für diese Bestellung ist ein gültiger Bestätigungs-Link unterwegs. Der
	// Link selbst kann hier nicht stehen — gespeichert ist nur sein Hash.
	LinkAktiv bool `json:"link_aktiv"`
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
	// LEFT JOIN + COALESCE: eine Bestellung überlebt ihren gelöschten Lieferanten als
	// Beleg (lieferant_id ON DELETE SET NULL) — dann gilt bietet_bestellbestaetigung=false.
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum,
		       b.gesamtbetrag, b.anzahl_exemplare, coalesce(l.bietet_bestellbestaetigung, false),
		       b.bestaetigt_am, b.etiketten_groesse, b.bestaetigt_durch,
		       (b.bestaetigungs_token_hash IS NOT NULL
		        AND (b.token_gueltig_bis IS NULL OR b.token_gueltig_bis > now()))
		FROM bestellungen_verlauf b
		LEFT JOIN lieferanten l ON l.id = b.lieferant_id
		ORDER BY b.bestelldatum DESC
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
			&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare, &o.BietetBestellbestaetigung,
			&o.BestaetigtAm, &o.EtikettenGroesse, &o.BestaetigtDurch, &o.LinkAktiv); err != nil {
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
