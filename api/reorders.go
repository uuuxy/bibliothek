package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
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
	// IstLernmittel: Vorschlag für den Topf der Bestellung (Warenkorb, mittel.js).
	// Mitgeliefert, obwohl die Vorgabe ?type=lmf ist — bei ?type=alle stünden sonst
	// Bücherei-Titel ohne Kennzeichen im Warenkorb.
	IstLernmittel bool `json:"ist_lernmittel"`
	// Auflagen: Gehört der Titel zu einem Buch mit mehreren Auflagen (Migration 148,
	// docs/OFFEN.md 4.18), ist die Zeile das Buch. Die beiden Bestandszahlen oben sind die
	// Summe, Titel, ISBN und Kennung oben die der neuesten Auflage — sie wird bestellt —, und
	// hier steht die Aufschlüsselung, die neueste zuerst. Ein Buch mit einer Auflage hat das
	// Feld nicht.
	Auflagen []ReorderAuflage `json:"auflagen,omitempty"`
}

// ReorderAuflage ist eine Auflage in der Aufschlüsselung einer Zeile, mit denselben zwei
// Zahlen, deren Summe die Zeile trägt, und mit ihrer ISBN: Wer in der Liste nach der alten
// Auflage sucht (oder sie scannt), soll die Zeile des Buchs finden.
type ReorderAuflage struct {
	ID                string `json:"id"`
	Auflage           string `json:"auflage"`
	ISBN              string `json:"isbn"`
	Erscheinungsjahr  int    `json:"erscheinungsjahr"`
	VerfuegbarBestand int    `json:"verfuegbarer_bestand"`
	GesamtBestand     int    `json:"gesamt_bestand"`
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
		reorders, err := s.reordersMitSettings(r.Context(), r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, reorders)
	}
}

// reordersMitSettings liefert den Bestellbedarf gemäß Konfiguration: Warnung aus →
// leere Liste; sonst gefiltert auf gesamt < Schwelle. Eine Quelle für Ansicht UND
// PDF-Export, damit beide nie auseinanderlaufen.
func (s *Server) reordersMitSettings(ctx context.Context, r *http.Request) ([]ReorderTitle, error) {
	settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.BestellbedarfWarnungAktiv {
		return []ReorderTitle{}, nil
	}
	return s.queryReorders(ctx, reorderFilter(r), settings.BestellbedarfSchwelle)
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

// queryReorders liefert die Titel, deren BESITZ (gesamt, nicht ausgesondert) unter der
// konfigurierten Bestellbedarf-Schwelle liegt — der größte Fehlbestand zuerst. Bewusst
// gesamt statt verfügbar: sonst würde jeder verliehene Lernmittel-Klassensatz (im
// Schuljahr der Normalfall) als Bestellbedarf gemeldet. Da "gesamt" die als "bestellt"
// markierten Platzhalter mitzählt, ist die Liste zugleich bereits-bestellt-bewusst.
// Die Schwelle ersetzt den früheren pro-Titel-Meldebestand (der pauschal auf 5 stand und
// fast jeden Titel meldete); t.meldebestand wird nur noch informativ mitgeliefert.
//
// Gezählt wird am Buch (Migration 148, docs/OFFEN.md 4.18, entschieden am 25.09.2026):
// Auflagen mit derselben werk_id bilden eine Zeile, ihre Summe steht gegen die Schwelle,
// und gezeigt wird die neueste Auflage (repository.SQLNeuesteAuflageZuerst) — die wird
// bestellt. Ein Titel ohne Werk ist über COALESCE(werk_id, id) sein eigenes Buch; für ihn
// ändert sich nichts. Der Filter nach Topf (typeFilter) gilt je Auflage, bevor gezählt wird.
func (s *Server) queryReorders(ctx context.Context, typeFilter string, schwelle int) ([]ReorderTitle, error) {
	neueste := repository.SQLNeuesteAuflageZuerst("t")
	// Ein LATERAL je Titel liefert beide Bestandszahlen in einem Durchgang; danach wird je
	// Buch summiert, und DISTINCT ON wählt die neueste Auflage je Buch in einem Durchgang.
	// Nicht als zweites LATERAL je Zeile: Das lief für jede Zeile einmal über alle Titel —
	// gemessen am 25.09.2026 lokal über 6.048 Titel (?type=alle, Schwelle 3) 810–825 ms gegen
	// 25–35 ms der Abfrage je Titel; so sind es 66–71 ms bei denselben 5.208 Zeilen.
	query := fmt.Sprintf(`
		WITH titel AS (
			SELECT t.id, COALESCE(t.werk_id, t.id) AS buch, t.titel, t.autor, t.isbn, t.verlag,
			       t.signatur, t.erscheinungsjahr, t.erstellt_am, t.auflage, t.cover_url,
			       t.meldebestand, t.ist_lernmittel, v.verfuegbar, v.gesamt
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
			WHERE true %s
		),
		buch AS (
			SELECT t.buch, sum(t.verfuegbar)::int AS verfuegbar, sum(t.gesamt)::int AS gesamt,
			       count(*) AS anzahl,
			       json_agg(json_build_object('id', t.id, 'auflage', coalesce(t.auflage, ''), 'isbn', coalesce(t.isbn, ''),
			           'erscheinungsjahr', coalesce(t.erscheinungsjahr, 0),
			           'verfuegbarer_bestand', t.verfuegbar, 'gesamt_bestand', t.gesamt)
			           ORDER BY %s) AS auflagen
			FROM titel t
			GROUP BY t.buch
			HAVING sum(t.gesamt) < $1
		),
		neueste AS (
			SELECT DISTINCT ON (t.buch) t.* FROM titel t ORDER BY t.buch, %s
		)
		SELECT n.id, n.titel, coalesce(n.autor, ''), coalesce(n.isbn, ''), coalesce(n.verlag, ''),
		       coalesce(n.signatur, ''), coalesce(n.erscheinungsjahr, 0),
		       COALESCE(NULLIF(n.cover_url, ''), CASE WHEN n.isbn IS NOT NULL AND n.isbn != ''
		           THEN 'https://portal.dnb.de/opac/mvb/cover?isbn=' || replace(n.isbn, '-', '') ELSE '' END),
		       n.meldebestand, b.verfuegbar, b.gesamt, n.ist_lernmittel,
		       CASE WHEN b.anzahl > 1 THEN b.auflagen END
		FROM buch b
		JOIN neueste n ON n.buch = b.buch
		ORDER BY ($1 - b.gesamt) DESC, n.titel ASC
	`, typeFilter, neueste, neueste)

	rows, err := s.DB.Pool.Query(ctx, query, schwelle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ReorderTitle, 0)
	for rows.Next() {
		var t ReorderTitle
		var auflagen []byte
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag, &t.Signatur,
			&t.Erscheinungsjahr, &t.CoverURL, &t.Meldebestand, &t.VerfuegbarBestand,
			&t.GesamtBestand, &t.IstLernmittel, &auflagen); err != nil {
			return nil, err
		}
		if len(auflagen) > 0 {
			if err := json.Unmarshal(auflagen, &t.Auflagen); err != nil {
				return nil, fmt.Errorf("auflagen der zeile %s: %w", t.ID, err)
			}
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
