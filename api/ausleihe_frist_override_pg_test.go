package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Der Frist-Override darf bei einem GESPERRTEN Schüler die Sanktion nicht aushebeln:
// Eine Frist in die Zukunft macht die Ausleihe wieder "nicht überfällig" und löscht
// die Mahn-Eskalation — genau das, was die beiden Verlängerungs-Endpunkte auf
// demselben Recht ausdrücklich verbieten. Ein VORGEZOGENES Datum (Rückruf) bleibt
// erlaubt, weil es die Sperre nicht aufhebt. Und jeder Override wird auditiert.
// Live gefunden 18.08.2026 (Rollen×Aktionen-Prüfung); vorher ging beides ungeprüft.
func TestOverrideDueDate_SperrKonsistenzUndAudit(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	// Echter Benutzer für den Audit-FK (audit_logs.admin_id).
	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Frist', 'Admin', 'frist-admin@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}

	sidNormal := seedSchueler(t, pool, "S-FO-1", "Normal", "5a")
	sidGesperrt := seedSchueler(t, pool, "S-FO-2", "Gesperrt", "5b")
	if _, err := pool.Exec(ctx, "UPDATE schueler SET is_manually_blocked = true, block_reason = 'Buchverlust' WHERE id = $1", sidGesperrt); err != nil {
		t.Fatalf("Sperre setzen: %v", err)
	}

	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)
	ausleiheNormal := seedAusleihe(t, pool, sidNormal, "Buch Normal", alteFrist)
	ausleiheGesperrt := seedAusleihe(t, pool, sidGesperrt, "Buch Gesperrt", alteFrist)

	zukunft := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	vergangenheit := "2022-06-01"

	call := func(ausleiheID, datum string) *httptest.ResponseRecorder {
		srv := &Server{DB: &db.Database{Pool: pool}}
		auditRepo := repository.NewAuditRepository(pool)
		mux := http.NewServeMux()
		mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", srv.OverrideDueDateHandler(auditRepo))
		req := httptest.NewRequest("PATCH", "/api/admin/ausleihen/"+ausleiheID+"/faelligkeit",
			strings.NewReader(`{"faellig_am":"`+datum+`"}`))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	auditZaehler := func() int {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE aktion = 'FRIST_OVERRIDE'").Scan(&n); err != nil {
			t.Fatalf("Audit zählen: %v", err)
		}
		return n
	}

	t.Run("gesperrt + Zukunftsdatum wird verweigert", func(t *testing.T) {
		vorher := auditZaehler()
		w := call(ausleiheGesperrt, zukunft)
		if w.Code != http.StatusForbidden {
			t.Fatalf("erwartet 403, war %d: %s", w.Code, w.Body.String())
		}
		if auditZaehler() != vorher {
			t.Error("ein abgelehnter Override darf keinen Audit-Eintrag erzeugen")
		}
	})

	t.Run("gesperrt + Vergangenheitsdatum (Rueckruf) ist erlaubt", func(t *testing.T) {
		w := call(ausleiheGesperrt, vergangenheit)
		if w.Code != http.StatusOK {
			t.Fatalf("Rückruf eines gesperrten Schülers muss gehen, war %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("normal + Zukunftsdatum geht durch und wird auditiert", func(t *testing.T) {
		vorher := auditZaehler()
		w := call(ausleiheNormal, zukunft)
		if w.Code != http.StatusOK {
			t.Fatalf("erwartet 200, war %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Antwort unlesbar: %v", err)
		}
		if resp["success"] != true {
			t.Errorf("success erwartet, war %v", resp["success"])
		}
		if auditZaehler() != vorher+1 {
			t.Error("ein durchgeführter Override muss GENAU EINEN Audit-Eintrag erzeugen")
		}
	})
}

// TestOverrideDueDate_TagesendeInSchulzeitzone belegt die eine Definition von "Tagesende"
// (Zeit-Sweep 19.08.2026): Der Frist-Override setzt 23:59:59 in der Schulzeitzone (Berlin),
// nicht roh in UTC. Sonst wäre eine überschriebene Frist zum selben Datum 1–2 h später
// fällig als eine regulär berechnete — zwei Antworten auf dieselbe fachliche Frage.
func TestOverrideDueDate_TagesendeInSchulzeitzone(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('TZ', 'Admin', 'tz-admin@test.invalid', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET vorname = EXCLUDED.vorname
		RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	sid := seedSchueler(t, pool, "S-TZ-1", "Zone", "5a")
	ausleihe := seedAusleihe(t, pool, sid, "Buch TZ", time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC))

	srv := &Server{DB: &db.Database{Pool: pool}}
	auditRepo := repository.NewAuditRepository(pool)
	mux := http.NewServeMux()
	mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", srv.OverrideDueDateHandler(auditRepo))

	// Sommerdatum (CEST, +02:00): Berliner Tagesende 23:59:59 = 21:59:59 UTC.
	req := httptest.NewRequest("PATCH", "/api/admin/ausleihen/"+ausleihe+"/faelligkeit",
		strings.NewReader(`{"faellig_am":"2027-07-15"}`))
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Override: erwartet 200, war %d: %s", w.Code, w.Body.String())
	}

	var frist time.Time
	if err := pool.QueryRow(ctx,
		`SELECT rueckgabe_frist FROM ausleihen WHERE id = $1`, ausleihe).Scan(&frist); err != nil {
		t.Fatalf("Frist lesen: %v", err)
	}
	fristUTC := frist.UTC()
	// Muss der 15.07. um 21:59:59 UTC sein (= 23:59:59 Berlin/CEST) — NICHT 23:59:59 UTC.
	if fristUTC.Hour() != 21 || fristUTC.Day() != 15 || fristUTC.Month() != time.July {
		t.Errorf("Frist muss Berliner Tagesende sein (21:59:59Z am 15.07.), war %s", fristUTC.Format(time.RFC3339))
	}
}
