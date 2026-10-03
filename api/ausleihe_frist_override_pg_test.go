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

	// Ein Kollege wird nie gesperrt (16.09. und 24.09.2026) — auch hier nicht, wenn an seinem
	// Konto noch eine Sperre von früher steht. Rot gesehen am Rückbau: ohne die Art in
	// checkAusleiheGesperrt 403.
	t.Run("Kollege mit alter Sperre + Zukunftsdatum geht durch", func(t *testing.T) {
		var kollege string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, is_manually_blocked, block_reason)
			VALUES ('Frist', 'Kollege', 'lehrkraft', true, 'alt') RETURNING id`).Scan(&kollege); err != nil {
			t.Fatalf("Kollegen anlegen: %v", err)
		}
		ausleiheKollege := seedAusleihe(t, pool, kollege, "Buch Kollege", alteFrist)
		if w := call(ausleiheKollege, zukunft); w.Code != http.StatusOK {
			t.Fatalf("erwartet 200, war %d: %s", w.Code, w.Body.String())
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

// Die Frist von Hand, über die Tür: Eine Ablehnung schreibt weder Frist noch Protokoll. Ein
// Zeitstempel gilt wörtlich, ein Datum als Tagesende, und nur eine Frist in der Zukunft setzt
// die Mahnstufe zurück — ein vorgezogenes Datum lässt sie stehen.
func TestOverrideDueDate_AblehnungenZeitstempelUndMahnstufe(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Tür', 'Admin', 'frist-tuer@test.invalid', 'admin', true) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("Test-Admin: %v", err)
	}
	sid := seedSchueler(t, pool, "S-FT-1", "Tuer", "5a")
	alteFrist := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)
	ausleihe := seedAusleihe(t, pool, sid, "Buch Tuer", alteFrist)
	zurueck := seedAusleihe(t, pool, sid, "Buch zurueck", alteFrist)
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = CURRENT_TIMESTAMP WHERE id = $1`, zurueck); err != nil {
		t.Fatalf("Rückgabe buchen: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET mahnstufe = 2, letztes_mahndatum = CURRENT_TIMESTAMP WHERE id = $1`, ausleihe); err != nil {
		t.Fatalf("Mahnstufe setzen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	mux := http.NewServeMux()
	mux.Handle("PATCH /api/admin/ausleihen/{id}/faelligkeit", srv.OverrideDueDateHandler(repository.NewAuditRepository(pool)))
	rufe := func(ausleiheID, rumpf string, angemeldet bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest("PATCH", "/api/admin/ausleihen/"+ausleiheID+"/faelligkeit", strings.NewReader(rumpf))
		if angemeldet {
			req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
				&auth.Claims{UserID: adminID, Rolle: auth.RoleAdmin}))
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}
	stand := func(ausleiheID string) (frist time.Time, mahnstufe int, gemahnt bool) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT rueckgabe_frist, mahnstufe, letztes_mahndatum IS NOT NULL FROM ausleihen WHERE id = $1`,
			ausleiheID).Scan(&frist, &mahnstufe, &gemahnt); err != nil {
			t.Fatalf("Ausleihe lesen: %v", err)
		}
		return frist, mahnstufe, gemahnt
	}

	zukunft := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	const unbekannt = "99999999-9999-9999-9999-999999999999"
	for _, f := range []struct {
		name, ausleiheID, rumpf string
		angemeldet              bool
		status                  int
		stueck                  string
	}{
		{"ohne Sitzung", ausleihe, `{"faellig_am":"` + zukunft + `"}`, false, http.StatusUnauthorized, "Sitzungs-Information"},
		{"ohne Datum", ausleihe, `{}`, true, http.StatusBadRequest, ""},
		{"Datum unlesbar", ausleihe, `{"faellig_am":"morgen"}`, true, http.StatusBadRequest, "ungültiges Datumsformat"},
		{"unbekannte Ausleihe, Frist in der Zukunft", unbekannt, `{"faellig_am":"` + zukunft + `"}`, true, http.StatusNotFound, "nicht gefunden"},
		{"unbekannte Ausleihe, vorgezogene Frist", unbekannt, `{"faellig_am":"2022-06-01"}`, true, http.StatusNotFound, "nicht gefunden"},
		{"zurückgegeben, Frist in der Zukunft", zurueck, `{"faellig_am":"` + zukunft + `"}`, true, http.StatusNotFound, "nicht gefunden"},
		{"zurückgegeben, vorgezogene Frist", zurueck, `{"faellig_am":"2022-06-01"}`, true, http.StatusNotFound, "nicht gefunden"},
	} {
		w := rufe(f.ausleiheID, f.rumpf, f.angemeldet)
		if w.Code != f.status {
			t.Errorf("%s: Status %d, erwartet %d: %s", f.name, w.Code, f.status, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), f.stueck) {
			t.Errorf("%s: die Antwort nennt %q nicht: %s", f.name, f.stueck, w.Body.String())
		}
	}
	for _, id := range []string{ausleihe, zurueck} {
		if frist, _, _ := stand(id); !frist.Equal(alteFrist) {
			t.Fatalf("eine Ablehnung hat die Frist geschrieben: %s", frist.UTC().Format(time.RFC3339))
		}
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'FRIST_OVERRIDE'`); n != 0 {
		t.Fatalf("%d Protokolleinträge nach sieben Ablehnungen", n)
	}

	// Vorgezogen: Die Frist steht, die Mahnstufe bleibt.
	if w := rufe(ausleihe, `{"faellig_am":"2022-06-01"}`, true); w.Code != http.StatusOK {
		t.Fatalf("vorgezogene Frist: Status %d: %s", w.Code, w.Body.String())
	}
	if _, mahnstufe, gemahnt := stand(ausleihe); mahnstufe != 2 || !gemahnt {
		t.Errorf("nach vorgezogener Frist: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 2 und true", mahnstufe, gemahnt)
	}

	// Ein Zeitstempel in der Zukunft gilt auf die Sekunde und beginnt die Mahnfolge neu.
	stempel := time.Now().AddDate(1, 0, 0).UTC().Truncate(time.Second).Add(90 * time.Minute)
	if w := rufe(ausleihe, `{"faellig_am":"`+stempel.Format(time.RFC3339)+`"}`, true); w.Code != http.StatusOK {
		t.Fatalf("Zeitstempel: Status %d: %s", w.Code, w.Body.String())
	}
	frist, mahnstufe, gemahnt := stand(ausleihe)
	if !frist.Equal(stempel) {
		t.Errorf("Frist %s, erwartet den Zeitstempel %s — kein Tagesende", frist.UTC().Format(time.RFC3339), stempel.Format(time.RFC3339))
	}
	if mahnstufe != 0 || gemahnt {
		t.Errorf("nach Frist in der Zukunft: Mahnstufe %d, Mahndatum gesetzt %v — erwartet 0 und false", mahnstufe, gemahnt)
	}
	// Das Protokoll nennt je Änderung den Bearbeiter, die Ausleihe und die neue Frist.
	rows, err := pool.Query(ctx, `
		SELECT details->>'neue_frist' FROM audit_logs
		WHERE aktion = 'FRIST_OVERRIDE' AND admin_id = $1 AND details->>'ausleihe_id' = $2 ORDER BY zeitstempel, id`, adminID, ausleihe)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	defer rows.Close()
	var protokolliert []time.Time
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		frist, err := time.Parse(time.RFC3339, text)
		if err != nil {
			t.Fatalf("neue_frist %q im Protokoll ist kein Zeitstempel: %v", text, err)
		}
		protokolliert = append(protokolliert, frist)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if len(protokolliert) != 2 || !protokolliert[1].Equal(stempel) {
		t.Errorf("Protokoll nennt die Fristen %v — erwartet zwei Einträge, der zweite mit %s", protokolliert, stempel.Format(time.RFC3339))
	}
}
