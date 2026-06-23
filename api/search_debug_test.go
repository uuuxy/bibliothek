package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

func TestSearchMax(t *testing.T) {
	// Integration test: requires a live Postgres instance. Skipped automatically
	// when the database is unreachable (e.g. CI unit-test runs) or under -short.
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn := "postgres://postgres:postgrespassword@127.0.0.1:5434/bibliothek?sslmode=disable"

	// Create a background context with a timeout so it doesn't hang if db isn't there
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	database, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("skipping: database unavailable (%v)", err)
	}
	defer database.Close()

	studentRepo := repository.NewStudentRepository(database.Pool)
	bookRepo := repository.NewBookRepository(database.Pool)

	srv := NewServer(database, nil, nil, false)
	handler := srv.SearchHandler(studentRepo, bookRepo)

	req := httptest.NewRequest("GET", "/api/search?q=max:1", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
	t.Logf("Success! Body: %s", rr.Body.String())
}
