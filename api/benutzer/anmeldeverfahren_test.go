package benutzer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/auth"
	"bibliothek/repository"

	"github.com/google/go-cmp/cmp"
)

// verfahrensAttrappe merkt sich, was Anlegen und Ändern schreiben würden.
type verfahrensAttrappe struct {
	eskalationsAttrappe
	kennungVergeben bool
	neu             *repository.KontoNeu
	aenderung       *repository.KontoAenderung
}

func (a *verfahrensAttrappe) IstLDAPKennungVergeben(_ context.Context, _, _ string) (bool, error) {
	return a.kennungVergeben, nil
}

func (a *verfahrensAttrappe) LegeAn(_ context.Context, neu repository.KontoNeu) (string, error) {
	a.neu = &neu
	return "neu-1", nil
}

func (a *verfahrensAttrappe) Aendere(_ context.Context, p repository.KontoAenderung) (repository.KontoStand, error) {
	a.aenderung = &p
	return repository.KontoStand{}, nil
}

func TestKontoAnlegen_Anmeldeverfahren(t *testing.T) {
	faelle := []struct {
		name            string
		zusatz          string
		kennungVergeben bool
		status          int
		verfahren       string
		kennung         string
	}{
		{name: "ohne Angabe IMAP", zusatz: ``, status: http.StatusOK, verfahren: "imap"},
		{name: "LDAP mit Kennung", zusatz: `,"anmeldeverfahren":"LDAP","ldap_kennung":"  mmueller "`, status: http.StatusOK, verfahren: "ldap", kennung: "mmueller"},
		{name: "IMAP verwirft eine Kennung", zusatz: `,"anmeldeverfahren":"imap","ldap_kennung":"rest"`, status: http.StatusOK, verfahren: "imap"},
		{name: "LDAP ohne Kennung", zusatz: `,"anmeldeverfahren":"ldap","ldap_kennung":" "`, status: http.StatusBadRequest},
		{name: "unbekanntes Verfahren", zusatz: `,"anmeldeverfahren":"kerberos"`, status: http.StatusBadRequest},
		{name: "Kennung vergeben", zusatz: `,"anmeldeverfahren":"ldap","ldap_kennung":"mmueller"`, kennungVergeben: true, status: http.StatusBadRequest},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &verfahrensAttrappe{kennungVergeben: f.kennungVergeben}
			srv := &Tueren{Basis: &tuer.Basis{}}
			rumpf := `{"vorname":"Max","nachname":"Muster","email":"max@schule.de","rolle":"helfer"` + f.zusatz + `}`
			w := httptest.NewRecorder()
			srv.KontoAnlegen(repo).ServeHTTP(w, anfrageAls(t, http.MethodPost, "/api/benutzer", rumpf, "admin-1", auth.RolleAdmin))

			if w.Code != f.status {
				t.Fatalf("KontoAnlegen: Status %d, erwartet %d (%s)", w.Code, f.status, w.Body.String())
			}
			if f.status != http.StatusOK {
				if repo.neu != nil {
					t.Error("KontoAnlegen hat trotz Ablehnung angelegt")
				}
				return
			}
			if repo.neu == nil {
				t.Fatal("KontoAnlegen hat nichts angelegt")
			}
			if got, want := [2]string{repo.neu.Anmeldeverfahren, repo.neu.LDAPKennung}, [2]string{f.verfahren, f.kennung}; got != want {
				t.Errorf("KontoAnlegen schrieb Verfahren und Kennung %q, erwartet %q", got, want)
			}
		})
	}
}

