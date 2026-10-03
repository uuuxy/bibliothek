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
	"bibliothek/repository"

	"github.com/google/uuid"
)

// Der Freitext einer Sperre (schueler.block_reason) erreicht auch über die Nachbuch-Tür nur, wer
// view_students hat — wie an der Online-Theke (ohneSperrgrund) und bei der Verlängerung. Die
// Tür selbst steht hinter perform_actions, das auch ein Helfer hat; die Meldung mit dem vollen
// Grund liegt hinter view_students. Der Helfer erfährt, dass gesperrt ist, nicht warum.
func TestNachbuchen_SperrgrundNurMitViewStudents(t *testing.T) {
	helferOhneRechte(t)
	// Der Helfer darf an die Tür (perform_actions), aber keine Schülerdaten ansehen.
	permCacheMu.Lock()
	permCache["helfer:perform_actions"] = cacheEntry{Allowed: true, ExpiresAt: time.Now().Add(time.Minute)}
	permCacheMu.Unlock()
	w := nbTuerAufbau(t)
	ctx := context.Background()
	const grund = "Eltern zahlen Schadensrechnung nicht - Fall Jugendamt"
	if _, err := w.pool.Exec(ctx, `UPDATE schueler SET ist_gesperrt = false, is_manually_blocked = true, block_reason = $2 WHERE id = $1`, w.carla, grund); err != nil {
		t.Fatalf("Sperre von Hand setzen: %v", err)
	}

	nachbuchenAls := func(rolle auth.Role, schluessel string) NachbuchenErgebnis {
		t.Helper()
		body, err := json.Marshal(map[string]any{"gesendet_am": time.Now(), "eintraege": []map[string]any{
			w.eintrag(schluessel, "ausleihe", &w.carla, time.Now().Add(-time.Minute)),
		}})
		if err != nil {
			t.Fatalf("Rumpf: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/action/nachbuchen", strings.NewReader(string(body)))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, &auth.Claims{Rolle: rolle, UserID: w.staff}))
		rec := httptest.NewRecorder()
		w.srv.NachbuchenHandler(w.nachbuch)(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Nachbuchen als %s: Status %d, %s", rolle, rec.Code, rec.Body.String())
		}
		var resp NachbuchenResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || len(resp.Ergebnisse) != 1 {
			t.Fatalf("Antwort als %s: %v, %s", rolle, err, rec.Body.String())
		}
		return resp.Ergebnisse[0]
	}

	schluesselHelfer := uuid.NewString()
	helfer := nachbuchenAls(auth.Role("helfer"), schluesselHelfer)
	if helfer.Ergebnis != repository.NachbuchNichtGebucht || !strings.Contains(helfer.Grund, "gesperrt") {
		t.Fatalf("Helfer: %+v — erwartet nicht_gebucht mit dem Hinweis auf die Sperre", helfer)
	}
	if strings.Contains(helfer.Grund, grund) {
		t.Errorf("der Grund der Sperre erreicht einen Aufrufer ohne view_students: %q", helfer.Grund)
	}
	// Eine wiederholte Portion bekommt die abgelegte Antwort — auch sie ohne den Freitext.
	if nochmal := nachbuchenAls(auth.Role("helfer"), schluesselHelfer); strings.Contains(nochmal.Grund, grund) || nochmal.Grund != helfer.Grund {
		t.Errorf("wiederholte Portion: Grund %q, erwartet dieselbe gekürzte Antwort %q", nochmal.Grund, helfer.Grund)
	}

	admin := nachbuchenAls(auth.RoleAdmin, uuid.NewString())
	if admin.Ergebnis != repository.NachbuchNichtGebucht || !strings.Contains(admin.Grund, grund) {
		t.Errorf("Admin: %+v — wer view_students hat, liest den Grund", admin)
	}

	// Die Meldung liegt hinter view_students und trägt den vollen Grund, auch die des Helfers.
	var mitGrund int
	if err := w.pool.QueryRow(ctx, `
		SELECT count(*) FROM nachbuch_meldungen WHERE barcode = $1 AND ergebnis = 'nicht_gebucht' AND position($2 in grund) > 0`,
		w.code, grund).Scan(&mitGrund); err != nil {
		t.Fatalf("Meldungen lesen: %v", err)
	}
	if mitGrund != 2 {
		t.Errorf("%d Meldungen mit dem vollen Grund, erwartet 2", mitGrund)
	}
}
