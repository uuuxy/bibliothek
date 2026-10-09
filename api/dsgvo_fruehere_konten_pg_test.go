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
	"bibliothek/internal/auskunft"
	"bibliothek/internal/pdftest"
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
// fkAlsAdmin ruft einen Handler mit der Sitzung des Admin-Kontos.
func fkAlsAdmin(t *testing.T, admin string, h http.HandlerFunc, methode, pfad, body string, pfadwerte map[string]string) *httptest.ResponseRecorder {
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

// fkAuskunft ruft die Auskunft eines Lesers über ihren Handler ab.
func fkAuskunft(t *testing.T, srv *Server, admin, leserID string) auskunft.DsgvoAuskunftResponse {
	t.Helper()
	rec := fkAlsAdmin(t, admin, srv.DsgvoAuskunftHandler(), http.MethodGet, "/api/schueler/"+leserID+"/dsgvo-auskunft", "",
		map[string]string{"id": leserID})
	if rec.Code != http.StatusOK {
		t.Fatalf("Auskunft: Status %d — %s", rec.Code, rec.Body.String())
	}
	var a auskunft.DsgvoAuskunftResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("Auskunft lesen: %v", err)
	}
	return a
}

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
		return fkAlsAdmin(t, admin, h, methode, pfad, body, pfadwerte)
	}
	auskunftZu := func(leserID string) auskunft.DsgvoAuskunftResponse {
		t.Helper()
		return fkAuskunft(t, srv, admin, leserID)
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

	a := auskunftZu(leserID)
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
	a = auskunftZu(kollege)
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

// Die eigene Anmeldung und der Nachtrag der Schul-E-Mail stehen beim Konto, auch nachdem es
// gelöscht ist. Beide Einträge trugen die Kennung des Kontos nicht in den Details: die
// Selbstanmeldung nur als Handelnden (admin_id, mit dem Konto geleert), der Nachtrag nur die
// Kennung des Lesers. Geprüft über die Anmeldung, die Leserakte und Benutzer & Rechte.
func TestDsgvoAuskunft_NenntAnmeldungUndNachtragBeimKonto(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("IMAP_HOST", "mock")
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "konto-eintraege.invalid")
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	admin := adminFuerAudit(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM benutzer WHERE email LIKE '%@konto-eintraege.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	konto := func(t *testing.T, email string) (kontoID, leserID string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT id::text, leser_id::text FROM benutzer WHERE email = $1`, email).
			Scan(&kontoID, &leserID); err != nil {
			t.Fatalf("Konto %s lesen: %v", email, err)
		}
		return kontoID, leserID
	}
	aktionen := func(ereignisse []repository.DsgvoKontoEreignis) map[string]bool {
		m := map[string]bool{}
		for _, e := range ereignisse {
			m[e.Aktion] = true
		}
		return m
	}
	// blatt liest den Text der gedruckten Auskunft.
	blatt := func(t *testing.T, leserID string) string {
		t.Helper()
		rec := fkAlsAdmin(t, admin, srv.DsgvoAuskunftPDFHandler(), http.MethodGet, "/api/schueler/"+leserID+"/dsgvo-auskunft/pdf", "",
			map[string]string{"id": leserID})
		if rec.Code != http.StatusOK {
			t.Fatalf("Auskunft als PDF: Status %d — %s", rec.Code, rec.Body.String())
		}
		return strings.Join(pdftest.Texte(t, rec.Body.Bytes()), "\n")
	}
	loescheKonto := func(t *testing.T, kontoID string) {
		t.Helper()
		if rec := fkAlsAdmin(t, admin, srv.DeleteUserHandler(auditRepo, userRepo), http.MethodDelete, "/api/benutzer/"+kontoID, "",
			map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
			t.Fatalf("Konto löschen: Status %d — %s", rec.Code, rec.Body.String())
		}
	}

	t.Run("eigene Anmeldung", func(t *testing.T) {
		const email = "selma.selbst@konto-eintraege.invalid"
		authenticator, err := auth.NewAuthenticator("konto-eintraege-testgeheimnis-32-bytes!!", pool, time.Hour)
		if err != nil {
			t.Fatalf("Authenticator: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"`+email+`","password":"beliebig"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		auth.LoginHandler(pool, authenticator, false)(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("erste Anmeldung: Status %d, erwartet 403 (Zugang beantragt) — %s", rec.Code, rec.Body.String())
		}
		kontoID, leserID := konto(t, email)
		// Freischalten gibt dem Konto eine Ausweisnummer; sie hält die Leserzeile, wenn das Konto fällt.
		if rec := fkAlsAdmin(t, admin, srv.UpdateUserHandler(userRepo), http.MethodPut, "/api/benutzer/"+kontoID,
			`{"barcode_id":"","vorname":"Selma","nachname":"Selbst","email":"`+email+`","rolle":"kollegium","aktiv":true}`,
			map[string]string{"id": kontoID}); rec.Code != http.StatusOK {
			t.Fatalf("Konto freischalten: Status %d — %s", rec.Code, rec.Body.String())
		}

		a := fkAuskunft(t, srv, admin, leserID)
		if a.Zugangskonto == nil || !aktionen(a.Zugangskonto.Ereignisse)["SELBSTANMELDUNG"] {
			t.Fatalf("die eigene Anmeldung fehlt beim bestehenden Konto: %+v", a.Zugangskonto)
		}
		loescheKonto(t, kontoID)

		a = fkAuskunft(t, srv, admin, leserID)
		if len(a.FruehereZugangskonten) != 1 {
			t.Fatalf("frühere Zugangskonten: %d, erwartet 1", len(a.FruehereZugangskonten))
		}
		if !aktionen(a.FruehereZugangskonten[0].Ereignisse)["SELBSTANMELDUNG"] {
			t.Errorf("die eigene Anmeldung fehlt beim gelöschten Konto: %v", aktionen(a.FruehereZugangskonten[0].Ereignisse))
		}
		if !strings.Contains(blatt(t, leserID), "Zugang über die eigene Anmeldung beantragt") {
			t.Error("das PDF nennt die eigene Anmeldung beim gelöschten Konto nicht")
		}
	})

	t.Run("Schul-E-Mail in der Leserakte nachgetragen", func(t *testing.T) {
		const email = "nora.nachtrag@konto-eintraege.invalid"
		var leserID string
		if err := pool.QueryRow(ctx, `INSERT INTO leser (art, vorname, nachname, barcode_id)
			VALUES ('lehrkraft', 'Nora', 'Nachtrag', 'KE-AUSWEIS-1') RETURNING id::text`).Scan(&leserID); err != nil {
			t.Fatalf("Leser anlegen: %v", err)
		}
		if rec := fkAlsAdmin(t, admin, srv.PatchStudentHandler(auditRepo), http.MethodPatch, "/api/schueler/"+leserID,
			`{"vorname":"Nora","nachname":"Nachtrag","email":"`+email+`"}`, map[string]string{"id": leserID}); rec.Code != http.StatusOK {
			t.Fatalf("Schul-E-Mail nachtragen: Status %d — %s", rec.Code, rec.Body.String())
		}
		kontoID, kontoLeser := konto(t, email)
		if kontoLeser != leserID {
			t.Fatalf("das Konto hängt an Leser %s, erwartet %s", kontoLeser, leserID)
		}

		a := fkAuskunft(t, srv, admin, leserID)
		if a.Zugangskonto == nil || !aktionen(a.Zugangskonto.Ereignisse)["KOLLEGIUMSKONTO_NACHGETRAGEN"] {
			t.Errorf("der Nachtrag fehlt beim bestehenden Konto: %+v", a.Zugangskonto)
		}
		loescheKonto(t, kontoID)

		a = fkAuskunft(t, srv, admin, leserID)
		if len(a.FruehereZugangskonten) != 1 {
			t.Fatalf("frühere Zugangskonten: %d, erwartet 1", len(a.FruehereZugangskonten))
		}
		if !aktionen(a.FruehereZugangskonten[0].Ereignisse)["KOLLEGIUMSKONTO_NACHGETRAGEN"] {
			t.Errorf("der Nachtrag fehlt beim gelöschten Konto: %v", aktionen(a.FruehereZugangskonten[0].Ereignisse))
		}
		if !strings.Contains(blatt(t, leserID), "Konto angelegt: Schul-E-Mail in der Leserakte nachgetragen") {
			t.Error("das PDF beschriftet den Nachtrag beim gelöschten Konto nicht")
		}

		// Mit dem Leser fällt die Verknüpfung zum Konto auch im Eintrag: Er nennt danach nur
		// noch, dass an dieser Leserzeile ein Konto nachgetragen wurde.
		if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = NOW() WHERE id = $1`, leserID); err != nil {
			t.Fatalf("in den Papierkorb legen: %v", err)
		}
		if err := auditRepo.PurgeStudent(ctx, leserID, admin); err != nil {
			t.Fatalf("endgültig löschen: %v", err)
		}
		if n := zaehleZeilen(t, pool, `SELECT count(*) FROM audit_logs
			WHERE aktion = 'KOLLEGIUMSKONTO_NACHGETRAGEN' AND details->>'schueler_id' = $1 AND NOT details ? 'ziel_id'`, leserID); n != 1 {
			t.Errorf("%d Einträge des Nachtrags ohne die Kennung des Kontos nach dem endgültigen Löschen, erwartet 1", n)
		}
	})
}
