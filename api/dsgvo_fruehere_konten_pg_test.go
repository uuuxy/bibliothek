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
)

// Die Auskunft findet ein gelöschtes Zugangskonto (29.09.2026, docs/OFFEN.md 5.19).
//
// Nach dem Löschen eines Kontos führt kein Fremdschlüssel mehr vom Leser zum Konto: benutzer
// ist weg, und jeder Verweis auf benutzer steht auf SET NULL. Die Auskunft der Person, deren
// Leserzeile stehen blieb, nannte das Konto und die Einträge darüber deshalb nicht mehr. Ein
// Konto fällt über zwei Türen — Benutzer & Rechte und die Leserzeile eines Kollegen im
// Papierkorb —, und beide schreiben seither denselben Löscheintrag mit der Leserkennung.
// Geprüft über die echten Handler und Repository-Wege; dazu, dass das endgültige Löschen des
// Lesers Name und Adresse aus dem Löscheintrag tilgt.
func TestDsgvoAuskunft_FindetGeloeschteZugangskonten(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	admin := adminFuerAudit(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM benutzer WHERE email LIKE '%@fruehere-konten.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
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
	auskunft := func(leserID string) DsgvoAuskunftResponse {
		t.Helper()
		rec := alsAdmin(srv.DsgvoAuskunftHandler(), http.MethodGet, "/api/schueler/"+leserID+"/dsgvo-auskunft", "",
			map[string]string{"id": leserID})
		if rec.Code != http.StatusOK {
			t.Fatalf("Auskunft: Status %d — %s", rec.Code, rec.Body.String())
		}
		var a DsgvoAuskunftResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
			t.Fatalf("Auskunft lesen: %v", err)
		}
		return a
	}

	// Tür 1: Benutzer & Rechte. Anlegen und Ändern schreiben USER_CREATE und USER_UPDATE mit
	// ziel_id; der Ausweis hält die Leserzeile, wenn das Konto fällt.
	if rec := alsAdmin(srv.CreateUserHandler(userRepo), http.MethodPost, "/api/benutzer",
		`{"barcode_id":"FK-AUSWEIS-1","vorname":"Frieda","nachname":"Frueher","email":"frieda@fruehere-konten.invalid","rolle":"mitarbeiter"}`,
		nil); rec.Code != http.StatusOK {
		t.Fatalf("Konto anlegen: Status %d — %s", rec.Code, rec.Body.String())
	}
	var kontoID, leserID string
	if err := pool.QueryRow(ctx, `SELECT id::text, leser_id::text FROM benutzer WHERE email = 'frieda@fruehere-konten.invalid'`).
		Scan(&kontoID, &leserID); err != nil {
		t.Fatalf("Konto lesen: %v", err)
	}
	if rec := alsAdmin(srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+kontoID,
		`{"barcode_id":"FK-AUSWEIS-1","vorname":"Frieda","nachname":"Frueher","email":"frieda@fruehere-konten.invalid","rolle":"helfer","aktiv":true}`,
		map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
		t.Fatalf("Konto ändern: Status %d — %s", rec.Code, rec.Body.String())
	}
	if rec := alsAdmin(srv.DeleteUserHandler(auditRepo, userRepo), http.MethodDelete, "/api/benutzer/"+kontoID, "",
		map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
		t.Fatalf("Konto löschen: Status %d — %s", rec.Code, rec.Body.String())
	}

	a := auskunft(leserID)
	if a.Zugangskonto != nil {
		t.Fatalf("das gelöschte Konto steht als bestehendes Konto in der Auskunft: %+v", a.Zugangskonto)
	}
	if len(a.FruehereZugangskonten) != 1 {
		t.Fatalf("frühere Zugangskonten: %d, erwartet 1 — %+v", len(a.FruehereZugangskonten), a.FruehereZugangskonten)
	}
	frueher := a.FruehereZugangskonten[0]
	if frueher.ID != kontoID || frueher.Email != "frieda@fruehere-konten.invalid" || frueher.Rolle != "helfer" {
		t.Errorf("früheres Konto: %+v", frueher)
	}
	aktionen := map[string]bool{}
	for _, e := range frueher.Ereignisse {
		aktionen[e.Aktion] = true
	}
	if !aktionen["USER_CREATE"] || !aktionen["USER_UPDATE"] {
		t.Errorf("die Einträge über das gelöschte Konto fehlen in der Auskunft: %v", aktionen)
	}

	// Geht die unberührte Leserzeile mit dem Konto, trägt der Löscheintrag keine Leserkennung:
	// Es gibt keinen Leser mehr, und ein Name neben einem verwaisten Bezug tilgte niemand. Das
	// ist die abgelehnte Zugangsanfrage: Ein nicht freigeschaltetes Konto hat keine
	// Ausweisnummer (Migration 136/145 vergibt sie nur aktiven Konten), seine Leserzeile ist
	// unberührt (loescheUnberuehrteLeserzeile).
	var olga string
	if err := pool.QueryRow(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
		VALUES ('Olga', 'Anfrage', 'olga@fruehere-konten.invalid', 'kollegium', false, NOW())
		RETURNING id::text`).Scan(&olga); err != nil {
		t.Fatalf("Zugangsanfrage anlegen: %v", err)
	}
	if rec := alsAdmin(srv.DeleteUserHandler(auditRepo, userRepo), http.MethodDelete, "/api/benutzer/"+olga, "",
		map[string]string{"id": olga}); rec.Code != http.StatusOK {
		t.Fatalf("Zugangsanfrage ablehnen (Konto löschen): Status %d — %s", rec.Code, rec.Body.String())
	}
	var mitLeser, zeileGing bool
	if err := pool.QueryRow(ctx, `SELECT details ? 'schueler_id', (details->>'leserzeile_geloescht')::boolean
		FROM audit_log WHERE tabelle = 'benutzer' AND aktion = 'DELETE' AND datensatz_id = $1::uuid`, olga).
		Scan(&mitLeser, &zeileGing); err != nil {
		t.Fatalf("Löscheintrag der abgelehnten Anfrage lesen: %v", err)
	}
	if !zeileGing || mitLeser {
		t.Errorf("Leserzeile mitgelöscht = %v, Leserkennung im Eintrag = %v — erwartet true und false", zeileGing, mitLeser)
	}

	// Tür 2: Die Leserzeile eines Kollegen geht in den Papierkorb, sein Konto fällt dabei mit.
	var kollege string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (art, vorname, nachname) VALUES ('lehrkraft', 'Paul', 'Papierkorb')
		RETURNING id::text`).Scan(&kollege); err != nil {
		t.Fatalf("Kollege anlegen: %v", err)
	}
	if _, err := repository.LegeKollegiumskonto(ctx, pool, repository.LegeKollegiumskontoParams{
		Vorname: "Paul", Nachname: "Papierkorb", Email: "paul@fruehere-konten.invalid", LeserID: kollege, Aktiv: true,
	}); err != nil {
		t.Fatalf("Konto des Kollegen anlegen: %v", err)
	}
	if err := auditRepo.DeleteStudent(ctx, kollege, admin, "Test"); err != nil {
		t.Fatalf("Kollegen löschen: %v", err)
	}
	a = auskunft(kollege)
	if len(a.FruehereZugangskonten) != 1 || a.FruehereZugangskonten[0].Email != "paul@fruehere-konten.invalid" {
		t.Errorf("das mit der Leserzeile gelöschte Konto fehlt in der Auskunft: %+v", a.FruehereZugangskonten)
	}

	// Beim endgültigen Löschen des Lesers fallen Name und Adresse aus dem Löscheintrag; der
	// Eintrag selbst bleibt (Rolle, Kennung als Pseudonym).
	if err := auditRepo.PurgeStudent(ctx, kollege, admin); err != nil {
		t.Fatalf("Kollegen endgültig löschen: %v", err)
	}
	var details map[string]any
	var roh []byte
	if err := pool.QueryRow(ctx, `SELECT details FROM audit_log
		WHERE tabelle = 'benutzer' AND aktion = 'DELETE' AND details->>'schueler_id' = $1`, kollege).Scan(&roh); err != nil {
		t.Fatalf("Löscheintrag nach dem endgültigen Löschen: %v", err)
	}
	if err := json.Unmarshal(roh, &details); err != nil {
		t.Fatalf("Löscheintrag lesen: %v", err)
	}
	for _, k := range []string{"vorname", "nachname", "email"} {
		if _, da := details[k]; da {
			t.Errorf("%s steht nach dem endgültigen Löschen noch im Löscheintrag: %v", k, details)
		}
	}
	if details["rolle"] != "kollegium" {
		t.Errorf("die Rolle fehlt im Löscheintrag: %v", details)
	}
}
