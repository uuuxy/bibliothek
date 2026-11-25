package adressabgleich

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testVCF = "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Beispiel;Anna;;;ABE\r\nEMAIL:anna.beispiel@schule.example\r\nEND:VCARD\r\n"

// fakeRoundcube spielt die Seiten von Roundcube nach, die der Abruf braucht: Anmeldeseite mit
// Token, Anmeldung mit Sitzungs-Keks, Kontakte mit Anfrage-Token und Liste der Adressbücher,
// Export (nur mit Keks und Token) und Abmeldung.
type fakeRoundcube struct {
	t            *testing.T
	prefix       string // Pfad, unter dem Roundcube liegt; die Wurzel leitet dorthin weiter
	objectSource bool   // Liste der Adressbücher als Objekt wie vor Roundcube 1.7
	withoutGlob  bool
	loggedOut    bool
	exports      int
}

func (f *fakeRoundcube) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f.prefix != "" && !strings.HasPrefix(r.URL.Path, f.prefix) {
		http.Redirect(w, r, f.prefix+"/?"+r.URL.RawQuery, http.StatusFound)
		return
	}
	q := r.URL.Query()
	authed := false
	if c, err := r.Cookie("roundcube_sessauth"); err == nil && c.Value == "ok" {
		authed = true
	}
	loginForm := `<form><input type="hidden" name="_token" value="LOGINTOKEN"><input name="_user"><input name="_pass" type="password"></form>`
	switch {
	case q.Get("_task") == "login" && r.Method == http.MethodGet:
		http.SetCookie(w, &http.Cookie{Name: "roundcube_sessid", Value: "s1", Path: "/"})
		f.schreibe(w, loginForm)
	case q.Get("_task") == "login" && r.Method == http.MethodPost:
		if _, err := r.Cookie("roundcube_sessid"); err != nil || r.FormValue("_token") != "LOGINTOKEN" ||
			r.FormValue("_user") != "lehrer@schule.example" || r.FormValue("_pass") != "geheim" {
			w.WriteHeader(http.StatusUnauthorized)
			f.schreibe(w, loginForm)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "roundcube_sessauth", Value: "ok", Path: "/"})
		http.Redirect(w, r, "./?_task=mail", http.StatusFound)
	case q.Get("_task") == "mail":
		f.schreibe(w, "Posteingang")
	case q.Get("_task") == "addressbook" && q.Get("_action") == "":
		if !authed {
			f.schreibe(w, loginForm)
			return
		}
		sources := `[{"id":"0","name":"Persönliches Adressbuch","groups":true},{"id":"global","name":"Globales Adressbuch","readonly":true}]`
		if f.objectSource {
			sources = `{"0":{"id":"0","name":"Persönliches Adressbuch"},"global":{"id":"global","name":"Globales Adressbuch"}}`
		}
		if f.withoutGlob {
			sources = `[{"id":"0","name":"Persönliches Adressbuch"}]`
		}
		f.schreibe(w, fmt.Sprintf(`<script>rcmail.set_env({"task":"addressbook","request_token":"REQTOKEN","address_sources":%s,"source":"0"});</script>`, sources))
	case q.Get("_action") == "export":
		if !authed || q.Get("_token") != "REQTOKEN" || q.Get("_source") != "global" {
			http.Error(w, "Request security check failed", http.StatusForbidden)
			return
		}
		f.exports++
		w.Header().Set("Content-Type", "text/vcard; charset=UTF-8")
		f.schreibe(w, testVCF)
	case q.Get("_task") == "logout":
		f.loggedOut = q.Get("_token") == "REQTOKEN"
		f.schreibe(w, loginForm)
	default:
		http.NotFound(w, r)
	}
}

// schreibe schreibt den Rumpf einer Antwort des Nachbaus.
func (f *fakeRoundcube) schreibe(w http.ResponseWriter, rumpf string) {
	if _, err := io.WriteString(w, rumpf); err != nil {
		f.t.Errorf("Antwort des Nachbaus: %v", err)
	}
}

func starteRoundcube(t *testing.T, f *fakeRoundcube) string {
	t.Helper()
	f.t = t
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	return server.URL
}

