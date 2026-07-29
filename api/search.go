package api

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// UnifiedSearchResult defines the combined payload for the fuzzy search.
// StudentsTotal/BooksTotal nennen die Gesamttrefferzahl vor dem Limit, damit die
// Omnibox eine gekürzte Liste als gekürzt ausweisen kann.
type UnifiedSearchResult struct {
	Students      []repository.Student   `json:"students"`
	Books         []repository.BookTitle `json:"books"`
	StudentsTotal int                    `json:"students_total"`
	BooksTotal    int                    `json:"books_total"`
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

		// Ensure we don't return nil slices in JSON
		if students == nil {
			students = []repository.Student{}
		}
		if books == nil {
			books = []repository.BookTitle{}
		}

		result := UnifiedSearchResult{
			Students:      students,
			Books:         books,
			StudentsTotal: studentsTotal,
			BooksTotal:    booksTotal,
		}

		RespondJSON(w, http.StatusOK, result)
	}
}
