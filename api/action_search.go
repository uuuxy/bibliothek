package api

import (
	"context"

	"bibliothek/repository"
)

// handleSearchAction queries book catalog using full-text search triggers.
func (s *Server) handleSearchAction(ctx context.Context, query string, repo repository.BookRepository, resp *ActionResponse) error {
	titles, err := repo.SearchTitles(ctx, query)
	if err != nil {
		return err
	}
	resp.Type = "search_results"
	resp.SearchResults = titles
	return nil
}
