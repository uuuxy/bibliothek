package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Einstellungen speichern, über die Tür: Ein leerer Patch und unlesbare Angaben werden
// abgelehnt und schreiben weder Wert noch Protokoll. Was gespeichert wird, steht in Normalform
// in der Datenbank und mit demselben Wortlaut im Protokoll.
func TestEinstellungenSpeichern_NormalformUndProtokoll(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var admin string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Ada', 'Einstellung', 'ada@einstellungen.invalid', 'admin', true) RETURNING id::text`).Scan(&admin); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	wert := func(schluessel string) string {
		t.Helper()
		var w string
		if err := pool.QueryRow(ctx, `
			SELECT coalesce((SELECT wert FROM system_einstellungen WHERE schluessel = $1), '')`, schluessel).Scan(&w); err != nil {
			t.Fatalf("Einstellung %s lesen: %v", schluessel, err)
		}
		return w
	}
	// Die Werte liegen in einer Tabelle, die kein Test leert: Der Stand davor kommt zurück.
	vorher := map[string]string{"sommerferien": wert("sommerferien"), "lmf_eingangsjahrgaenge": wert("lmf_eingangsjahrgaenge")}
	t.Cleanup(func() {
		for schluessel, alt := range vorher {
			if _, err := pool.Exec(context.Background(), `DELETE FROM system_einstellungen WHERE schluessel = $1`, schluessel); err != nil {
				t.Errorf("Einstellung %s zurücknehmen: %v", schluessel, err)
			}
			if alt == "" {
				continue
			}
			if _, err := pool.Exec(context.Background(),
				`INSERT INTO system_einstellungen (schluessel, wert) VALUES ($1, $2)`, schluessel, alt); err != nil {
				t.Errorf("Einstellung %s wiederherstellen: %v", schluessel, err)
			}
		}
	})
	speichere := func(rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/einstellungen/speichern", strings.NewReader(rumpf))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.UpdateSettingsHandler(repository.NewSystemSettingsRepository(pool)).ServeHTTP(rec, req)
		return rec
	}

	for _, f := range []struct {
		name, rumpf, stueck string
	}{
		{"leerer Patch", `{}`, "keine einzige Einstellung"},
		{"Sommerferien mit unmöglichem Datum",
			`{"sommerferien":"[{\"jahr\":2031,\"von\":\"2031-07-07\",\"bis\":\"2031-08-51\"}]"}`, "Sommerferien 2031"},
		{"Eingangsjahrgänge mit Schrägstrich", `{"lmf_eingangsjahrgaenge":"8/9"}`, "8/9"},
	} {
		rec := speichere(f.rumpf)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: Status %d, erwartet 400: %s", f.name, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), f.stueck) {
			t.Errorf("%s: die Antwort nennt %q nicht: %s", f.name, f.stueck, rec.Body.String())
		}
	}
	for schluessel, alt := range vorher {
		if got := wert(schluessel); got != alt {
			t.Fatalf("eine Ablehnung hat %s geschrieben: %q", schluessel, got)
		}
	}
	if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs WHERE aktion = 'UPDATE_SETTINGS'`); n != 0 {
		t.Fatalf("%d Protokolleinträge nach drei Ablehnungen", n)
	}

	rec := speichere(`{"sommerferien":" [ {\"jahr\":2032,\"von\":\"2032-07-05\",\"bis\":\"2032-08-13\"}, {\"jahr\":2031,\"von\":\"2031-07-07\",\"bis\":\"2031-08-15\"} ] ","lmf_eingangsjahrgaenge":" 7 ; 05 "}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("gültige Angaben: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	const ferien = `[{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"},{"jahr":2032,"von":"2032-07-05","bis":"2032-08-13"}]`
	if got := wert("sommerferien"); got != ferien {
		t.Errorf("Sommerferien gespeichert als %q, erwartet nach Jahr sortiert und ohne Leerraum", got)
	}
	if got := wert("lmf_eingangsjahrgaenge"); got != "5, 7" {
		t.Errorf("Eingangsjahrgänge gespeichert als %q, erwartet %q", got, "5, 7")
	}
	if n := zaehleZeilen(t, pool, `
		SELECT count(*) FROM audit_logs
		WHERE aktion = 'UPDATE_SETTINGS' AND admin_id = $1 AND ip_adresse IS NULL
		  AND details->>'sommerferien' = $2 AND details->>'lmf_eingangsjahrgaenge' = '5, 7'`, admin, ferien); n != 1 {
		t.Errorf("%d Protokolleinträge mit den gespeicherten Werten und ohne Adresse, erwartet 1", n)
	}
}
