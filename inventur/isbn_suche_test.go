package inventur

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleLookupRejectsInvalidISBN(t *testing.T) {
	handler := &APIHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/lookup/123&foo=bar", nil)
	rr := httptest.NewRecorder()

	handler.handleLookup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
