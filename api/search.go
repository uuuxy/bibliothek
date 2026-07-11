package api

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// UnifiedSearchResult defines the combined payload for the fuzzy search.
type UnifiedSearchResult struct {
	Students []repository.Student   `json:"students"`
	Books    []repository.BookTitle `json:"books"`
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

		var students []repository.Student
		var books []repository.BookTitle
		var studentErr, bookErr error

		// Run both queries concurrently to reduce search latency since they are independent
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			students, studentErr = studentRepo.SearchStudentsFuzzy(ctx, query, limit)
		}()

		go func() {
			defer wg.Done()
			books, bookErr = bookRepo.SearchTitlesFuzzy(ctx, query, limit)
		}()

		wg.Wait()

		if studentErr != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, studentErr)
			return
		}

		if bookErr != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, bookErr)
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
			Students: students,
			Books:    books,
		}

		RespondJSON(w, http.StatusOK, result)
	}
}
