package theke

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/internal/tuertest"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"
)

// pruefeFrist belegt die neue Frist auf die Sekunde: Die Verlängerung rechnet mit der Uhr
// des Servers statt mit CURRENT_TIMESTAMP, und der Test setzt sie fest. Mit einem Tag Toleranz
// gegen time.Now() sähe er weder das Tagesende noch den nächsten Schultag.
func pruefeFrist(t *testing.T, frist, soll time.Time) {
	t.Helper()
	if !frist.Equal(soll) {
		t.Errorf("Frist %s, erwartet %s", frist.In(schulzeit.Zone()), soll)
	}
}

func TestAusleiheVerlaengern(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)

	// Zwei Schüler, der zweite ist gesperrt.
	sid1 := testdaten.LegeSchuelerAn(t, pool, "S-EXT-1", "Normal", "5a")
	sidGesperrt := testdaten.LegeSchuelerAn(t, pool, "S-EXT-2", "Gesperrt", "5b")

	_, err := pool.Exec(context.Background(), "UPDATE schueler SET ist_gesperrt = true, block_reason = 'Mahnung' WHERE id = $1", sidGesperrt)
	if err != nil {
		t.Fatalf("Failed to block student: %v", err)
	}

	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)

	// Ausleihen mit einer längst abgelaufenen Frist.
	ausleiheNormal := testdaten.LegeAusleiheMitFristAn(t, pool, sid1, "Testbuch Normal", alteFrist)
	ausleiheGesperrt := testdaten.LegeAusleiheMitFristAn(t, pool, sidGesperrt, "Testbuch Gesperrt", alteFrist)

	// Eine eigene Ausleihe für den Fall, dass die neue Frist in die Ferien fällt.
	ausleiheFerien := testdaten.LegeAusleiheMitFristAn(t, pool, sid1, "Testbuch Ferien", alteFrist)

	// Eine zurückgegebene Ausleihe.
	ausleiheReturned := testdaten.LegeAusleiheMitFristAn(t, pool, sid1, "Testbuch Returned", alteFrist)
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
		setupRoute     func(*http.ServeMux, *Tueren, repository.EinstellungsAbfragen)
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

				// Die Verlängerung rechnet ab dem späteren von alter Frist und heute
				// (api/theke/ausleihe.go). Bei einer längst überfälligen Ausleihe (hier 2023)
				// ist das heute, nicht die alte Frist: Sonst käme eine Verlängerung heraus,
				// die im Moment der Buchung schon wieder abgelaufen ist.
				// Heute plus 14 Tage, auf das Tagesende: Montag 16.11.2026.
				pruefeFrist(t, testdaten.LadeFrist(t, pool, ausleiheNormal), testdaten.FristEnde(2026, time.November, 16))
			},
		},
		{
			name:           "Happy Path - Falls back to 28 days if interval missing",
			ausleiheID:     ausleiheNormal, // dieselbe Ausleihe, sie wird erneut verlängert
			jetzt:          november,
			extensionDays:  0, // ohne Einstellung gelten 28 Tage
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				// Der vorige Fall hat die Frist bereits auf den 16.11. gesetzt. Die liegt
				// in der Zukunft, also rechnet die Verlängerung ab ihr: 16.11. plus 28 Tage.
				pruefeFrist(t, testdaten.LadeFrist(t, pool, ausleiheNormal), testdaten.FristEnde(2026, time.December, 14))
			},
		},
		{
			// Donnerstag 24.09.2026 plus 21 Tage ist der 15.10.,
			// mitten in den Herbstferien (05.10.–17.10.) — die Frist ist der Montag danach.
			name:           "Frist in den Herbstferien rückt auf den nächsten Schultag",
			ausleiheID:     ausleiheFerien,
			jetzt:          time.Date(2026, time.September, 24, 10, 0, 0, 0, schulzeit.Zone()),
			extensionDays:  21,
			expectedStatus: http.StatusOK,
			verify: func(t *testing.T, resp map[string]interface{}) {
				pruefeFrist(t, testdaten.LadeFrist(t, pool, ausleiheFerien), testdaten.FristEnde(2026, time.October, 19))
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
			expectedStatus: http.StatusBadRequest, // der Mux trifft /extend/, die Tür antwortet 400
			setupRoute: func(mux *http.ServeMux, s *Tueren, settingsRepo repository.EinstellungsAbfragen) {
				// Direkt eingehängt, damit die Tür selbst die leere Kennung prüft.
				mux.Handle("POST /extend/", s.AusleiheVerlaengern(settingsRepo))
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
			srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
			if !tt.jetzt.IsZero() {
				srv.Uhr = func() time.Time { return tt.jetzt }
			}

			settingsRepo := &tuertest.EinstellungenAttrappe{
				Einstellungen: &repository.SystemEinstellungen{
					FristBuchTage: tt.extensionDays,
				},
			}

			mux := http.NewServeMux()
			if tt.setupRoute != nil {
				tt.setupRoute(mux, srv, settingsRepo)
			} else {
				mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", srv.AusleiheVerlaengern(settingsRepo))
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
				// Auch eine Fehlerantwort wird gelesen.
				var resp map[string]interface{}
				if rec.Body.Len() > 0 {
					_ = json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck // eine Fehlerantwort muss kein JSON sein
				}
				tt.verify(t, resp)
			}
		})
	}
}

