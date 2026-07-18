package api

import (
	"context"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
)

// ReorderTitle ist ein Titel, dessen verfügbarer Bestand unter den Meldebestand
// gefallen ist.
type ReorderTitle struct {
	ID     string `json:"id"`
	Titel  string `json:"titel"`
	Autor  string `json:"autor"`
	ISBN   string `json:"isbn"`
	Verlag string `json:"verlag"`
	// Signatur und Erscheinungsjahr helfen beim Nachbestellen, die richtige Ausgabe zu
	// treffen — bei Lernmitteln unterscheiden sich Jahrgänge oft nur darin.
	Signatur         string `json:"signatur,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
	CoverURL         string `json:"cover_url,omitempty"`
	Meldebestand     int    `json:"meldebestand"`
	// VerfuegbarBestand = aktuell im Regal (nicht ausgeliehen, nicht ausgesondert).
	// Reine Anzeige-Kontextzahl — NICHT die Nachbestell-Schwelle (siehe GesamtBestand).
	VerfuegbarBestand int `json:"verfuegbarer_bestand"`
	// GesamtBestand = alle nicht ausgesonderten Exemplare (inkl. der als "bestellt"
	// markierten Platzhalter). DAS ist die Nachbestell-Schwelle: Meldebestand meint die
	// Zahl der Exemplare, die man BESITZEN will — nicht, wie viele gerade im Regal stehen.
	// Bei Lernmitteln ist ein Klassensatz das ganze Schuljahr verliehen — "0 verfügbar"
	// bei 30 vorhandenen ist KEIN Bestellgrund, deshalb triggert gesamt < meldebestand.
	GesamtBestand int `json:"gesamt_bestand"`
}

// GetReordersHandler liefert den Bestellbedarf.
//
// Default ist der LMF-Bestand (Lernmittel): Nachbestellt werden praktisch nur
// Lernmittel-Klassensätze; im Freihandbestand steht meist ein einzelnes Prüf- oder
// Leseexemplar, das bewusst ein Einzelstück bleibt. Ohne diese Vorauswahl bestand die
// Liste zu ~99% aus Titeln, die niemand nachbestellen will — bei realem Bestand
// tausende Einträge, die die Ansicht unbenutzbar machten.
//
// ?type=freihand oder ?type=alle bleiben möglich; für Einzelfälle ausserhalb der
// Lernmittel gibt es ausserdem die Titelsuche im Bestellworkspace.
func (s *Server) GetReordersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reorders, err := s.queryReorders(r.Context(), reorderFilter(r))
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, reorders)
	}
}

// reorderFilter liest ?type= und fällt auf den LMF-Bestand zurück (siehe
// GetReordersHandler). Das SQL-Fragment ist serverkontrolliert, der Parameter wählt
// nur zwischen festen Varianten.
func reorderFilter(r *http.Request) string {
	typ := r.URL.Query().Get("type")
	if typ == "" {
		typ = "lmf"
	}
	fragment, _ := resolveBestandsFilter(typ)
	return fragment
}

// queryReorders liefert die Titel, deren BESITZ (gesamt, nicht ausgesondert) unter dem
// Meldebestand liegt — der größte Fehlbestand zuerst. Bewusst gesamt statt verfügbar:
// sonst würde jeder verliehene Lernmittel-Klassensatz (im Schuljahr der Normalfall) als
// Bestellbedarf gemeldet. Da "gesamt" die als "bestellt" markierten Platzhalter mitzählt,
// ist die Liste zugleich bereits-bestellt-bewusst (keine Doppelbestellung).
func (s *Server) queryReorders(ctx context.Context, typeFilter string) ([]ReorderTitle, error) {
	// Ein LATERAL je Titel liefert beide Bestandszahlen in einem Durchgang.
	query := fmt.Sprintf(`
		SELECT t.id, t.titel, coalesce(t.autor, ''), coalesce(t.isbn, ''), coalesce(t.verlag, ''),
		       coalesce(t.signatur, ''), coalesce(t.erscheinungsjahr, 0),
		       COALESCE(NULLIF(t.cover_url, ''), CASE WHEN t.isbn IS NOT NULL AND t.isbn != ''
		           THEN 'https://portal.dnb.de/opac/mvb/cover?isbn=' || replace(t.isbn, '-', '') ELSE '' END),
		       t.meldebestand, v.verfuegbar, v.gesamt
		FROM buecher_titel t
		JOIN LATERAL (
			SELECT
				COUNT(*) FILTER (
					WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false
					  AND NOT EXISTS (SELECT 1 FROM ausleihen a
					                  WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
				)::int AS verfuegbar,
				COUNT(*) FILTER (WHERE e.ist_ausgesondert = false)::int AS gesamt
			FROM buecher_exemplare e
			WHERE e.titel_id = t.id
		) v ON true
		WHERE v.gesamt < t.meldebestand %s
		ORDER BY (t.meldebestand - v.gesamt) DESC, t.titel ASC
	`, typeFilter)

	rows, err := s.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ReorderTitle, 0)
	for rows.Next() {
		var t ReorderTitle
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag, &t.Signatur,
			&t.Erscheinungsjahr, &t.CoverURL, &t.Meldebestand, &t.VerfuegbarBestand,
			&t.GesamtBestand); err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
