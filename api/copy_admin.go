package api

// copy_admin.go — Handlers for physical book copy and title administration:
// damage notes, deletion, decommissioning and copy listing.
// Part of the admin layer; authentication/authorization is enforced in router.go.

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// DeleteCopyHandler removes a physical copy from circulation.
// @Summary      Delete physical book copy
// @Description  Deletes a specific physical book copy by its ID from the library catalog and registers the deletion in the audit trail.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book copy ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/exemplare/{id} [delete]
func (s *Server) DeleteCopyHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}

		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing copy ID parameter"))
			return
		}

		ctx := r.Context()

		err := auditRepo.DeleteCopy(ctx, id, claims.UserID)
		if err != nil {
			// Sentinels statt Textvergleich: Der frühere Vergleich („Exemplar ist
			// aktuell noch verliehen!") traf den Repo-Text nie — ein verliehenes Buch
			// endete als 500, und der Sanitizer verschluckte die Auskunft.
			if errors.Is(err, repository.ErrExemplarNochVerliehen) {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
				return
			}
			if errors.Is(err, repository.ErrExemplarNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondSuccess(w)
	}
}

// DeleteTitleHandler deletes a book title and all its physical copies from the database, creating an audit log.
// @Summary      Delete book title
// @Description  Deletes a specific book title and all associated physical copies, registering the deletion in the audit trail.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book title ID (UUID)"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/titel/{id} [delete]
func (s *Server) DeleteTitleHandler(auditRepo repository.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("missing session information"))
			return
		}
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		err := auditRepo.DeleteTitle(ctx, id, claims.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrTitelHatAktiveAusleihen) {
				apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			} else if errors.Is(err, repository.ErrTitelNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
			} else {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			}
			return
		}

		RespondSuccess(w)
	}
}