// Die Verlängerung einer gesperrten Ausleihe wird abgelehnt, und den Grund der Sperre liest nur,
// wer view_students hat (wie an der Theke, ohneSperrgrund). Gelingt sie, beginnt die Mahnfolge neu.
func TestAusleiheVerlaengern_SperrgrundNurMitViewStudentsUndMahnstufe(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	tuer.LeereRechteCache()
	t.Cleanup(tuer.LeereRechteCache)
	tuer.MerkeRechtFuerTest("helfer", "view_students", false)

	const grund = "Schadensrechnung offen - Familie"
	gesperrt := testdaten.LegeSchuelerAn(t, pool, "S-VG-1", "Gesperrt", "5a")
	if _, err := pool.Exec(ctx, `UPDATE schueler SET is_manually_blocked = true, block_reason = $2 WHERE id = $1`, gesperrt, grund); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}
	frei := testdaten.LegeSchuelerAn(t, pool, "S-VG-2", "Frei", "5a")
	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)
	ausleiheGesperrt := testdaten.LegeAusleiheMitFristAn(t, pool, gesperrt, "Buch der Gesperrten", alteFrist)
	ausleiheFrei := testdaten.LegeAusleiheMitFristAn(t, pool, frei, "Buch des Freien", alteFrist)
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET mahnstufe = 2, letztes_mahndatum = CURRENT_TIMESTAMP WHERE id = $1`, ausleiheFrei); err != nil {
		t.Fatalf("Mahnstufe setzen: %v", err)
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	mux := http.NewServeMux()
	mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", srv.AusleiheVerlaengern(&tuertest.EinstellungenAttrappe{
		Einstellungen: &repository.SystemEinstellungen{FristBuchTage: 21},
	}))
	verlaengere := func(ausleiheID string, rolle auth.Rolle) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/ausleihen/"+ausleiheID+"/verlaengern", nil)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsSchluessel, &auth.Claims{Rolle: rolle}))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	helfer := verlaengere(ausleiheGesperrt, auth.Rolle("helfer"))
	if helfer.Code != http.StatusForbidden || !strings.Contains(helfer.Body.String(), "Ausleihe gesperrt") {
		t.Errorf("Helfer: Status %d, erwartet 403 mit dem Hinweis auf die Sperre: %s", helfer.Code, helfer.Body.String())
	}
	if strings.Contains(helfer.Body.String(), grund) {
		t.Errorf("der Grund der Sperre erreicht einen Aufrufer ohne view_students: %s", helfer.Body.String())
	}
	admin := verlaengere(ausleiheGesperrt, auth.RolleAdmin)
	if admin.Code != http.StatusForbidden || !strings.Contains(admin.Body.String(), grund) {
		t.Errorf("Admin: Status %d, erwartet 403 mit dem Grund der Sperre: %s", admin.Code, admin.Body.String())
	}
	if frist := testdaten.LadeFrist(t, pool, ausleiheGesperrt); !frist.Equal(alteFrist) {
		t.Errorf("die abgelehnte Verlängerung hat die Frist geschrieben: %s", frist.Format(time.RFC3339))
	}

	if rec := verlaengere(ausleiheFrei, auth.Rolle("helfer")); rec.Code != http.StatusOK {
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
	if frist := testdaten.LadeFrist(t, pool, ausleiheFrei); !frist.After(time.Now()) {
		t.Errorf("die neue Frist %s liegt nicht in der Zukunft", frist.Format(time.RFC3339))
	}
}

// Die Verlängerung rechnet mit den Sommerferien aus den Einstellungen, auch für ein Jahr, das die
// Tabelle des Programms nicht kennt. Sind die Einstellungen nicht lesbar, gelten 28 Tage.
func TestAusleiheVerlaengern_FerienDerEinstellungenUndVorgabeOhneEinstellungen(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	sid := testdaten.LegeSchuelerAn(t, pool, "S-VF-1", "Ferien", "5a")
	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)

	verlaengere := func(ausleiheID string, jetzt time.Time, einstellungen repository.EinstellungsAbfragen) {
		t.Helper()
		srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}, Uhr: func() time.Time { return jetzt }}}
		mux := http.NewServeMux()
		mux.Handle("POST /api/ausleihen/{ausleihe_id}/verlaengern", srv.AusleiheVerlaengern(einstellungen))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/ausleihen/"+ausleiheID+"/verlaengern", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("Verlängerung: Status %d: %s", rec.Code, rec.Body.String())
		}
	}

	// Montag 30.06.2031 plus 21 Tage ist Montag, der 21.07.2031 — nach den Einstellungen mitten
	// in den Sommerferien (07.07.–15.08.), die Frist ist der Montag danach.
	inDenFerien := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "Buch vor den Ferien", alteFrist)
	verlaengere(inDenFerien, time.Date(2031, time.June, 30, 10, 0, 0, 0, schulzeit.Zone()), &tuertest.EinstellungenAttrappe{
		Einstellungen: &repository.SystemEinstellungen{
			FristBuchTage: 21,
			Sommerferien:  `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]`,
		},
	})
	pruefeFrist(t, testdaten.LadeFrist(t, pool, inDenFerien), testdaten.FristEnde(2031, time.August, 18))

	// Montag 02.11.2026 plus 28 Tage ist Montag, der 30.11.2026.
	ohneEinstellungen := testdaten.LegeAusleiheMitFristAn(t, pool, sid, "Buch ohne Einstellungen", alteFrist)
	verlaengere(ohneEinstellungen, time.Date(2026, time.November, 2, 10, 0, 0, 0, schulzeit.Zone()),
		&tuertest.EinstellungenAttrappe{Fehler: errors.New("einstellungen nicht lesbar")})
	pruefeFrist(t, testdaten.LadeFrist(t, pool, ohneEinstellungen), testdaten.FristEnde(2026, time.November, 30))
}
