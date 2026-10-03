package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/schulzeit"
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

func (m *mockSystemSettingsRepo) SaveSettings(ctx context.Context, req *repository.EinstellungenPatch) error {
	return nil
}

// pruefeFrist belegt die neue Frist auf die Sekunde: Seit dem 24.09.2026 rechnet die
// Verlängerung mit der Uhr des Servers (s.jetzt) statt mit CURRENT_TIMESTAMP, der Test setzt
// sie fest. Vorher prüfte er mit einem Tag Toleranz gegen time.Now() — das hätte weder das
// Tagesende noch den nächsten Schultag gesehen.
func pruefeFrist(t *testing.T, frist, soll time.Time) {
	t.Helper()
	if !frist.Equal(soll) {
		t.Errorf("Frist %s, erwartet %s", frist.In(schulzeit.Zone()), soll)
	}
}

// fristEnde ist das Tagesende eines Kalendertags in der Schulzeitzone.
func fristEnde(j int, m time.Month, d int) time.Time {
	return time.Date(j, m, d, 23, 59, 59, 0, schulzeit.Zone())
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

	// Eine eigene Ausleihe für den Fall, dass die neue Frist in die Ferien fällt.
	ausleiheFerien := seedAusleihe(t, pool, sid1, "Testbuch Ferien", alteFrist)

	// Create a returned loan
	ausleiheReturned := seedAusleihe(t, pool, sid1, "Testbuch Returned", alteFrist)
	_, err = pool.Exec(context.Background(), "UPDATE ausleihen SET rueckgabe_am = CURRENT_TIMESTAMP WHERE id = $1", ausleiheReturned)
	if err != nil {
		t.Fatalf("Failed to mark loan as returned: %v", err)
	}

	// Montag 02.11.2026: 14, 21 und 28 Tage fallen von hier aus auf verschiedene Schultage,
	// die Fälle unterscheiden die Tageszahl also.
	november := time.Date(2026, time.November, 2, 10, 0, 0, 0, schulzeit.Zone())

	tests := []struct {
		name           string
		ausleiheID     string
		jetzt          time.Time
		extensionDays  int
		setupRoute     func(*http.ServeMux, *Server, repository.SystemSettingsRepository)
		expectedStatus int
		verify         func(t *testing.T, resp map[string]interface{})
	}{
		{
			name:           "Happy Path - Extends Loan with configured interval",
			ausleiheID:     ausleiheNormal,
			jetzt:          november,
			extensionDays:  14,
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				if success, ok := resp["success"].(bool); !ok || !success {
					t.Errorf("Expected success=true, got %v", resp["success"])
				}

				// Die Verlaengerung rechnet ab GREATEST(alte Frist, jetzt) — siehe
				// api/ausleihe.go. Bei einer laengst ueberfaelligen Ausleihe (hier 2023)
				// ist das HEUTE, nicht die alte Frist: Sonst käme eine Verlaengerung
				// heraus, die im Moment der Buchung schon wieder abgelaufen ist.
				// Heute plus 14 Tage, auf das Tagesende: Montag 16.11.2026.
				pruefeFrist(t, fristVon(t, pool, ausleiheNormal), fristEnde(2026, time.November, 16))
			},
		},
		{
			name:           "Happy Path - Falls back to 28 days if interval missing",
			ausleiheID:     ausleiheNormal, // Using same loan is fine, we just update it again
			jetzt:          november,
			extensionDays:  0, // Will trigger fallback to 28
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				// Der vorige Fall hat die Frist bereits auf den 16.11. gesetzt. Die liegt
				// in der ZUKUNFT, also rechnet die Verlängerung ab ihr: 16.11. plus 28 Tage.
				pruefeFrist(t, fristVon(t, pool, ausleiheNormal), fristEnde(2026, time.December, 14))
			},
		},
		{
			// Entscheidung vom 24.09.2026: Donnerstag 24.09.2026 plus 21 Tage ist der 15.10.,
			// mitten in den Herbstferien (05.10.–17.10.) — die Frist ist der Montag danach.
			name:           "Frist in den Herbstferien rückt auf den nächsten Schultag",
			ausleiheID:     ausleiheFerien,
			jetzt:          time.Date(2026, time.September, 24, 10, 0, 0, 0, schulzeit.Zone()),
			extensionDays:  21,
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				pruefeFrist(t, fristVon(t, pool, ausleiheFerien), fristEnde(2026, time.October, 19))
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
			verify:         func(t *testing.T, resp map[string]interface{}) {},
		},
		{
			name:           "Error - Loan already returned",
			ausleiheID:     ausleiheReturned,
			extensionDays:  28,
			expectedStatus: http.StatusNotFound,
			verify:         func(t *testing.T, resp map[string]interface{}) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{DB: &db.Database{Pool: pool}}
			if !tt.jetzt.IsZero() {
				srv.Uhr = func() time.Time { return tt.jetzt }
			}

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
					_ = json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck // Error cases don't strictly require JSON
				}
				tt.verify(t, resp)
			}
		})
	}
}