// GetTitleCopiesHandler lists all physical copies belonging to a book title.
// @Summary      List copies for a title
// @Description  Retrieves all physical book copies associated with a given title ID, including availability status.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Book title ID (UUID)"
// @Success      200  {array}   object
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /buecher/titel/{id}/exemplare [get]
func (s *Server) GetTitleCopiesHandler(bescheidRepo repository.BescheidRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		query := `
			SELECT e.id, e.barcode_id, coalesce(e.zustand_notiz, ''), e.ist_ausleihbar, e.ist_ausgesondert,
			       coalesce(e.zustand_abwertung_prozent, 0),
			       NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL) AS ist_verfuegbar
			FROM buecher_exemplare e
			WHERE e.titel_id = $1
			ORDER BY e.ist_ausgesondert ASC, e.barcode_id
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		// CopyResponse is the per-copy DTO returned by this handler.
		type CopyResponse struct {
			ID           string `json:"id"`
			BarcodeID    string `json:"barcode_id"`
			ZustandNotiz string `json:"zustand_notiz"`
			// ZustandAbwertungProzent ist der erfasste Beschädigungsgrad (Migration 127).
			// Er steht hier, weil die Buchakte ihn anzeigt UND ihr Status-Editor ihn
			// zurückschickt — ohne den Lesewert wäre jedes Speichern eine Neueingabe.
			ZustandAbwertungProzent int  `json:"zustand_abwertung_prozent"`
			IstAusleihbar           bool `json:"ist_ausleihbar"`
			IstAusgesondert         bool `json:"ist_ausgesondert"`
			IstVerfuegbar           bool `json:"ist_verfuegbar"`
			// Ersatzwert und Herleitung: was ein Ersatz für DIESES Exemplar heute kostet
			// (OFFEN.md 9.8, Stufe 2b). Gerechnet wird mit ersatzwertVorschlagAus —
			// derselben Funktion wie im Melde-Dialog, damit die Buchakte und die
			// Forderung nie zwei Zahlen für dasselbe Buch zeigen.
			Ersatzwert           float64 `json:"ersatzwert"`
			ErsatzwertHerleitung string  `json:"ersatzwert_herleitung"`
			// ErsatzwertBekannt: Liegt dem Betrag ein Preis zugrunde? Ohne dieses Feld
			// müsste die Karte die Antwort aus dem Herleitungssatz lesen — und 0,00 € heißt
			// zweierlei (Totalschaden oder kein Preis erfasst).
			ErsatzwertBekannt bool `json:"ersatzwert_bekannt"`
		}

		copies := []CopyResponse{}
		for rows.Next() {
			var cp CopyResponse
			if err := rows.Scan(&cp.ID, &cp.BarcodeID, &cp.ZustandNotiz, &cp.IstAusleihbar,
				&cp.IstAusgesondert, &cp.ZustandAbwertungProzent, &cp.IstVerfuegbar); err == nil {
				copies = append(copies, cp)
			}
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Der Ersatzwert je Exemplar: EINE Abfrage für den ganzen Titel, dann die
		// Staffel im Speicher. Ein Aufruf je Karte wäre bei einem Klassensatz mit
		// 30 Bänden 30 Abfragen für eine Seite.
		//
		// Scheitert das, bleibt die Liste stehen und die Werte fehlen: Der Ersatzwert
		// ist eine Auskunft, kein Grund, die Exemplar-Liste zu verweigern.
		if groessen, err := bescheidRepo.GroessenFuerTitel(ctx, id); err == nil {
			quelle := s.preisquelle(ctx)
			for i := range copies {
				g, da := groessen[copies[i].ID]
				if !da {
					continue
				}
				v := ersatzwertVorschlagAus(g, quelle)
				copies[i].Ersatzwert = v.Betrag
				copies[i].ErsatzwertHerleitung = v.Herleitung
				copies[i].ErsatzwertBekannt = v.Bekannt
			}
		}

		RespondJSON(w, http.StatusOK, copies)
	}
}

// TitleBorrower represents a student who is currently borrowing a copy of a title.
type TitleBorrower struct {
	Vorname         string    `json:"schueler_name"`
	Nachname        string    `json:"schueler_nachname"`
	Klasse          string    `json:"klasse"`
	SchuelerBarcode string    `json:"schueler_barcode"`
	ExemplarBarcode string    `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist  time.Time `json:"rueckgabe_frist"`
	// IstDauerleihe: Ausleihe an jemanden, der kein Schüler ist (`ausleihen.ist_handapparat`).
	// Ohne das Merkmal färbten die Ausleiher-Liste und ihr Druck die Frist eines Kollegen
	// nach einem Jahr rot — die Akte wusste es seit dem 16.09.2026 besser (OFFEN.md 5.18).
	IstDauerleihe bool `json:"ist_dauerleihe"`
}

// GetTitleBorrowersHandler lists all active borrowers for a book title.
// @Router       /buecher/titel/{id}/ausleiher [get]
func (s *Server) GetTitleBorrowersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
			return
		}

		ctx := r.Context()

		// LEFT JOIN auf BEIDE Ausleiher-Arten: Eine Ausleihe an eine Lehrkraft trägt keine
		// schueler_id, und der frühere INNER JOIN auf schueler ließ sie damit verschwinden
		// — der Reiter zeigte weniger Ausleiher, als der Titel hat, und wer das Exemplar
		// suchte, suchte im Regal. Die Klasse 'Lehrer' ist dieselbe Auskunft, die die
		// Titel-Historie seit dem 22.08.2026 gibt; der Klassenfilter des Reiters liest
		// genau dieses Feld. COALESCE auf 'Anonym' deckt die getrennte Ausleihe ab —
		// laufende trifft die Lesehistorie-Befristung zwar nicht, aber die Antwort soll
		// auch dann keinen leeren Namen tragen.
		query := `
			SELECT
			  COALESCE(l.vorname, 'Anonym') AS vorname,
			  COALESCE(l.nachname, '') AS nachname,
			  -- „Lehrer" steht jetzt an der Art des Lesers statt an der Spalte, in der
			  -- er stand (Migration 125). Der Klassenfilter des Reiters liest dieses Feld.
			  CASE WHEN l.art IS NOT NULL AND l.art <> 'schueler' THEN 'Lehrer'
			       ELSE COALESCE(l.klasse, '') END AS klasse,
			  COALESCE(l.barcode_id, '') AS ausleiher_barcode,
			  e.barcode_id, a.ausgeliehen_am, a.rueckgabe_frist, a.ist_handapparat
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			LEFT JOIN leser l ON a.schueler_id = l.id
			WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
			ORDER BY a.rueckgabe_frist ASC
		`
		rows, err := s.DB.Pool.Query(ctx, query, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		borrowers := []TitleBorrower{}
		for rows.Next() {
			var b TitleBorrower
			if err := rows.Scan(&b.Vorname, &b.Nachname, &b.Klasse, &b.SchuelerBarcode, &b.ExemplarBarcode, &b.AusgeliehenAm, &b.RueckgabeFrist, &b.IstDauerleihe); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			borrowers = append(borrowers, b)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, borrowers)
	}
}

