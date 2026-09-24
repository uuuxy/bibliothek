package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sperrTuerAufruf ruft den Sperr-Endpunkt als angemeldeter Akteur — die Tür protokolliert
// seit dem 24.09.2026, und audit_logs.admin_id zeigt auf einen echten Benutzer.
func sperrTuerAufruf(t *testing.T, pool *pgxpool.Pool, akteur, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	srv := &Server{DB: &db.Database{Pool: pool}}
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/students/"+id+"/lock", strings.NewReader(body))
	req.SetPathValue("id", id)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: akteur, Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.LockStudentHandler(repository.NewAuditRepository(pool)).ServeHTTP(rec, req)
	return rec
}

// TestLockStudentHandler_RequiresReasonAndSetsBlockReason prüft den Lock-Endpoint end-to-end:
// Sperren ohne Grund wird abgelehnt (400), Sperren mit Grund setzt is_manually_blocked und
// block_reason, Entsperren räumt beides wieder. Vorher konnte der Toggle eine grundlose
// "Zombie-Sperre" anlegen.
func TestLockStudentHandler_RequiresReasonAndSetsBlockReason(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	akteur := adminFuerAudit(t, pool)

	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		 VALUES ('LK-1', 'Max', 'Muster', '7a', 2030) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	call := func(body string) *httptest.ResponseRecorder { return sperrTuerAufruf(t, pool, akteur, id, body) }
	leseSperre := func() (bool, *string) {
		t.Helper()
		var blocked bool
		var reason *string
		if err := pool.QueryRow(ctx,
			`SELECT is_manually_blocked, block_reason FROM schueler WHERE id = $1`, id).Scan(&blocked, &reason); err != nil {
			t.Fatal(err)
		}
		return blocked, reason
	}

	// Sperren ohne Grund → 400, keine Änderung.
	if rec := call(`{"is_locked":true}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Sperre ohne Grund erwartet 400, war %d: %s", rec.Code, rec.Body.String())
	}
	if blocked, _ := leseSperre(); blocked {
		t.Error("Schüler wurde trotz fehlendem Grund gesperrt")
	}

	// Sperren mit Grund → 200, block_reason gesetzt.
	if rec := call(`{"is_locked":true,"reason":"Wiederholt nicht zurückgegeben"}`); rec.Code != http.StatusOK {
		t.Fatalf("Sperre mit Grund erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if blocked, reason := leseSperre(); !blocked || reason == nil || *reason != "Wiederholt nicht zurückgegeben" {
		t.Errorf("Sperre nicht korrekt gesetzt: blocked=%v reason=%v", blocked, reason)
	}

	// Entsperren → 200, block_reason geräumt (keine Systemsperre vorhanden).
	if rec := call(`{"is_locked":false}`); rec.Code != http.StatusOK {
		t.Fatalf("Entsperren erwartet 200, war %d: %s", rec.Code, rec.Body.String())
	}
	if blocked, reason := leseSperre(); blocked || reason != nil {
		t.Errorf("Entsperren nicht korrekt: blocked=%v reason=%v", blocked, reason)
	}
}

// Aufheben nimmt BEIDE Sperren weg — auch die, die das Programm den Ehemaligen setzt
// (entschieden am 24.09.2026). Sie lassen an der Theke nur die Rückgabe zu; ohne diese Tür
// fiel die Sperre der Ehemaligen erst mit dem nächsten LUSD-Import, mit offenem Buch nie.
// Sperren und Aufheben stehen im Protokoll, das Aufheben mit dem Grund, der galt.
// Rot gesehen am Rückbau: mit der Anweisung bis zum 24.09.2026 (nur die Sperre von Hand
// umschalten) bleibt die Sperre der Ehemaligen stehen; ohne protokolliereSperre fehlen beide
// Einträge.
func TestSperrTuer_HebtBeideSperrenAufUndProtokolliert(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	akteur := adminFuerAudit(t, pool)

	id := schuelerAnlegen(t, pool, "Ehemalig", "10R1", "S-SPERRTUER-1")
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true,
		block_reason = 'Automatisierte Abgänger-Sperre (offene Vorgänge)' WHERE id = $1`, id); err != nil {
		t.Fatalf("Ehemaligen sperren: %v", err)
	}

	protokoll := func(aktion string) []map[string]any {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT details FROM audit_logs
			WHERE aktion = $1 AND details->>'schueler_id' = $2 AND admin_id = $3 ORDER BY zeitstempel`, aktion, id, akteur)
		if err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		defer rows.Close()
		var out []map[string]any
		for rows.Next() {
			var roh []byte
			if err := rows.Scan(&roh); err != nil {
				t.Fatal(err)
			}
			var d map[string]any
			if err := json.Unmarshal(roh, &d); err != nil {
				t.Fatal(err)
			}
			out = append(out, d)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		return out
	}
	stand := func() (vomProgramm, vonHand bool, grund *string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT ist_gesperrt, is_manually_blocked, block_reason FROM leser WHERE id = $1`,
			id).Scan(&vomProgramm, &vonHand, &grund); err != nil {
			t.Fatal(err)
		}
		return
	}

	// Eine Sperre von Hand dazu: Sie überschreibt den Grund, die Sperre der Ehemaligen bleibt.
	if rec := sperrTuerAufruf(t, pool, akteur, id, `{"is_locked":true,"reason":"Ausweis verloren"}`); rec.Code != http.StatusOK {
		t.Fatalf("Sperren: %d %s", rec.Code, rec.Body.String())
	}
	if p, h, _ := stand(); !p || !h {
		t.Fatalf("nach dem Sperren: vomProgramm=%v vonHand=%v", p, h)
	}
	if e := protokoll("LESER_GESPERRT"); len(e) != 1 || e[0]["grund"] != "Ausweis verloren" {
		t.Errorf("LESER_GESPERRT: %v", e)
	}

	// Aufheben nimmt beide Sperren und den Grund weg.
	rec := sperrTuerAufruf(t, pool, akteur, id, `{"is_locked":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Aufheben: %d %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		IsManuallyBlocked bool `json:"is_manually_blocked"`
		IstGesperrt       bool `json:"ist_gesperrt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || antwort.IsManuallyBlocked || antwort.IstGesperrt {
		t.Errorf("die Antwort nennt noch eine Sperre: %s (%v)", rec.Body.String(), err)
	}
	if p, h, g := stand(); p || h || g != nil {
		t.Errorf("nach dem Aufheben: vomProgramm=%v vonHand=%v grund=%v", p, h, g)
	}
	e := protokoll("LESER_ENTSPERRT")
	if len(e) != 1 || e[0]["von_hand"] != true || e[0]["vom_programm"] != true || e[0]["grund"] != "Ausweis verloren" {
		t.Errorf("LESER_ENTSPERRT: %v", e)
	}

	// Aufheben ohne Sperre ändert nichts und schreibt nichts.
	if rec := sperrTuerAufruf(t, pool, akteur, id, `{"is_locked":false}`); rec.Code != http.StatusOK {
		t.Fatalf("zweites Aufheben: %d %s", rec.Code, rec.Body.String())
	}
	if e := protokoll("LESER_ENTSPERRT"); len(e) != 1 {
		t.Errorf("Aufheben ohne Sperre steht im Protokoll: %v", e)
	}
}

