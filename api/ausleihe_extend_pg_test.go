package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/repository"
)

// mockSystemSettingsRepo is a simple mock to provide custom SystemEinstellungen.
type mockSystemSettingsRepo struct {
	settings *repository.SystemEinstellungen
	err      error
}

func (m *mockSystemSettingsRepo) GetSettings(ctx context.Context) (*repository.SystemEinstellungen, error) {
	return m.settings, m.err
}

func (m *mockSystemSettingsRepo) SaveSettings(ctx context.Context, req *repository.SystemEinstellungen) error {
	return nil
}

func TestExtendLoanHandler(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)

	// Create a test student
	sid1 := seedSchueler(t, pool, "S-EXT-1", "Normal", "5a")
	sidGesperrt := seedSchueler(t, pool, "S-EXT-2", "Gesperrt", "5b")

	// Manually block the second student
	_, err := pool.Exec(context.Background(), "UPDATE schueler SET ist_gesperrt = true, block_reason = 'Mahnung' WHERE id = $1", sidGesperrt)
	if err != nil {
		t.Fatalf("Failed to block student: %v", err)
	}

	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)

	// Create loans
	ausleiheNormal := seedAusleihe(t, pool, sid1, "Testbuch Normal", alteFrist)
	ausleiheGesperrt := seedAusleihe(t, pool, sidGesperrt, "Testbuch Gesperrt", alteFrist)

	// Create a returned loan
	ausleiheReturned := seedAusleihe(t, pool, sid1, "Testbuch Returned", alteFrist)
	_, err = pool.Exec(context.Background(), "UPDATE ausleihen SET rueckgabe_am = CURRENT_TIMESTAMP WHERE id = $1", ausleiheReturned)
	if err != nil {
		t.Fatalf("Failed to mark loan as returned: %v", err)
	}

	tests := []struct {
		name           string
		ausleiheID     string
		extensionDays  int
		setupRoute     func(*http.ServeMux, *Server, repository.SystemSettingsRepository)
		expectedStatus int
		verify         func(t *testing.T, resp map[string]interface{})
	}{
		{
			name:           "Happy Path - Extends Loan with configured interval",
			ausleiheID:     ausleiheNormal,
			extensionDays:  14,
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); !ok || !success {
					t.Errorf("Expected success=true, got %v", resp["success"])
				}

				// Verify DB state
				newFrist := fristVon(t, pool, ausleiheNormal)
				expectedFrist := alteFrist.AddDate(0, 0, 14)

				if !newFrist.Equal(expectedFrist) {
					t.Errorf("Frist not updated correctly. Expected %v, got %v", expectedFrist, newFrist)
				}
			},
		},
		{
			name:           "Happy Path - Falls back to 28 days if interval missing",
			ausleiheID:     ausleiheNormal, // Using same loan is fine, we just update it again
			extensionDays:  0, // Will trigger fallback to 28
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				// The previous frist was alteFrist + 14, now it will be THAT + 28
				newFrist := fristVon(t, pool, ausleiheNormal)
				expectedFrist := alteFrist.AddDate(0, 0, 14+28)

				if !newFrist.Equal(expectedFrist) {
					t.Errorf("Frist not updated correctly with fallback. Expected %v, got %v", expectedFrist, newFrist)
				}
			},
		},
		{
			name:           "Error - Student is suspended",
			ausleiheID:     ausleiheGesperrt,
			extensionDays:  28,
			expectedStatus: http.StatusForbidden,
			verify: func(t *testing.T, resp map[string]interface{}) {
				if msg, ok := resp["error"].(string); !ok || msg == "" {
					t.Errorf("Expected error message, got %v", resp)
				}
			},
		},
		{
			name:           "Error - Missing ausleihe_id",
			ausleiheID:     "",
			extensionDays:  28,
			expectedStatus: http.StatusBadRequest, // Mux will match the /extend/ route, handler will return 400
			setupRoute: func(mux *http.ServeMux, s *Server, settingsRepo repository.SystemSettingsRepository) {
				// We map it directly to test the handler's internal check for ""
				mux.Handle("POST /extend/", s.ExtendLoanHandler(settingsRepo))
			},
			verify: func(t *testing.T, resp map[string]interface{}) {},
		},
		{
			name:           "Error - Loan not found",
			ausleiheID:     "99999999-9999-9999-9999-999999999999",
			extensionDays:  28,
			expectedStatus: http.StatusNotFound,
			verify: func(t *testing.T, resp map[string]interface{}) {},
		},
		{
			name:           "Error - Loan already returned",
			ausleiheID:     ausleiheReturned,
			extensionDays:  28,
			expectedStatus: http.StatusNotFound,
			verify: func(t *testing.T, resp map[string]interface{}) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{DB: &db.Database{Pool: pool}}

			settingsRepo := &mockSystemSettingsRepo{
				settings: &repository.SystemEinstellungen{
					FristBuchTage: tt.extensionDays,
				},
			}

			mux := http.NewServeMux()
			if tt.setupRoute != nil {
				tt.setupRoute(mux, srv, settingsRepo)
			} else {
				mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", srv.ExtendLoanHandler(settingsRepo))
			}

			path := "/api/ausleihen/" + tt.ausleiheID + "/verlaengern"
			if tt.ausleiheID == "" && tt.setupRoute != nil {
				path = "/extend/"
			}

			req := httptest.NewRequest(http.MethodPost, path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			if tt.verify != nil && rec.Code == http.StatusOK {
				var resp map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.verify(t, resp)
			} else if tt.verify != nil && rec.Code != http.StatusOK {
				// For error cases we also want to decode
				var resp map[string]interface{}
				if rec.Body.Len() > 0 {
					json.NewDecoder(rec.Body).Decode(&resp)
				}
				tt.verify(t, resp)
			}
		})
	}
}
