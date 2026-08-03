package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// signaturBuecherLimit kappt die Regalansicht. Ohne Grenze liefert eine kurze
// Signatur ("BIB") den halben Bestand in einer Antwort — dieselbe Falle, in die
// /api/audit schon einmal gelaufen ist. Die Kappung wird gemeldet, nicht verschwiegen.
const signaturBuecherLimit = 500

// SignaturGruppe ist eine im Bestand tatsächlich vorkommende Signatur samt Umfang.
type SignaturGruppe struct {
	Signatur  string `json:"signatur"`
	Titel     int    `json:"titel"`
	Exemplare int    `json:"exemplare"`
}

// GetSignaturenHandler liefert die im Bestand vorkommenden Signaturen mit Umfang.
//
// Die Liste wird aus buecher_titel.signatur ABGELEITET, nicht aus einer Stammtabelle.
// Die frühere Tabelle `signatures` war eine zweite, ungepflegte Wahrheit: Sie kannte
// Namen, die an keinem Buch hingen, und kannte die Signaturen der Bücher nicht.
//
// @Summary      List signatures in stock
// @Description  Returns the signatures that actually occur on titles, with title and copy counts.
// @Tags         books
// @Produce      json
// @Success      200  {array}   SignaturGruppe
// @Failure      500  {object}  map[string]string
// @Router       /signaturen [get]
func (s *Server) GetSignaturenHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		rows, err := s.DB.Pool.Query(r.Context(), `
			SELECT btrim(t.signatur) AS signatur,
			       count(DISTINCT t.id) AS titel,
			       count(e.id) FILTER (WHERE e.ist_ausgesondert = false) AS exemplare
			FROM buecher_titel t
			LEFT JOIN buecher_exemplare e ON e.titel_id = t.id
			WHERE COALESCE(btrim(t.signatur), '') <> ''
			GROUP BY btrim(t.signatur)
			ORDER BY btrim(t.signatur)
		`)
		if err != nil {
			return apierrors.Internal("Signaturen konnten nicht geladen werden", err)
		}
		defer rows.Close()

		gruppen := []SignaturGruppe{}
		for rows.Next() {
			var g SignaturGruppe
			if err := rows.Scan(&g.Signatur, &g.Titel, &g.Exemplare); err != nil {
				return apierrors.Internal("Signaturzeile unlesbar", err)
			}
			gruppen = append(gruppen, g)
		}
		if err := rows.Err(); err != nil {
			return apierrors.Internal("Signaturen konnten nicht geladen werden", err)
		}

		RespondJSON(w, http.StatusOK, gruppen)
		return nil
	})
}

// SignaturBuch ist eine Zeile der Regalansicht.
type SignaturBuch struct {
	TitelID   string `json:"titel_id"`
	Signatur  string `json:"signatur"`
	Titel     string `json:"titel"`
	Autor     string `json:"autor"`
	ISBN      string `json:"isbn"`
	Exemplare int    `json:"exemplare"`
	Verliehen int    `json:"verliehen"`
}

// SignaturBuecherResponse ist die Regalansicht zu einer Signatur.
type SignaturBuecherResponse struct {
	Signatur string         `json:"signatur"`
	Buecher  []SignaturBuch `json:"buecher"`
	Gesamt   int            `json:"gesamt"`
	Gekappt  bool           `json:"gekappt"`
}

// GetSignaturBuecherHandler liefert die Titel unter einer Signatur — in Regalreihenfolge.
//
// Die Signatur wird als PRÄFIX verstanden. Das Prädikat kommt aus
// repository.SignaturPraefixBedingung — derselben Funktion, aus der sich auch der
// Inventur-Scope speist. Zwei eigene Kopien wären hier auseinandergelaufen, und der
// Unterschied fiele erst auf, wenn eine Inventur andere Bücher bucht, als die
// Regalansicht zeigt.
//
// @Summary      List books under a signature
// @Description  Returns titles whose signature matches the given prefix, in shelf order.
// @Tags         books
// @Produce      json
// @Param        signatur  query     string  true  "Signature prefix"
// @Success      200       {object}  SignaturBuecherResponse
// @Failure      400       {object}  map[string]string
// @Failure      500       {object}  map[string]string
// @Router       /signaturen/buecher [get]
func (s *Server) GetSignaturBuecherHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		signatur := strings.TrimSpace(r.URL.Query().Get("signatur"))
		if signatur == "" {
			return apierrors.BadRequest("signatur ist erforderlich", errors.New("leere signatur"))
		}

		rows, err := s.DB.Pool.Query(r.Context(), fmt.Sprintf(`
			SELECT t.id::text,
			       btrim(t.signatur),
			       t.titel,
			       COALESCE(t.autor, ''),
			       COALESCE(t.isbn, ''),
			       count(e.id) FILTER (WHERE e.ist_ausgesondert = false) AS exemplare,
			       count(e.id) FILTER (
			           WHERE e.ist_ausgesondert = false
			             AND EXISTS (SELECT 1 FROM ausleihen a
			                         WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
			       ) AS verliehen
			FROM buecher_titel t
			LEFT JOIN buecher_exemplare e ON e.titel_id = t.id
			WHERE %s
			GROUP BY t.id, t.signatur, t.titel, t.autor, t.isbn
			ORDER BY btrim(t.signatur), t.titel
			LIMIT $2
		`, repository.SignaturPraefixBedingung("t.signatur", 1)), signatur, signaturBuecherLimit+1)
		if err != nil {
			return apierrors.Internal("Bücher zur Signatur konnten nicht geladen werden", err)
		}
		defer rows.Close()

		buecher := []SignaturBuch{}
		for rows.Next() {
			var b SignaturBuch
			if err := rows.Scan(&b.TitelID, &b.Signatur, &b.Titel, &b.Autor, &b.ISBN, &b.Exemplare, &b.Verliehen); err != nil {
				return apierrors.Internal("Buchzeile unlesbar", err)
			}
			buecher = append(buecher, b)
		}
		if err := rows.Err(); err != nil {
			return apierrors.Internal("Bücher zur Signatur konnten nicht geladen werden", err)
		}

		// Eine Zeile mehr als das Limit geholt: Nur so lässt sich "es gibt noch mehr"
		// von "genau voll" unterscheiden, ohne eine zweite Zählquery zu fahren.
		gekappt := len(buecher) > signaturBuecherLimit
		if gekappt {
			buecher = buecher[:signaturBuecherLimit]
		}

		RespondJSON(w, http.StatusOK, SignaturBuecherResponse{
			Signatur: signatur,
			Buecher:  buecher,
			Gesamt:   len(buecher),
			Gekappt:  gekappt,
		})
		return nil
	})
}