func TestHoleGlobalesAdressbuch_ExportsWithCookieAndToken(t *testing.T) {
	f := &fakeRoundcube{}
	adresse := starteRoundcube(t, f)

	vcf, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: adresse + "/?_task=mail&_mbox=INBOX", Benutzer: "lehrer@schule.example", Passwort: "geheim"})
	if err != nil {
		t.Fatalf("HoleGlobalesAdressbuch: %v", err)
	}
	if string(vcf) != testVCF {
		t.Errorf("HoleGlobalesAdressbuch = %q, erwartet %q", vcf, testVCF)
	}
	if !f.loggedOut {
		t.Error("HoleGlobalesAdressbuch hat sich nicht mit dem Anfrage-Token abgemeldet")
	}
}

func TestHoleGlobalesAdressbuch_FollowsRedirectToRoundcubePath(t *testing.T) {
	f := &fakeRoundcube{prefix: "/roundcube"}
	adresse := starteRoundcube(t, f)

	if _, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: adresse, Benutzer: "lehrer@schule.example", Passwort: "geheim"}); err != nil {
		t.Fatalf("HoleGlobalesAdressbuch hinter einer Weiterleitung: %v", err)
	}
	if f.exports != 1 {
		t.Errorf("Exporte = %d, erwartet 1", f.exports)
	}
}

func TestHoleGlobalesAdressbuch_ReadsSourceListOfOlderRoundcube(t *testing.T) {
	adresse := starteRoundcube(t, &fakeRoundcube{objectSource: true})

	if _, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: adresse, Benutzer: "lehrer@schule.example", Passwort: "geheim"}); err != nil {
		t.Fatalf("HoleGlobalesAdressbuch mit der Liste als Objekt: %v", err)
	}
}

func TestHoleGlobalesAdressbuch_WrongPasswordIsRejected(t *testing.T) {
	f := &fakeRoundcube{}
	adresse := starteRoundcube(t, f)

	_, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: adresse, Benutzer: "lehrer@schule.example", Passwort: "falsch"})
	if !errors.Is(err, ErrAnmeldungAbgelehnt) {
		t.Errorf("HoleGlobalesAdressbuch mit falschem Passwort = %v, erwartet ErrAnmeldungAbgelehnt", err)
	}
	if f.exports != 0 {
		t.Errorf("Exporte = %d, erwartet 0", f.exports)
	}
}

func TestHoleGlobalesAdressbuch_WithoutGlobalContacts(t *testing.T) {
	adresse := starteRoundcube(t, &fakeRoundcube{withoutGlob: true})

	_, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: adresse, Benutzer: "lehrer@schule.example", Passwort: "geheim"})
	if !errors.Is(err, ErrKeinGlobalesAdressbuch) {
		t.Errorf("HoleGlobalesAdressbuch ohne globales Adressbuch = %v, erwartet ErrKeinGlobalesAdressbuch", err)
	}
}

