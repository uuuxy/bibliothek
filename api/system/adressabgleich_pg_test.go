package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/crypto"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"
)

// abgleichVCF ist das globale Adressbuch des nachgebauten Webmails, mit erfundenen Namen.
const abgleichVCF = "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Abgleich;Adria;;;AAB\r\nEMAIL:adria@schule.example\r\nEND:VCARD\r\n" +
	"BEGIN:VCARD\r\nN:Abgleich;Bruno;;;BA1\r\nEMAIL:bruno1@schule.example\r\nEND:VCARD\r\n" +
	"BEGIN:VCARD\r\nN:Abgleich;Bruno;;;BA2\r\nEMAIL:bruno2@schule.example\r\nEND:VCARD\r\n" +
	"BEGIN:VCARD\r\nN:Abgleich;Cora;;;CAB\r\nEMAIL:cora@schule.example\r\nEND:VCARD\r\n" +
	"BEGIN:VCARD\r\nN:Abgleich;Dana;;;DAB\r\nEMAIL:dana@schule.example\r\nEND:VCARD\r\n"

// abgleichWebmail spielt Anmeldung, Kontakte und Export von Roundcube nach; nur das Passwort
// „richtig" kommt hinein.
func abgleichWebmail(t *testing.T) string {
	t.Helper()
	schreibe := func(w http.ResponseWriter, rumpf string) {
		if _, err := io.WriteString(w, rumpf); err != nil {
			t.Errorf("Antwort des Nachbaus: %v", err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		_, angemeldet := r.Cookie("roundcube_sessauth")
		formular := `<input type="hidden" name="_token" value="T1"><input name="_pass">`
		switch {
		case q.Get("_task") == "login" && r.Method == http.MethodGet:
			schreibe(w, formular)
		case q.Get("_task") == "login":
			if r.FormValue("_pass") != "richtig" {
				w.WriteHeader(http.StatusUnauthorized)
				schreibe(w, formular)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "roundcube_sessauth", Value: "ok", Path: "/"})
			http.Redirect(w, r, "./?_task=mail", http.StatusFound)
		case q.Get("_task") == "addressbook" && q.Get("_action") == "" && angemeldet == nil:
			schreibe(w, `rcmail.set_env({"request_token":"R1","address_sources":[{"id":"global","name":"Globales Adressbuch"}]});`)
		case q.Get("_action") == "export" && angemeldet == nil && q.Get("_token") == "R1":
			schreibe(w, abgleichVCF)
		default:
			schreibe(w, formular)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestAdressabgleich_ThroughTheDoors(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, "abgleich-test-schluessel-32-zeic")
	t.Setenv("SELBSTANMELDUNG_DOMAIN", "schule.example")
	pool := pgtest.Pool(t)
	ctx := context.Background()
	mailAbfragen := repository.NewMailEinstellungsAbfragen(pool)
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}

	adria := testdaten.LegeKontoAn(t, pool, "Adria", "Abgleich", "littera-9001@littera.invalid", "kollegium", true)
	bruno := testdaten.LegeKontoAn(t, pool, "Bruno", "Abgleich", "littera-9002@littera.invalid", "kollegium", true)
	cora := testdaten.LegeKontoAn(t, pool, "Cora", "Abgleich", "cora.alt@schule.example", "kollegium", true)
	dana := testdaten.LegeKontoAn(t, pool, "Dana", "Abgleich", "littera-9003@littera.invalid", "kollegium", true)
	selbst := testdaten.LegeKontoAn(t, pool, "D.", "Selbst", "dana@schule.example", "kollegium", false)
	konten := []string{adria, bruno, cora, dana, selbst}
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_logs WHERE details->>'ziel_id' = ANY($1::text[])`, konten)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM benutzer WHERE id = ANY($1::uuid[])`, konten)
		pgtest.RaeumeAuf(t, pool, `UPDATE mail_settings_config SET webmail_adresse = '', webmail_benutzer = '',
			webmail_passwort_verschluesselt = NULL WHERE id = 1`)
		pgtest.RaeumeAuf(t, pool, `DELETE FROM system_einstellungen WHERE schluessel = $1`, adressabgleichStandSchluessel)
	})
	email := func(id string) string {
		t.Helper()
		var e string
		if err := pool.QueryRow(ctx, `SELECT email FROM benutzer WHERE id = $1`, id).Scan(&e); err != nil {
			t.Fatalf("Adresse von %s lesen: %v", id, err)
		}
		return e
	}
	tuerAufruf := func(handler http.HandlerFunc, methode, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(methode, "/api/admin/settings/adressabgleich", strings.NewReader(rumpf)))
		return rec
	}
	lauf := func(probelauf bool) (int, adressabgleichErgebnis) {
		t.Helper()
		rec := tuerAufruf(srv.AdressabgleichLauf(), http.MethodPost, fmt.Sprintf(`{"probelauf":%t}`, probelauf))
		var ergebnis adressabgleichErgebnis
		if err := json.Unmarshal(rec.Body.Bytes(), &ergebnis); err != nil {
			t.Fatalf("Antwort des Laufs kein JSON: %v, %s", err, rec.Body.String())
		}
		return rec.Code, ergebnis
	}

	// Ohne Zugang ist nichts eingerichtet, und ein Lauf ohne Zugang gilt nicht als gescheitert.
	if code, _ := lauf(true); code != http.StatusConflict {
		t.Errorf("Probelauf ohne Zugang: Status %d, erwartet 409", code)
	}
	if code, _ := lauf(false); code != http.StatusConflict || srv.ladeAdressabgleichStand(ctx) != nil {
		t.Errorf("Abgleich ohne Zugang: Status %d, Stand %+v, erwartet 409 ohne gespeicherten Stand", code, srv.ladeAdressabgleichStand(ctx))
	}
	// Speichern ohne Passwort, solange keins hinterlegt ist, wird abgelehnt.
	adresse := abgleichWebmail(t)
	if rec := tuerAufruf(srv.AdressabgleichZugangSchreiben(mailAbfragen), http.MethodPut,
		`{"adresse":"`+adresse+`/?_task=mail","benutzer":"bibliothek@schule.example","passwort":""}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Zugang ohne Passwort: Status %d, erwartet 400", rec.Code)
	}
	if rec := tuerAufruf(srv.AdressabgleichZugangSchreiben(mailAbfragen), http.MethodPut,
		`{"adresse":"`+adresse+`/?_task=mail","benutzer":"bibliothek@schule.example","passwort":"richtig"}`); rec.Code != http.StatusOK {
		t.Fatalf("Zugang speichern: Status %d, %s", rec.Code, rec.Body.String())
	}

	// Der Probelauf zeigt die Übernahme und ändert nichts.
	code, probe := lauf(true)
	if code != http.StatusOK || len(probe.Uebernahmen) != 1 || probe.Uebernahmen[0].KontoID != adria {
		t.Fatalf("Probelauf: Status %d, Übernahmen %+v, erwartet nur Adria", code, probe.Uebernahmen)
	}
	if got := email(adria); got != "littera-9001@littera.invalid" {
		t.Errorf("nach dem Probelauf trägt Adria %q, erwartet den Platzhalter", got)
	}
	if srv.ladeAdressabgleichStand(ctx) != nil {
		t.Error("der Probelauf hat einen Stand gespeichert")
	}

	// Der echte Lauf trägt nur Adria ein; Bruno (Name doppelt), Cora (echte Adresse) und Dana
	// (Adresse vergeben) bleiben.
	if code, _ := lauf(false); code != http.StatusOK {
		t.Fatalf("Abgleich: Status %d", code)
	}
	for id, want := range map[string]string{
		adria: "adria@schule.example", bruno: "littera-9002@littera.invalid",
		cora: "cora.alt@schule.example", dana: "littera-9003@littera.invalid",
	} {
		if got := email(id); got != want {
			t.Errorf("Konto %s trägt nach dem Abgleich %q, erwartet %q", id, got, want)
		}
	}
	var spuren int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE aktion = 'ADRESSE_AUS_WEBMAIL'
		AND details->>'ziel_id' = $1 AND details->>'email' = 'adria@schule.example'`, adria).Scan(&spuren); err != nil || spuren != 1 {
		t.Errorf("Protokolleinträge der Übernahme: %d (%v), erwartet 1", spuren, err)
	}
	stand := srv.ladeAdressabgleichStand(ctx)
	if stand == nil || stand.Uebernommen != 1 || stand.Fehler != "" {
		t.Errorf("gespeicherter Stand %+v, erwartet eine Übernahme ohne Fehler", stand)
	}

	// Ein zweiter Lauf findet nichts mehr, und die leere Liste geht als [] hinaus: Die Maske zählt
	// ihre Einträge.
	rec := tuerAufruf(srv.AdressabgleichLauf(), http.MethodPost, `{"probelauf":false}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"uebernahmen":[]`) {
		t.Errorf("zweiter Lauf: %d %s, erwartet 200 mit leerer Liste der Übernahmen", rec.Code, rec.Body.String())
	}

	// Ein falsches Passwort meldet das Webmail als ablehnend und landet im Stand.
	if rec := tuerAufruf(srv.AdressabgleichZugangSchreiben(mailAbfragen), http.MethodPut,
		`{"adresse":"`+adresse+`","benutzer":"bibliothek@schule.example","passwort":"falsch"}`); rec.Code != http.StatusOK {
		t.Fatalf("Zugang ändern: Status %d", rec.Code)
	}
	rec = tuerAufruf(srv.AdressabgleichLauf(), http.MethodPost, `{"probelauf":false}`)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "Anmeldung abgelehnt") {
		t.Errorf("Abgleich mit falschem Passwort: %d %s, erwartet 502 mit der Ablehnung", rec.Code, rec.Body.String())
	}
	if stand := srv.ladeAdressabgleichStand(ctx); stand == nil || stand.Fehler != meldungWebmailAbgelehnt {
		t.Errorf("Stand nach der Ablehnung %+v, erwartet den Satz der Ablehnung", stand)
	}
}