// Die Verlängerung einer gesperrten Ausleihe wird abgelehnt, und den Grund der Sperre liest nur,
// wer view_students hat (wie an der Theke, ohneSperrgrund). Gelingt sie, beginnt die Mahnfolge neu.
func TestExtendLoan_SperrgrundNurMitViewStudentsUndMahnstufe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	InvalidatePermissionCache()
	t.Cleanup(InvalidatePermissionCache)
	permCacheMu.Lock()
	permCache["helfer:view_students"] = cacheEntry{Allowed: false, ExpiresAt: time.Now().Add(time.Minute)}
	permCacheMu.Unlock()

	const grund = "Schadensrechnung offen - Familie"
	gesperrt := seedSchueler(t, pool, "S-VG-1", "Gesperrt", "5a")
	if _, err := pool.Exec(ctx, `UPDATE schueler SET is_manually_blocked = true, block_reason = $2 WHERE id = $1`, gesperrt, grund); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}
	frei := seedSchueler(t, pool, "S-VG-2", "Frei", "5a")
	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)
	ausleiheGesperrt := seedAusleihe(t, pool, gesperrt, "Buch der Gesperrten", alteFrist)
	ausleiheFrei := seedAusleihe(t, pool, frei, "Buch des Freien", alteFrist)
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET mahnstufe = 2, letztes_mahndatum = CURRENT_TIMESTAMP WHERE id = $1`, ausleiheFrei); err != nil {
		t.Fatalf("Mahnstufe setzen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	mux := http.NewServeMux()
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", srv.ExtendLoanHandler(&mockSystemSettingsRepo{
		settings: &repository.SystemEinstellungen{FristBuchTage: 21},
	}))
	verlaengere := func(ausleiheID string, rolle auth.Role) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/ausleihen/"+ausleiheID+"/verlaengern", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{Rolle: rolle}))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	helfer := verlaengere(ausleiheGesperrt, auth.Role("helfer"))
	if helfer.Code != http.StatusForbidden || !strings.Contains(helfer.Body.String(), "Ausleihe gesperrt") {
		t.Errorf("Helfer: Status %d, erwartet 403 mit dem Hinweis auf die Sperre: %s", helfer.Code, helfer.Body.String())
	}
	if strings.Contains(helfer.Body.String(), grund) {
		t.Errorf("der Grund der Sperre erreicht einen Aufrufer ohne view_students: %s", helfer.Body.String())
	}
	admin := verlaengere(ausleiheGesperrt, auth.RoleAdmin)
	if admin.Code != http.StatusForbidden || !strings.Contains(admin.Body.String(), grund) {
		t.Errorf("Admin: Status %d, erwartet 403 mit dem Grund der Sperre: %s", admin.Code, admin.Body.String())
	}
	if frist := fristVon(t, pool, ausleiheGesperrt); !frist.Equal(alteFrist) {
		t.Errorf("die abgelehnte Verlängerung hat die Frist geschrieben: %s", frist.Format(time.RFC3339))
	}

	if rec := verlaengere(ausleiheFrei, auth.Role("helfer")); rec.Code != http.StatusOK {
		t.Fatalf("Verlängerung: Status %d: %s", rec.Code, rec.Body.String())
	}
	var mahnstufe int
	var gemahnt bool
	if err := pool.QueryRow(ctx, `SELECT mahnstufe, letztes_mahndatum IS NOT NULL FROM ausleihen WHERE id = $1`,
		ausleiheFrei).Scan(&mahnstufe, &gemahnt); err != nil {
		t.Fatalf("Ausleihe lesen: %v", err)
	}
	if mahnstufe != 0 || gemahnt {
		t.Errorf("nach der Verlängerung: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 0 und false", mahnstufe, gemahnt)
	}
	if frist := fristVon(t, pool, ausleiheFrei); !frist.After(time.Now()) {
		t.Errorf("die neue Frist %s liegt nicht in der Zukunft", frist.Format(time.RFC3339))
	}
}

// Die Verlängerung rechnet mit den Sommerferien aus den Einstellungen, auch für ein Jahr, das die
// Tabelle des Programms nicht kennt. Sind die Einstellungen nicht lesbar, gelten 28 Tage.
func TestExtendLoan_FerienDerEinstellungenUndVorgabeOhneEinstellungen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	sid := seedSchueler(t, pool, "S-VF-1", "Ferien", "5a")
	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)

	verlaengere := func(ausleiheID string, jetzt time.Time, einstellungen repository.SystemSettingsRepository) {
		t.Helper()
		srv := &Server{DB: &db.Database{Pool: pool}, Uhr: func() time.Time { return jetzt }}
		mux := http.NewServeMux()
		mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", srv.ExtendLoanHandler(einstellungen))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/ausleihen/"+ausleiheID+"/verlaengern", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Verlängerung: Status %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Montag 30.06.2031 plus 21 Tage ist Montag, der 21.07.2031 — nach den Einstellungen mitten
	// in den Sommerferien (07.07.–15.08.), die Frist ist der Montag danach.
	inDenFerien := seedAusleihe(t, pool, sid, "Buch vor den Ferien", alteFrist)
	verlaengere(inDenFerien, time.Date(2031, time.June, 30, 10, 0, 0, 0, schulzeit.Zone()), &mockSystemSettingsRepo{
		settings: &repository.SystemEinstellungen{
			FristBuchTage: 21,
			Sommerferien:  `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]`,
		},
	})
	pruefeFrist(t, fristVon(t, pool, inDenFerien), fristEnde(2031, time.August, 18))

	// Montag 02.11.2026 plus 28 Tage ist Montag, der 30.11.2026.
	ohneEinstellungen := seedAusleihe(t, pool, sid, "Buch ohne Einstellungen", alteFrist)
	verlaengere(ohneEinstellungen, time.Date(2026, time.November, 2, 10, 0, 0, 0, schulzeit.Zone()),
		&mockSystemSettingsRepo{err: errors.New("einstellungen nicht lesbar")})
	pruefeFrist(t, fristVon(t, pool, ohneEinstellungen), fristEnde(2026, time.November, 30))
}