func TestHoleGlobalesAdressbuch_ForeignPageWithoutForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, "Willkommen"); err != nil {
			t.Errorf("Antwort: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	_, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: server.URL, Benutzer: "x", Passwort: "y"})
	if err == nil || !strings.Contains(err.Error(), "keine Anmeldeseite von Roundcube") {
		t.Errorf("HoleGlobalesAdressbuch auf einer fremden Seite = %v, erwartet den Hinweis auf die fehlende Anmeldeseite", err)
	}
}

func TestGrundadresse(t *testing.T) {
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{in: "webmail.schule.example", want: "https://webmail.schule.example"},
		{in: "https://webmail.schule.example/", want: "https://webmail.schule.example"},
		{in: "https://webmail.schule.example/?_task=mail&_mbox=INBOX", want: "https://webmail.schule.example"},
		{in: "https://www.schule.example/roundcube/index.php?_task=login", want: "https://www.schule.example/roundcube"},
		{in: " http://127.0.0.1:8099/ ", want: "http://127.0.0.1:8099"},
		{in: "http://localhost/mail", want: "http://localhost/mail"},
		{in: "http://webmail.schule.example", wantErr: true},
		{in: "ftp://webmail.schule.example", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := Grundadresse(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("Grundadresse(%q): Fehler %v, erwartet Fehler: %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("Grundadresse(%q) = %q, erwartet %q", tt.in, got, tt.want)
		}
	}
}

// Leitet das Webmail auf http weiter, ginge die Anmeldung im Klartext durchs Netz.
// webmail.schule.invalid steht für einen fremden Rechner; folgte der Abruf, scheiterte er dort an
// der Namensauflösung statt an der Regel.
func TestHoleGlobalesAdressbuch_WeiterleitungOhneHTTPS(t *testing.T) {
	for _, methode := range []string{http.MethodGet, http.MethodPost} {
		t.Run(methode, func(t *testing.T) {
			f := &fakeRoundcube{t: t}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == methode && r.URL.Query().Get("_task") == "login" {
					// 307 behält Methode und Rumpf: Beim POST ginge das Passwort mit.
					http.Redirect(w, r, "http://webmail.schule.invalid"+r.URL.RequestURI(), http.StatusTemporaryRedirect)
					return
				}
				f.ServeHTTP(w, r)
			}))
			t.Cleanup(server.Close)

			_, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: server.URL, Benutzer: "lehrer@schule.example", Passwort: "geheim"})
			if !errors.Is(err, ErrWeiterleitungOhneHTTPS) || !strings.Contains(err.Error(), "http://webmail.schule.invalid") {
				t.Errorf("HoleGlobalesAdressbuch nach einer Weiterleitung auf http = %v, erwartet die Ablehnung mit dem Ziel", err)
			}
		})
	}
}

// Ein Ziel mit https lässt die Regel zu, auch auf einem anderen Rechner; nach zehn
// Weiterleitungen ist Schluss wie mit der Vorgabe von net/http.
func TestPruefeWeiterleitung(t *testing.T) {
	tests := []struct {
		ziel    string
		vorige  int
		wantErr bool
	}{
		{ziel: "https://mail.schule.example/roundcube/?_task=login"},
		{ziel: "http://127.0.0.1:8099/?_task=login"},
		{ziel: "http://localhost/roundcube/?_task=login"},
		{ziel: "http://mail.schule.example/?_task=login", wantErr: true},
		{ziel: "http://169.254.169.254/latest/meta-data/", wantErr: true},
		{ziel: "ftp://mail.schule.example/", wantErr: true},
		{ziel: "https://mail.schule.example/?_task=login", vorige: 9},
		{ziel: "https://mail.schule.example/?_task=login", vorige: 10, wantErr: true},
	}
	for _, tt := range tests {
		anfrage, err := http.NewRequest(http.MethodGet, tt.ziel, nil)
		if err != nil {
			t.Fatalf("http.NewRequest(%q): %v", tt.ziel, err)
		}
		err = pruefeWeiterleitung(anfrage, make([]*http.Request, tt.vorige))
		if (err != nil) != tt.wantErr {
			t.Errorf("pruefeWeiterleitung(%q nach %d Weiterleitungen) = %v, erwartet Fehler: %v", tt.ziel, tt.vorige, err, tt.wantErr)
		}
	}
}

func TestHoleGlobalesAdressbuch_ErrorCarriesNoToken(t *testing.T) {
	f := &fakeRoundcube{t: t}
	server := httptest.NewServer(f)
	adresse := server.URL
	// Die Verbindung bricht ab, sobald der Export angefragt wird.
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("_action") == "export" {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("die Antwort lässt sich nicht übernehmen")
				return
			}
			verbindung, _, err := hijacker.Hijack()
			if err != nil {
				t.Errorf("Hijack: %v", err)
				return
			}
			if err := verbindung.Close(); err != nil {
				t.Errorf("Verbindung schließen: %v", err)
			}
			return
		}
		f.ServeHTTP(w, r)
	})
	t.Cleanup(server.Close)

	_, err := HoleGlobalesAdressbuch(context.Background(), Zugang{Adresse: adresse, Benutzer: "lehrer@schule.example", Passwort: "geheim"})
	if err == nil || strings.Contains(err.Error(), "REQTOKEN") || strings.Contains(err.Error(), "_token") {
		t.Errorf("HoleGlobalesAdressbuch bei abgebrochenem Export = %v, erwartet einen Fehler ohne Token", err)
	}
}
