package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/internal/service"
	"bibliothek/repository"
)

// UnifiedSearchResult defines the combined payload for the fuzzy search.
// StudentsTotal/BooksTotal nennen die Gesamttrefferzahl vor dem Limit, damit die
// Omnibox eine gekürzte Liste als gekürzt ausweisen kann.
type UnifiedSearchResult struct {
	Students      []SchuelerKiosk        `json:"students"`
	Books         []repository.BookTitle `json:"books"`
	StudentsTotal int                    `json:"students_total"`
	BooksTotal    int                    `json:"books_total"`
	// Treffer: exakte Auflösung eines Scans (Exemplar-Barcode, Littera-EAN, Ausweis) —
	// die globale Suchleiste springt damit direkt zur Akte (03.09.2026). nil = kein Scan.
	Treffer *service.ScanTreffer `json:"treffer,omitempty"`
}

// SearchHandler provides a unified fuzzy search for students and books without requiring prefixes.
func (s *Server) SearchHandler(studentRepo repository.StudentRepository, bookRepo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("search query 'q' cannot be empty"))
			return
		}

		ctx := r.Context()

		limit := 10

		// Using channels or just sequential is fine, but sequential is easier and perfectly fast enough
		// for a local postgres database with limits.
		students, studentsTotal, err := studentRepo.SearchStudentsFuzzy(ctx, query, limit)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		books, booksTotal, err := bookRepo.SearchTitlesFuzzy(ctx, query, limit)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		treffer, err := service.ErkenneScan(ctx, bookRepo, studentRepo, query)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if books == nil {
			books = []repository.BookTitle{}
		}

		// Reduktion auf die Theken-Sicht: /api/search hängt an perform_actions
		// (auch Helfer-Rolle) — Adresse, Eltern-Mail und Geburtsdatum gehören
		// nicht in diese Antwort (bewertung/sicherheitsbefund-kiosk-suche.md).
		// zuKioskSchuelern liefert nie nil, damit das JSON ein leeres Array trägt.
		result := UnifiedSearchResult{
			Students:      zuKioskSchuelern(students),
			Books:         books,
			StudentsTotal: studentsTotal,
			BooksTotal:    booksTotal,
			Treffer:       treffer,
		}

		RespondJSON(w, http.StatusOK, result)
	}
}

// TitelSucheResult ist die Antwort von GET /api/buecher/titel/suche: Titel, sonst nichts.
type TitelSucheResult struct {
	Books      []repository.BookTitle `json:"books"`
	BooksTotal int                    `json:"books_total"`
}

// TitelSucheHandler ist die Titel-Hälfte von SearchHandler als eigene Tür (view_books).
//
// Die Etiketten-Titelsuche im Druck-Center rief bis 05.09.2026 GET /api/search und las aus
// der Antwort nur `books` — die Schüler-Kiosk-Daten darin zeigte sie nie, holte sie aber
// mit; und weil /api/search am Theken-Recht perform_actions hängt, meldete der
// Druckbildschirm ohne dieses Recht nur noch „Suche nicht möglich". Die Inventur-Liste
// GET /api/books wäre keine Alternative: Sie sucht per ILIKE über Titel/Autor/ISBN, ohne
// Signatur und ohne Umlaut-Normalisierung — SearchTitlesFuzzy ist die Suchgüte der Theke,
// und die bleibt hier erhalten.
func (s *Server) TitelSucheHandler(bookRepo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("search query 'q' cannot be empty"))
			return
		}
		books, total, err := bookRepo.SearchTitlesFuzzy(r.Context(), query, 10)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if books == nil {
			books = []repository.BookTitle{}
		}
		RespondJSON(w, http.StatusOK, TitelSucheResult{Books: books, BooksTotal: total})
	}
}