// Zwei Zustände, die keine Sperre im Sinne der Theke sind, schaltet die Tür nicht um: der
// Papierkorb (zurück über Wiederherstellen) und die Anonymisierung (keine Person mehr).
// Rot gesehen am Rückbau: ohne die zwei Fälle antwortet die Tür 200 und schaltet um.
func TestSperrTuer_PapierkorbUndAnonymisiertBleiben(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	akteur := adminFuerAudit(t, pool)

	anonym := schuelerAnlegen(t, pool, "Anonym", "10R1", "S-SPERRTUER-A")
	if _, err := pool.Exec(ctx, `UPDATE schueler SET ist_abgaenger = true, ist_gesperrt = true,
		block_reason = 'Abgänger anonymisiert', anonymized_at = NOW() WHERE id = $1`, anonym); err != nil {
		t.Fatalf("anonymisieren: %v", err)
	}
	geloescht := schuelerAnlegen(t, pool, "Papierkorb", "07A", "S-SPERRTUER-P")
	if _, err := pool.Exec(ctx, `UPDATE schueler SET deleted_at = NOW(), ist_gesperrt = true,
		block_reason = 'Systematisch gelöscht' WHERE id = $1`, geloescht); err != nil {
		t.Fatalf("löschen: %v", err)
	}

	for name, id := range map[string]string{"anonymisiert": anonym, "Papierkorb": geloescht} {
		for _, body := range []string{`{"is_locked":false}`, `{"is_locked":true,"reason":"x"}`} {
			rec := sperrTuerAufruf(t, pool, akteur, id, body)
			if rec.Code != http.StatusConflict {
				t.Errorf("%s, %s: Status %d, erwartet 409 — %s", name, body, rec.Code, rec.Body.String())
			}
		}
		var gesperrt, vonHand bool
		if err := pool.QueryRow(ctx, `SELECT ist_gesperrt, is_manually_blocked FROM leser WHERE id = $1`, id).
			Scan(&gesperrt, &vonHand); err != nil {
			t.Fatal(err)
		}
		if !gesperrt || vonHand {
			t.Errorf("%s: umgeschaltet trotz 409 (ist_gesperrt=%v, von Hand=%v)", name, gesperrt, vonHand)
		}
	}
}
