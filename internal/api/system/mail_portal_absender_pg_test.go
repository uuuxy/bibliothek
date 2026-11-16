package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/crypto"
	"bibliothek/internal/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/repository"

	"github.com/google/go-cmp/cmp"
)

// Der zweite Absender über die Tür: Die Maske speichert und liest ihn samt den Arten von Mails,
// die über ihn gehen, und ein Rumpf ohne das Feld portal oder ohne fuer lässt stehen, was da ist.
// So löscht eine Seite, die vor dem Update geladen wurde, beim Speichern nicht, was eine neuere
// eingetragen hat.
func TestMailEinstellungen_PortalAbsenderUeberDieTuer(t *testing.T) {
	t.Setenv(crypto.SchluesselVariable, "mail-test-schluessel-32-zeichen!")
	pool := pgtest.Pool(t)
	repo := repository.NewMailEinstellungsAbfragen(pool)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `UPDATE mail_settings_config
			SET smtp_host = 'localhost', smtp_port = '1025', smtp_user = '',
			    smtp_password_encrypted = NULL, sender_email = 'noreply@bibliothek-schule.de',
			    portal_absender_email = '', portal_smtp_benutzer = '',
			    portal_smtp_passwort_verschluesselt = NULL, portal_absender_fuer = '{portal}'
			WHERE id = 1`)
	})
	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	speichere := func(rumpf string) {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.MailEinstellungenSchreiben(repo)(rec,
			httptest.NewRequest(http.MethodPut, "/api/admin/settings/mail", strings.NewReader(rumpf)))
		if rec.Code != http.StatusOK {
			t.Fatalf("Rumpf %s: Status %d — %s, erwartet 200", rumpf, rec.Code, rec.Body.String())
		}
	}
	lies := func() mailEinstellungenAntwort {
		t.Helper()
		rec := httptest.NewRecorder()
		srv.MailEinstellungen(repo)(rec, httptest.NewRequest(http.MethodGet, "/api/admin/settings/mail", nil))
		var antwort mailEinstellungenAntwort
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Antwort kein JSON: %v — %s", err, rec.Body.String())
		}
		return antwort
	}

	speichere(`{"smtp_host":"smtp.schule.invalid","smtp_port":"587","smtp_user":"bibliothek","smtp_password":"",` +
		`"sender_email":"bibliothek@schule.invalid",` +
		`"portal":{"sender_email":" portal@schule.invalid ","smtp_user":" portal ","smtp_password":"Portal-1!"}}`)
	gelesen := lies()
	want := mailPortalAntwort{AbsenderEmail: "portal@schule.invalid", SMTPBenutzer: "portal", HatPasswort: true,
		Fuer: []string{"portal"}, Arten: []string{"portal", "mahnliste", "abgaenger", "bestellung"}}
	if diff := cmp.Diff(want, gelesen.Portal); diff != "" {
		t.Errorf("gelesen (-erwartet +ist):\n%s", diff)
	}

	// Ohne das Feld portal bleibt der Absender stehen.
	speichere(`{"smtp_host":"smtp.schule.invalid","smtp_port":"587","smtp_user":"bibliothek","smtp_password":"",` +
		`"sender_email":"bibliothek@schule.invalid"}`)
	if diff := cmp.Diff(gelesen.Portal, lies().Portal); diff != "" {
		t.Errorf("ohne portal im Rumpf verändert (-vorher +nachher):\n%s", diff)
	}

	// Die Arten stehen in fester Reihenfolge, jede einmal; ohne fuer bleiben sie stehen.
	const zweiter = `"portal":{"sender_email":"portal@schule.invalid","smtp_user":"portal","smtp_password":""`
	const postausgang = `{"smtp_host":"smtp.schule.invalid","smtp_port":"587","smtp_user":"bibliothek","smtp_password":"",` +
		`"sender_email":"bibliothek@schule.invalid",`
	speichere(postausgang + zweiter + `,"fuer":["bestellung","mahnliste","bestellung"]}}`)
	if got := lies().Portal.Fuer; !cmp.Equal(got, []string{"mahnliste", "bestellung"}) {
		t.Errorf("fuer gelesen als %q, erwartet [mahnliste bestellung]", got)
	}
	// In der Zeile steht dieselbe feste Form, nicht erst beim Lesen.
	if zeile, err := repo.Lade(t.Context()); err != nil || !cmp.Equal(zeile.PortalAbsenderFuer, []string{"mahnliste", "bestellung"}) {
		t.Errorf("fuer in der Zeile %q (%v), erwartet [mahnliste bestellung]", zeile.PortalAbsenderFuer, err)
	}
	speichere(postausgang + zweiter + `}}`)
	if got := lies().Portal.Fuer; !cmp.Equal(got, []string{"mahnliste", "bestellung"}) {
		t.Errorf("ohne fuer: %q, erwartet unverändert [mahnliste bestellung]", got)
	}

	// Eine unbekannte Art lehnt die Tür ab und speichert nichts.
	rec := httptest.NewRecorder()
	srv.MailEinstellungenSchreiben(repo)(rec, httptest.NewRequest(http.MethodPut, "/api/admin/settings/mail",
		strings.NewReader(postausgang+zweiter+`,"fuer":["alarm"]}}`)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unbekannte Art: Status %d, erwartet 400", rec.Code)
	}
	if got := lies().Portal.Fuer; !cmp.Equal(got, []string{"mahnliste", "bestellung"}) {
		t.Errorf("nach der Ablehnung: %q, erwartet unverändert", got)
	}

	// Eine leere Adresse schaltet den eigenen Absender ab: Die Mails gehen wieder über den Postausgang.
	speichere(`{"smtp_host":"smtp.schule.invalid","smtp_port":"587","smtp_user":"bibliothek","smtp_password":"",` +
		`"sender_email":"bibliothek@schule.invalid","portal":{"sender_email":"","smtp_user":"","smtp_password":""}}`)
	if leer := lies(); leer.Portal.AbsenderEmail != "" {
		t.Errorf("leere Adresse gespeichert als %q", leer.Portal.AbsenderEmail)
	}
}
