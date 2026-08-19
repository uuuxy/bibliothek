package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/inventur"

	"github.com/jackc/pgx/v5"
)

// parseErscheinungsjahr parst ein plausibles Erscheinungsjahr (1001–2099) aus rohem Text.
func parseErscheinungsjahr(raw string) *int {
	if raw == "" {
		return nil
	}
	var j int
	if _, err := fmt.Sscanf(raw, "%d", &j); err == nil && j > 1000 && j < 2100 {
		return &j
	}
	return nil
}

// findeLokalenTitel sucht einen Titel im lokalen Katalog anhand der (entstrichenen) ISBN.
// Rückgabe (nil, nil) bedeutet: nicht im Katalog vorhanden.
//
// Liefert die vorhandene Signatur mit — die Bestellung übernimmt sie unverändert und
// fragt dafür nie erneut DNB/eine Kategorisierung ab (siehe upsertTitelAusMetadaten,
// die dieser Funktion vorgeschaltet ist und nur bei NICHT gefundenem Titel läuft).
func (s *Server) findeLokalenTitel(ctx context.Context, isbn string) (*ISBNLookupResponse, error) {
	resp := ISBNLookupResponse{ISBN: isbn}
	err := s.DB.Pool.QueryRow(ctx, `
		SELECT id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,''), coalesce(signatur,'')
		FROM buecher_titel WHERE replace(isbn, '-', '') = $1 LIMIT 1
	`, isbn).Scan(&resp.TitelID, &resp.Titel, &resp.Autor, &resp.Verlag, &resp.CoverURL, &resp.Signatur)
	if err == nil {
		resp.Exists = true
		return &resp, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

// signaturVorschlagAusMetadaten baut den Signatur-Vorschlag "BIB {Kategorie}" aus der
// DNB-Genre-/Altersheuristik (dieselbe Ableitung wie im Buchformular, IsbnFeld.svelte).
// Leer, wenn die Heuristik keine Kategorie ermitteln konnte — dann bleibt das Feld leer
// und muss wie im Buchformular manuell gesetzt werden, statt einen sinnlosen "BIB "-Wert
// zu erzeugen.
func signaturVorschlagAusMetadaten(meta *inventur.MetadatenErgebnis) string {
	if meta.BibKategorie == "" {
		return ""
	}
	return "BIB " + meta.BibKategorie
}

// upsertTitelAusMetadaten legt einen neuen Titel aus den Nachschlage-Metadaten an
// (ON CONFLICT (isbn) als Schutz gegen parallele Inserts) und liefert die Antwort.
//
// Läuft nur, wenn findeLokalenTitel zuvor NICHTS gefunden hat — ein bestehender Titel
// erreicht diese Funktion nie, das ON CONFLICT ist reine Race-Condition-Absicherung
// gegen einen zeitgleichen zweiten Import derselben ISBN. Deshalb dürfen signatur/subject
// hier ungeschützt geschrieben werden: Es gibt nichts Bestehendes, das verloren gehen
// könnte (anders als bei BulkUpsertBookTitles, siehe [[upsert-blanking-bugklasse]]).
func (s *Server) upsertTitelAusMetadaten(ctx context.Context, isbn string, meta *inventur.MetadatenErgebnis) (ISBNLookupResponse, error) {
	jahrInt := parseErscheinungsjahr(meta.Jahr)
	signatur := signaturVorschlagAusMetadaten(meta)

	// subject ist FK auf die Systematik (Migration 078): das Fach aus der
	// Titel-Heuristik erst registrieren, die kanonische Schreibweise schreiben.
	kanonisch, err := inventur.StelleFaecherSicher(ctx, s.DB.Pool, []string{meta.Fach})
	if err != nil {
		return ISBNLookupResponse{}, err
	}

	resp := ISBNLookupResponse{ISBN: isbn}
	err = s.DB.Pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, cover_url, signatur, subject)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''))
		ON CONFLICT (isbn) DO UPDATE
			SET titel      = EXCLUDED.titel,
			    autor      = EXCLUDED.autor,
			    verlag     = EXCLUDED.verlag,
			    erscheinungsjahr = EXCLUDED.erscheinungsjahr,
			    cover_url  = COALESCE(NULLIF(EXCLUDED.cover_url, ''), buecher_titel.cover_url),
			    aktualisiert_am = CURRENT_TIMESTAMP
		RETURNING id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,''), coalesce(signatur,'')
	`, meta.Titel, meta.Autor, isbn, meta.Verlag, jahrInt, meta.CoverURL, signatur, kanonisch[meta.Fach]).
		Scan(&resp.TitelID, &resp.Titel, &resp.Autor, &resp.Verlag, &resp.CoverURL, &resp.Signatur)
	if err != nil {
		return ISBNLookupResponse{}, err
	}
	resp.Exists = false
	return resp, nil
}

// ISBNLookupResponse is the result of a live ISBN metadata query.
// exists=true means the title is already in the catalog and has a stable titel_id.
type ISBNLookupResponse struct {
	Exists   bool   `json:"exists"`
	TitelID  string `json:"titel_id"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	ISBN     string `json:"isbn"`
	Verlag   string `json:"verlag,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// Signatur ist bei exists=true die BEREITS VORHANDENE Regalsignatur (unverändert
	// übernommen) und bei exists=false ein VORSCHLAG aus der DNB-Kategorisierung — in
	// beiden Fällen im Bestellkorb vor dem Bestellen editierbar, siehe
	// PUT /api/buecher/titel/{id}/signatur.
	Signatur string `json:"signatur,omitempty"`
}

// ISBNZuTitelHandler handles POST /api/buecher/aus-isbn.
// It receives an ISBN, checks the local catalog, and—if the title is not
// yet catalogued—fetches metadata from DNB / Google Books / OpenLibrary and
// creates a new buecher_titel record. The response always contains a titel_id
// that the order workspace can add to the cart immediately.
func (s *Server) ISBNZuTitelHandler() http.HandlerFunc {
	metaClient := inventur.NeuerMetadatenClient()
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ISBN string `json:"isbn"`
		}
		if !DecodeAndValidate(w, r, &req) {
			return
		}
		// Normalise: strip dashes and spaces
		req.ISBN = strings.TrimSpace(strings.NewReplacer("-", "", " ", "").Replace(req.ISBN))
		if req.ISBN == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("isbn fehlt"))
			return
		}

		ctx := r.Context()

		// 1. Check whether the title is already in the local catalog.
		lokal, err := s.findeLokalenTitel(ctx, req.ISBN)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if lokal != nil {
			RespondJSON(w, http.StatusOK, *lokal)
			return
		}

		// 2. Not yet in catalog – fetch metadata from DNB / Google / OpenLibrary.
		meta, err := metaClient.SucheNachISBN(ctx, req.ISBN)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine Metadaten für ISBN %s gefunden", req.ISBN))
			return
		}

		// 3. Insert new title; use ON CONFLICT as safety net for concurrent inserts.
		resp, err := s.upsertTitelAusMetadaten(ctx, req.ISBN, meta)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}
