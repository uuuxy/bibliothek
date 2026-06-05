package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
)

// OpacTitel is a DSGVO-compliant book view for the public catalog.
// Contains no loan data and no reader data.
type OpacTitel struct {
	ID         string `json:"id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	ISBN       string `json:"isbn,omitempty"`
	CoverURL   string `json:"cover_url,omitempty"`
	Verfuegbar int    `json:"verfuegbar"` // copies currently available
	Gesamt     int    `json:"gesamt"`     // total copies
}

// PublicCatalogSearchHandler handles GET /api/opac/suche?q=...
// Public endpoint: no auth required. Never exposes loan or reader data (DSGVO).
func (s *Server) PublicCatalogSearchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		// Join only buecher_titel and buecher_exemplare.
		// The LEFT JOIN on ausleihen is filtered to active loans (rueckgabe_am IS NULL)
		// only to determine availability — no ausleihe column values are returned.
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT bt.id, bt.titel, COALESCE(bt.autor, ''), COALESCE(bt.isbn, ''),
			       COALESCE(bt.cover_url, ''),
			       COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
			       COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND coalesce(e.zustand_notiz, '') NOT LIKE 'Im Zulauf%' AND coalesce(e.zustand_notiz, '') != 'bestellt' AND coalesce(e.zustand_notiz, '') NOT LIKE 'Bestellt%') AS gesamt
			FROM buecher_titel bt
			LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
			LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			WHERE bt.search_vector @@ plainto_tsquery('german', $1)
			   OR bt.titel ILIKE '%' || $1 || '%'
			   OR bt.autor ILIKE '%' || $1 || '%'
			   OR bt.isbn ILIKE '%' || $1 || '%'
			GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.cover_url
			HAVING COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND coalesce(e.zustand_notiz, '') NOT LIKE 'Im Zulauf%' AND coalesce(e.zustand_notiz, '') != 'bestellt' AND coalesce(e.zustand_notiz, '') NOT LIKE 'Bestellt%') > 0
			ORDER BY bt.titel
			LIMIT 50
		`, q)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		result := make([]OpacTitel, 0)
		for rows.Next() {
			var t OpacTitel
			if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.CoverURL, &t.Verfuegbar, &t.Gesamt); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			result = append(result, t)
		}
		if result == nil {
			result = []OpacTitel{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}
