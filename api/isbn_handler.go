package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/inventur"

	"github.com/jackc/pgx/v5"
)

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
		if !apierrors.DecodeJSONRequest(w, r, &req) {
			return
		}
		// Normalise: strip dashes and spaces
		req.ISBN = strings.TrimSpace(strings.NewReplacer("-", "", " ", "").Replace(req.ISBN))
		if req.ISBN == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("isbn fehlt"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		// 1. Check whether the title is already in the local catalog.
		var resp ISBNLookupResponse
		resp.ISBN = req.ISBN
		err := s.DB.Pool.QueryRow(ctx, `
			SELECT id, titel, coalesce(autor,''), coalesce(verlag,''), coalesce(cover_url,'')
			FROM buecher_titel WHERE isbn = $1 LIMIT 1
		`, req.ISBN).Scan(&resp.TitelID, &resp.Titel, &resp.Autor, &resp.Verlag, &resp.CoverURL)
		if err == nil {
			resp.Exists = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// 2. Not yet in catalog – fetch metadata from DNB / Google / OpenLibrary.
		meta, err := metaClient.SucheNachISBN(ctx, req.ISBN)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusNotFound, fmt.Errorf("keine Metadaten für ISBN %s gefunden", req.ISBN))
			return
		}

		// 3. Insert new title; use ON CONFLICT as safety net for concurrent inserts.
		var newID, newTitel, newAutor, newCoverURL string
		err = s.DB.Pool.QueryRow(ctx, `
			INSERT INTO buecher_titel (titel, autor, isbn, cover_url)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (isbn) DO UPDATE
				SET titel      = EXCLUDED.titel,
				    autor      = EXCLUDED.autor,
				    cover_url  = EXCLUDED.cover_url,
				    aktualisiert_am = CURRENT_TIMESTAMP
			RETURNING id, titel, coalesce(autor,''), coalesce(cover_url,'')
		`, meta.Titel, meta.Autor, req.ISBN, meta.CoverURL).Scan(&newID, &newTitel, &newAutor, &newCoverURL)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		resp.Exists = false
		resp.TitelID = newID
		resp.Titel = newTitel
		resp.Autor = newAutor
		resp.CoverURL = newCoverURL

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