func TestKontoAendern_Anmeldeverfahren(t *testing.T) {
	text := func(s string) *string { return &s }
	faelle := []struct {
		name   string
		rumpf  string
		status int
		want   *repository.KontoAenderung
	}{
		{
			name: "zu LDAP mit Kennung", rumpf: `{"anmeldeverfahren":"ldap","ldap_kennung":"mmueller"}`, status: http.StatusOK,
			want: &repository.KontoAenderung{ID: "kollege-1", Anmeldeverfahren: text("ldap"), LDAPKennung: text("mmueller")},
		},
		{
			name: "zu IMAP, die Kennung fällt weg", rumpf: `{"anmeldeverfahren":"imap","ldap_kennung":""}`, status: http.StatusOK,
			want: &repository.KontoAenderung{ID: "kollege-1", Anmeldeverfahren: text("imap")},
		},
		{name: "Kennung ohne Verfahren", rumpf: `{"ldap_kennung":"mmueller"}`, status: http.StatusBadRequest},
		{name: "leere Kennung", rumpf: `{"anmeldeverfahren":"ldap","ldap_kennung":""}`, status: http.StatusBadRequest},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			repo := &verfahrensAttrappe{eskalationsAttrappe: eskalationsAttrappe{rollen: map[string]string{"kollege-1": "HELFER"}}}
			srv := &Tueren{Basis: &tuer.Basis{}}
			req := anfrageAls(t, http.MethodPut, "/api/benutzer/kollege-1", f.rumpf, "admin-1", auth.RolleAdmin)
			req.SetPathValue("id", "kollege-1")
			w := httptest.NewRecorder()
			srv.KontoAendern(repo).ServeHTTP(w, req)

			if w.Code != f.status {
				t.Fatalf("KontoAendern: Status %d, erwartet %d (%s)", w.Code, f.status, w.Body.String())
			}
			if diff := cmp.Diff(f.want, repo.aenderung); diff != "" {
				t.Errorf("KontoAendern schrieb (-erwartet +ist):\n%s", diff)
			}
		})
	}
}

// Die LDAP-Kennung öffnet ein Konto wie die E-Mail. Sie am Konto eines Admins zu setzen, darf
// deshalb nur ein Admin: Sonst stellte ein Mitarbeiter mit manage_users das Admin-Konto auf
// seinen eigenen Verzeichniseintrag und meldete sich damit an.
func TestKontoAendern_KennungEinesAdminsNurDurchAdmin(t *testing.T) {
	repo := &verfahrensAttrappe{eskalationsAttrappe: eskalationsAttrappe{rollen: map[string]string{"admin-2": "ADMIN"}}}
	srv := &Tueren{Basis: &tuer.Basis{}}
	req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-2",
		`{"anmeldeverfahren":"ldap","ldap_kennung":"mitarbeiter"}`, "mitarbeiter-1", auth.RolleMitarbeiter)
	req.SetPathValue("id", "admin-2")
	w := httptest.NewRecorder()
	srv.KontoAendern(repo).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("KontoAendern: Status %d, erwartet 403", w.Code)
	}
	if repo.aenderung != nil {
		t.Error("KontoAendern hat die Kennung des Admin-Kontos trotz Ablehnung geschrieben")
	}
}

// Das eigene Anmeldeverfahren ändert niemand selbst: Ein Tippfehler in der eigenen Kennung
// sperrte bei der nächsten Anmeldung aus. Ein anderes Konto bleibt änderbar.
func TestKontoAendern_EigenesAnmeldeverfahrenNichtSelbst(t *testing.T) {
	for _, rumpf := range []string{
		`{"anmeldeverfahren":"ldap","ldap_kennung":"ich"}`,
		`{"anmeldeverfahren":"imap","ldap_kennung":""}`,
	} {
		repo := &verfahrensAttrappe{eskalationsAttrappe: eskalationsAttrappe{rollen: map[string]string{"admin-1": "ADMIN"}}}
		srv := &Tueren{Basis: &tuer.Basis{}}
		req := anfrageAls(t, http.MethodPut, "/api/benutzer/admin-1", rumpf, "admin-1", auth.RolleAdmin)
		req.SetPathValue("id", "admin-1")
		w := httptest.NewRecorder()
		srv.KontoAendern(repo).ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("KontoAendern(%s) am eigenen Konto: Status %d, erwartet 403", rumpf, w.Code)
		}
		if repo.aenderung != nil {
			t.Errorf("KontoAendern(%s) hat am eigenen Konto geschrieben", rumpf)
		}
	}
}
