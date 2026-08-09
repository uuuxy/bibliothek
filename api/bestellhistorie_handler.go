package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"bibliothek/apierrors"
)

// BestellPositionResponse ist eine Position innerhalb einer Bestellung des Verlaufs.
// Die Feldkommentare unten tragen die eigentlichen Fallstricke — nullbare TitelID und
// die Frage, worauf sich EtikettenOffen bezieht.
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
	// die dieser Lieferung. Die Zahl ist damit ehrlich das, was die Nachdruck-Liste beim
	// Filtern auf diesen Titel auch zeigen wird.
	//
	// Hier stand bis zum 08.08.2026 als Begründung, welches Exemplar aus welcher
	// Lieferung stammt, stehe nirgends. Das stimmte, als der Satz geschrieben wurde, und
	// war seit Migration 063 falsch: buecher_exemplare.bestellung_id ist gesetzt, und die
	// Detailansicht (bestelldetail_handler.go) zeigt genau diese Exemplare. Die
	// Titel-Zählung hier bleibt trotzdem richtig — sie gehört zum Verweis auf die
	// Nachdruck-Liste, und die filtert nach Titel, nicht nach Lieferung.
	//
	// Sie trägt den Verweis: Ohne sie müsste die Oberfläche blind verlinken und öffnete
	// in der Hälfte der Fälle eine leere Liste — ein Verweis, der ins Leere führt,
	// entwertet alle anderen gleich mit.
	EtikettenOffen int `json:"etiketten_offen"`
}

// BestellVerlaufResponse ist eine Bestellung im Verlauf, mit ihren Positionen und dem
// Zustand des Bestätigungs-Wegs.
type BestellVerlaufResponse struct {
	ID              string                    `json:"id"`
	LieferantName   string                    `json:"lieferant_name"`
	LieferantEmail  string                    `json:"lieferant_email"`
	Kundennummer    string                    `json:"kundennummer"`
	Bestelldatum    time.Time                 `json:"bestelldatum"`
	Gesamtbetrag    float64                   `json:"gesamtbetrag"`
	AnzahlExemplare int                       `json:"anzahl_exemplare"`
	Positionen      []BestellPositionResponse `json:"positionen"`

	// MitBestaetigung: DIESE Bestellung ist mit einem Bestätigungs-Link rausgegangen —
	// sie hat einen Token bekommen. Steuert, ob die Oberfläche den Bestätigen-Schritt
	// überhaupt anzeigt.
	//
	// Bewusst am Token der Bestellung und NICHT am heutigen Merkmal des Lieferanten:
	// Hauptlieferant darf nur einer sein (Migration 066). Zöge die Zahl über den
	// Lieferanten, verlören alle offenen Bestellungen des bisherigen Hauptlieferanten in
	// dem Moment ihren Bestätigen-Schritt, in dem jemand anderes den Link bekommt —
	// obwohl sie mit Link rausgegangen sind und weiter auf Bestätigung warten. Der Token
	// bleibt, was er ist. Dieselbe Regel nutzt die Übersicht (bestellhistorie_uebersicht.go).
	MitBestaetigung bool `json:"mit_bestaetigung"`
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

// bestellhistorieStandardLimit / -MaxLimit deckeln die Liste.
//
// Ohne Grenze lieferte dieser Endpunkt ALLE Bestellungen samt Positionen: auf einer
// gewachsenen Datenbank (5.257 Bestellungen) waren das 2,45 MB und 3,9 Sekunden, und es
// wird jedes Schuljahr mehr. Dieselbe Bugklasse hatte das Audit-Log (247k Zeilen, 72 MB).
//
// Die Summen im Kopf der Oberfläche dürfen davon NICHT abhängen — sie kommen aus
// /api/bestellhistorie/uebersicht und zählen weiterhin alles.
const (
	bestellhistorieStandardLimit = 200
	bestellhistorieMaxLimit      = 500
)

// GetBestellhistorieHandler returns the most recent orders with their line items.
func (s *Server) GetBestellhistorieHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		limit := bestellhistorieStandardLimit
		if roh := r.URL.Query().Get("limit"); roh != "" {
			if n, err := strconv.Atoi(roh); err == nil && n > 0 {
				limit = min(n, bestellhistorieMaxLimit)
			}
		}

		orders, orderIndex, err := s.ladeBestellhistorie(ctx, limit)
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
func (s *Server) ladeBestellhistorie(ctx context.Context, limit int) ([]BestellVerlaufResponse, map[string]int, error) {
	// Kein JOIN auf lieferanten mehr nötig: Beide Bestätigungs-Angaben stehen an der
	// Bestellung selbst. Das ist auch der Grund, warum eine Bestellung ihren gelöschten
	// Lieferanten als vollständiger Beleg überlebt (lieferant_id ON DELETE SET NULL).
	rows, err := s.DB.Pool.Query(ctx, `
		SELECT b.id, b.lieferant_name, b.lieferant_email, b.kundennummer, b.bestelldatum,
		       b.gesamtbetrag, b.anzahl_exemplare,
		       b.bestaetigungs_token_hash IS NOT NULL,
		       b.bestaetigt_am, b.etiketten_groesse, b.bestaetigt_durch,
		       (b.bestaetigungs_token_hash IS NOT NULL
		        AND (b.token_gueltig_bis IS NULL OR b.token_gueltig_bis > now()))
		FROM bestellungen_verlauf b
		ORDER BY b.bestelldatum DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	orders := make([]BestellVerlaufResponse, 0)
	orderIndex := map[string]int{}

	for rows.Next() {
		var o BestellVerlaufResponse
		if err := rows.Scan(&o.ID, &o.LieferantName, &o.LieferantEmail, &o.Kundennummer,
			&o.Bestelldatum, &o.Gesamtbetrag, &o.AnzahlExemplare, &o.MitBestaetigung,
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
		       coalesce(e_counts.etiketten_offen, 0)
		FROM bestellungen_positionen p
		LEFT JOIN (
			SELECT titel_id, count(*) as etiketten_offen
			FROM buecher_exemplare e
			WHERE `+etikettenOffenBedingung+`
			GROUP BY titel_id
		) e_counts ON e_counts.titel_id = p.titel_id
		WHERE p.bestellung_id = ANY($1)
		ORDER BY p.bestellung_id, p.titel_name
	`, geladeneIDs(orderIndex))
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

// geladeneIDs liefert die IDs der tatsächlich geladenen Bestellungen.
//
// Die Positionen-Abfrage holte vorher die Positionen ALLER Bestellungen der Datenbank
// (Unterabfrage ohne Grenze) und warf die überzähligen beim Zuordnen weg. Mit dem Limit
// oben wäre das der teuerste Teil der Anfrage geblieben.
func geladeneIDs(orderIndex map[string]int) []string {
	ids := make([]string, 0, len(orderIndex))
	for id := range orderIndex {
		ids = append(ids, id)
	}
	return ids
}
