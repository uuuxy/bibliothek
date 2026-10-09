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
	"bibliothek/internal/auskunft"
)

// Ein Konto, das über die Leserdatei entsteht, hinterlässt dieselbe Spur wie über Benutzer &
// Rechte: USER_CREATE mit ziel_id und der angemeldeten Person als Handelnder (29.09.2026,
// docs/OFFEN.md 5.19). Vorher schrieb die Neuanlage einer Lehrkraft mit Schul-E-Mail keinen
// Eintrag; zwei Türen zum selben Zustand, die Rechenschaft hing an der Tür. Über die ziel_id
// findet die Auskunft den Eintrag beim Konto.
func TestLeserdatei_KontoAnlageSchreibtUserCreate(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "")
	ctx := context.Background()
	admin := adminFuerAudit(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM audit_logs WHERE details->>'email' = 'lena@leserdatei-anlage.invalid'`,
			`DELETE FROM benutzer WHERE email = 'lena@leserdatei-anlage.invalid'`,
		} {
			if _, err := pool.Exec(context.Background(), sql); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	})

	alsAdmin := func(h http.HandlerFunc, methode, pfad, body string, pfadwerte map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(methode, pfad, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range pfadwerte {
			req.SetPathValue(k, v)
		}
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: admin, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec
	}

	if rec := alsAdmin(srv.CreateStudentHandler(), http.MethodPost, "/api/schueler",
		`{"vorname":"Lena","nachname":"Leserdatei","art":"lehrkraft","email":"lena@leserdatei-anlage.invalid"}`,
		nil); rec.Code != http.StatusCreated {
		t.Fatalf("Lehrkraft mit Zugang anlegen: Status %d — %s", rec.Code, rec.Body.String())
	}
	var kontoID, leserID string
	if err := pool.QueryRow(ctx, `SELECT id::text, leser_id::text FROM benutzer WHERE email = 'lena@leserdatei-anlage.invalid'`).
		Scan(&kontoID, &leserID); err != nil {
		t.Fatalf("das Konto ist nicht entstanden: %v", err)
	}

	var eintraege int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs
		WHERE aktion = 'USER_CREATE' AND details->>'ziel_id' = $1 AND admin_id = $2
		  AND details->>'rolle' = 'kollegium'`, kontoID, admin).Scan(&eintraege); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	if eintraege != 1 {
		t.Errorf("USER_CREATE mit ziel_id des neuen Kontos: %d Einträge, erwartet 1", eintraege)
	}

	rec := alsAdmin(srv.DsgvoAuskunftHandler(), http.MethodGet, "/api/schueler/"+leserID+"/dsgvo-auskunft", "",
		map[string]string{"id": leserID})
	if rec.Code != http.StatusOK {
		t.Fatalf("Auskunft: Status %d — %s", rec.Code, rec.Body.String())
	}
	var a auskunft.DsgvoAuskunftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("Auskunft lesen: %v", err)
	}
	if a.Zugangskonto == nil {
		t.Fatal("die Auskunft nennt das Konto nicht")
	}
	gefunden := false
	for _, e := range a.Zugangskonto.Ereignisse {
		gefunden = gefunden || e.Aktion == "USER_CREATE"
	}
	if !gefunden {
		t.Errorf("die Anlage steht nicht unter den Einträgen zum Konto: %+v", a.Zugangskonto.Ereignisse)
	}
}
