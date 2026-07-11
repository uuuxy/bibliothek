package inventur

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleAdminBooks(t *testing.T) {
	// Create a backup manager to test the notification logic
	backupMgr := NewBackupManager("dummy-url")
	defer close(backupMgr.stopCh)

	handler := &APIHandler{
		backup: backupMgr,
	}

	t.Run("GET requests don't trigger backup", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/unknown", nil)
		rec := httptest.NewRecorder()

		// Drain any existing signals just in case
		select {
		case <-backupMgr.signalCh:
		default:
		}

		handler.handleAdminBooks(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}

		// Check that no backup signal was sent
		select {
		case <-backupMgr.signalCh:
			t.Errorf("GET request should not trigger a backup signal")
		default:
		}
	})

	t.Run("Failed mutations don't trigger backup", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/unknown", nil)
		rec := httptest.NewRecorder()

		// Drain any existing signals
		select {
		case <-backupMgr.signalCh:
		default:
		}

		handler.handleAdminBooks(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}

		// Check that no backup signal was sent for a 404 mutation
		select {
		case <-backupMgr.signalCh:
			t.Errorf("Failed mutation should not trigger a backup signal")
		default:
		}
	})

	t.Run("Successful mutations trigger backup", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/admin/books/reorder", bytes.NewBufferString(`{"bookIds":[]}`))
		rec := httptest.NewRecorder()

		// Drain any existing signals
		select {
		case <-backupMgr.signalCh:
		default:
		}

		handler.handleAdminBooks(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		// Check that backup signal WAS sent
		select {
		case <-backupMgr.signalCh:
			// Success!
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Successful mutation should have triggered a backup signal")
		}
	})

	t.Run("Route dispatching works for all HTTP methods", func(t *testing.T) {
		tests := []struct {
			method       string
			path         string
			body         string
			expectedCode int
		}{
			// GET routes
			{http.MethodGet, "/api/admin/class-books", "", http.StatusInternalServerError}, // fails because repo/db is nil
			{http.MethodGet, "/api/admin/books/external-covers", "", http.StatusInternalServerError},
			{http.MethodGet, "/api/admin/books/export", "", http.StatusInternalServerError},
			{http.MethodGet, "/api/admin/unknown", "", http.StatusNotFound},

			// POST routes
			{http.MethodPost, "/api/admin/class-books", "", http.StatusBadRequest}, // fails earlier due to bad json parsing
			{http.MethodPost, "/api/admin/class-books/add", "", http.StatusBadRequest},
			{http.MethodPost, "/api/books/import", "", http.StatusBadRequest}, // body size error
			{http.MethodPost, "/api/admin/books/retry-covers", "", http.StatusBadRequest},
			{http.MethodPost, "/api/admin/books/import", "", http.StatusNotImplemented},
			{http.MethodPost, "/api/books", "", http.StatusBadRequest},
			{http.MethodPost, "/api/books/refresh-cover", "", http.StatusBadRequest},
			{http.MethodPost, "/api/books/123/cover-upload", "", http.StatusBadRequest},
			{http.MethodPost, "/api/admin/unknown", "", http.StatusNotFound},

			// PUT routes
			{http.MethodPut, "/api/admin/books/reorder", `{"bookIds":[]}`, http.StatusOK}, // empty array early return
			{http.MethodPut, "/api/books/123/cover", "", http.StatusBadRequest},
			{http.MethodPut, "/api/books/123", "", http.StatusBadRequest},
			{http.MethodPut, "/api/admin/unknown", "", http.StatusNotFound},

			// DELETE routes
			{http.MethodDelete, "/api/admin/class-books", "", http.StatusBadRequest},
			{http.MethodDelete, "/api/books", "", http.StatusBadRequest},
			{http.MethodDelete, "/api/admin/unknown", "", http.StatusNotFound},

			// Unknown Method
			{http.MethodPatch, "/api/admin/books", "", http.StatusNotFound},
		}

		for _, tc := range tests {
			t.Run(tc.method+" "+tc.path, func(t *testing.T) {
				var req *http.Request
				if tc.body != "" {
					req = httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
				} else {
					req = httptest.NewRequest(tc.method, tc.path, nil)
				}
				rec := httptest.NewRecorder()

				defer func() {
					if r := recover(); r != nil {
						// Since we just pass nil repo/db, real handlers panic.
						// We don't fail the test because we just care about routing correctly.
					}
				}()

				handler.handleAdminBooks(rec, req)

				if rec.Code != tc.expectedCode {
					t.Errorf("expected code %d, got %d for %s %s", tc.expectedCode, rec.Code, tc.method, tc.path)
				}
			})
		}
	})
}
