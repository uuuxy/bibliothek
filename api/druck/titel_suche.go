package druck

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// TitelSucheResult ist die Antwort von GET /api/buecher/titel/suche: Titel, sonst nichts.
type TitelSucheResult struct {
	Books      []repository.Titel `json:"books"`
	BooksTotal int                `json:"books_total"`
}

// TitelSucheHandler ist die Titel-Hälfte von SearchHandler als eigene Tür (view_books).
//
// Die Etiketten-Titelsuche im Druck-Center rief bis 05.09.2026 GET /api/search und las aus
// der Antwort nur `books` — die Schüler-Kiosk-Daten darin zeigte sie nie, holte sie aber
// mit; und weil /api/search am Theken-Recht perform_actions hängt, meldete der
// Druckbildschirm ohne dieses Recht nur noch „Suche nicht möglich". Die Inventur-Liste
// GET /api/books wäre keine Alternative: Sie sucht per ILIKE über Titel/Autor/ISBN, ohne
// Signatur und ohne Umlaut-Normalisierung — KatalogAbfragen.FindeTitelUnscharf ist die Suchgüte der Theke,
// und die bleibt hier erhalten.
func (s *Tueren) TitelSucheHandler(bookRepo repository.KatalogAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("search query 'q' cannot be empty"))
			return
		}
		books, total, err := bookRepo.FindeTitelUnscharf(r.Context(), query, 10)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if books == nil {
			books = []repository.Titel{}
		}
		tuer.RespondJSON(w, http.StatusOK, TitelSucheResult{Books: books, BooksTotal: total})
	}
}