// TitleHistory represents a historical loan for a copy of a title.
type TitleHistory struct {
	Vorname         string     `json:"schueler_name"`
	Nachname        string     `json:"schueler_nachname"`
	Klasse          string     `json:"klasse"`
	ExemplarBarcode string     `json:"exemplar_barcode"`
	AusgeliehenAm   time.Time  `json:"ausgeliehen_am"`
	RueckgabeAm     *time.Time `json:"rueckgabe_am"` // Can be null if still borrowed
}

// GetTitleHistoryHandler lists all historical loans for a book title.
// @Router       /buecher/titel/{id}/historie [get]
func (s *Server) GetTitleHistoryHandler() http.HandlerFunc {
	return s.handleGetTitleHistory
}

// handleGetTitleHistory liefert die letzten 200 Ausleih-Vorgänge eines Titels. Seit der
// Lesehistorie-Befristung (jobs/cron_dsgvo_lesehistorie.go) trägt ein Großteil davon keine
// schueler_id mehr: Name → "Anonym", Klasse leer. "Lehrer" gilt nur, wenn wirklich ein
// Benutzer-Konto ausgeliehen hat — vorher machte COALESCE(s.klasse, 'Lehrer') aus jeder
// getrennten Schüler-Ausleihe eine Lehrer-Ausleihe. Als
// Top-Level-Methode ausgelagert (nicht Inline-Closure), damit die Scan-Schleife nicht
// zusätzlich als Verschachtelung zählt (S3776).
func (s *Server) handleGetTitleHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing title ID parameter"))
		return
	}

	ctx := r.Context()

	query := `
			SELECT 
			  l.vorname AS vorname,
			  l.nachname AS nachname,
			  CASE WHEN l.art IS NOT NULL AND l.art <> 'schueler' THEN 'Lehrer'
			       ELSE l.klasse END AS klasse,
			  e.barcode_id, a.ausgeliehen_am, a.rueckgabe_am
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			LEFT JOIN leser l ON a.schueler_id = l.id
			WHERE e.titel_id = $1
			ORDER BY a.ausgeliehen_am DESC
			LIMIT 200
		`
	rows, err := s.DB.Pool.Query(ctx, query, id)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	history := []TitleHistory{}
	for rows.Next() {
		var h TitleHistory
		var vorname, nachname, klasse *string
		if err := rows.Scan(&vorname, &nachname, &klasse, &h.ExemplarBarcode, &h.AusgeliehenAm, &h.RueckgabeAm); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		h.Vorname = stringOrDefault(vorname, "Anonym")
		h.Nachname = stringOrDefault(nachname, "")
		h.Klasse = stringOrDefault(klasse, "")
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	RespondJSON(w, http.StatusOK, history)
}

// stringOrDefault liefert den Wert von s, oder fallback wenn s nil ist.
func stringOrDefault(s *string, fallback string) string {
	if s != nil {
		return *s
	}
	return fallback
}
