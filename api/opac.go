package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/pkg/httpresp"
	"bibliothek/repository"
)

// queryOpacTitel führt die (parametrisierte) OPAC-Suche aus und mappt die Zeilen.
// Bei einem Query- oder Iterationsfehler wird der Fehler propagiert, damit der
// öffentliche Katalog keine irreführenden Teildaten als vollständig ausliefert.
func (s *Server) queryOpacTitel(ctx context.Context, query string, args []any) ([]OpacTitel, error) {
	rows, err := s.DB.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]OpacTitel, 0)
	for rows.Next() {
		var t OpacTitel
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.CoverURL, &t.Verfuegbar, &t.Gesamt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

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
			w.Header().Set(headerContentType, contentTypeJSON)
			httpresp.Write(w, []byte("[]"))
			return
		}

		ctx := r.Context()

		// Join only buecher_titel and buecher_exemplare.
		// The LEFT JOIN on ausleihen is filtered to active loans (rueckgabe_am IS NULL)
		// only to determine availability — no ausleihe column values are returned.
		args := []any{}

		// Was ohne Anmeldung sichtbar ist (kein Lernmittel, mindestens ein Exemplar im
		// Haus), steht genau einmal in repository.OeffentlichSichtbar — dieselbe Regel
		// wie im Flur-Monitor. Nur die öffentlichen Seiten filtern so: Verwaltung,
		// Inventur, Bestellwesen und die Klassensatz-Reservierung müssen die Bücher
		// weiter finden.
		var searchConditions = []string{repository.OeffentlichSichtbar("bt")}

		if q != "" {
			// $1 roh für die Volltextsuche, $2 mit maskierten LIKE-Jokern: Der Endpunkt ist
			// öffentlich, und ein nacktes "%" oder "_" in $1 machte aus dem Teilstring-
			// Vergleich einen Treffer auf den ganzen Bestand — LIMIT 50 hinter einem GROUP
			// BY über alle Titel, Exemplare und Ausleihen, 50-mal pro Sekunde und Adresse.
			args = append(args, q, maskiereLikeJoker(q))
			searchConditions = append(searchConditions, `(bt.search_vector @@ plainto_tsquery('german', $1)
			   OR bt.titel ILIKE '%' || $2 || '%'
			   OR bt.autor ILIKE '%' || $2 || '%'
			   OR bt.isbn ILIKE '%' || $2 || '%')`)
		}

		whereClause := ""
		if len(searchConditions) > 0 {
			whereClause = "WHERE " + strings.Join(searchConditions, " AND ")
		}

		query := fmt.Sprintf(`
			SELECT bt.id, bt.titel, COALESCE(bt.autor, ''), COALESCE(bt.isbn, ''),
			       COALESCE(bt.cover_url, ''),
			       COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
			       COUNT(e.id) FILTER (WHERE e.ist_ausgesondert = false AND e.bestellstatus IS NULL) AS gesamt
			FROM buecher_titel bt
			LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
			LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
			%s
			GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.cover_url
			ORDER BY bt.titel
			LIMIT 50
		`, whereClause)

		result, err := s.queryOpacTitel(ctx, query, args)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, result)
	}
}

// maskiereLikeJoker macht aus einer Nutzereingabe einen wörtlichen LIKE-Teilstring:
// Backslash, Prozent und Unterstrich verlieren ihre Sonderbedeutung (Postgres-Vorgabe
// ESCAPE '\\'). Dieselbe Schreibweise wie in repository/student_queries.go.
func maskiereLikeJoker(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
