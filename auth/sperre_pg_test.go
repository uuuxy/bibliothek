package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Sperre nach Inaktivität am ganzen Weg: echter LoginHandler, echte Tabelle sitzungen,
// ein Mailserver, der antwortet, ablehnt oder nicht erreichbar ist. Zwei Zusagen: Eine
// gesperrte Anmeldung liefert nichts, bis das Passwort eingegeben ist, und ein Ausfall des
// Mailservers sperrt niemanden aus.

const sperreTestGeheimnis = "test-secret-mit-mindestens-32-zeichen!!"

// sperreKonto legt ein aktives Konto an und räumt es am Testende ab.
func sperreKonto(t *testing.T, pool *pgxpool.Pool) (id, email string) {
	t.Helper()
	email = fmt.Sprintf("sperre-%d@sperrtest.invalid", time.Now().UnixNano())
	raeumeKontoAb(t, pool, email)
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Sperr', 'Probe', $1, 'admin', true)
		RETURNING id::text`, email).Scan(&id); err != nil {
		t.Fatalf("Konto anlegen: %v", err)
	}
	return id, email
}

func sperreAuthenticator(t *testing.T, pool *pgxpool.Pool, dauer time.Duration) *Authenticator {
	t.Helper()
	a, err := NewAuthenticator(sperreTestGeheimnis, pool, dauer)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	t.Cleanup(a.Blacklist.Stop)
	t.Cleanup(a.Sitzungen.Stop)
	return a
}

func sperreAnfrage(handler http.HandlerFunc, pfad, rumpf string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, pfad, strings.NewReader(rumpf))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// sperreAnmelden meldet über den echten Handler an und liefert das Sitzungs-Cookie.
func sperreAnmelden(t *testing.T, pool *pgxpool.Pool, a *Authenticator, email, passwort string) *http.Cookie {
	t.Helper()
	rumpf := fmt.Sprintf(`{"email":%q,"password":%q}`, email, passwort)
	rec := sperreAnfrage(LoginHandler(pool, a, false), "/login", rumpf, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Anmeldung: Status %d: %s", rec.Code, rec.Body.String())
	}
	return sitzungsCookie(t, rec)
}

func sperreEntsperren(pool *pgxpool.Pool, a *Authenticator, cookie *http.Cookie, passwort string) *httptest.ResponseRecorder {
	return sperreAnfrage(EntsperrenHandler(pool, a), "/api/auth/entsperren",
		fmt.Sprintf(`{"password":%q}`, passwort), cookie)
}

func sperreSperren(t *testing.T, a *Authenticator, cookie *http.Cookie) bool {
	t.Helper()
	rec := sperreAnfrage(SperrenHandler(a), "/api/auth/sperren", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("Sperren: Status %d: %s", rec.Code, rec.Body.String())
	}
	var antwort struct {
		Gesperrt bool `json:"gesperrt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
		t.Fatalf("Sperren: Antwort unlesbar: %v", err)
	}
	return antwort.Gesperrt
}

func mailserverTot(t *testing.T) {
	t.Helper()
	host, port := totePortAdresse(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("IMAP_HOST", host)
	t.Setenv("IMAP_PORT", port)
}

func TestSperre_GiltAmServerUndGehtOhneMailserverAuf(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	benutzerID, email := sperreKonto(t, pool)
	a := sperreAuthenticator(t, pool, 12*time.Hour)

	mitMiniIMAP(t, "OK LOGIN completed")
	cookie := sperreAnmelden(t, pool, a, email, "geheim-eins")

	// Die Anmeldung hat eine Zeile, und in ihr steht das Passwort nicht.
	var pruefwert string
	var gesperrt bool
	if err := pool.QueryRow(ctx, `
		SELECT passwort_pruefwert, gesperrt_seit IS NOT NULL FROM sitzungen WHERE benutzer_id = $1`,
		benutzerID).Scan(&pruefwert, &gesperrt); err != nil {
		t.Fatalf("Zeile der Anmeldung lesen: %v", err)
	}
	if strings.Contains(pruefwert, "geheim-eins") || !strings.HasPrefix(pruefwert, "argon2id$") {
		t.Fatalf("Prüfwert = %q, erwartet ein Argon2id-Wert ohne das Passwort", pruefwert)
	}
	if gesperrt {
		t.Fatal("eine frische Anmeldung darf nicht gesperrt sein")
	}
	if _, err := a.VerifyToken(cookie.Value); err != nil {
		t.Fatalf("vor der Sperre muss das Token gelten: %v", err)
	}

	// Sperren: Ab jetzt gilt das Token für keine Anfrage mehr, die Daten liefert.
	if !sperreSperren(t, a, cookie) {
		t.Fatal("Sperren meldet gesperrt=false, obwohl die Anmeldung eine Zeile mit Prüfwert hat")
	}
	if _, err := a.VerifyToken(cookie.Value); !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("nach dem Sperren: VerifyToken = %v, erwartet ErrSitzungGesperrt", err)
	}

	// Der eigene Zustand ist lesbar, aber nur als „gesperrt" mit der E-Mail-Adresse.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	MeHandler(pool, a)(rec, req)
	if rec.Code != http.StatusLocked {
		t.Fatalf("/api/auth/me gesperrt: Status %d, erwartet 423: %s", rec.Code, rec.Body.String())
	}
	var zustand map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &zustand); err != nil {
		t.Fatalf("/api/auth/me: Antwort unlesbar: %v", err)
	}
	if zustand["email"] != email {
		t.Errorf("/api/auth/me gesperrt: email = %v, erwartet %q", zustand["email"], email)
	}
	for _, feld := range []string{"permissions", "rolle", "vorname", "nachname", "user_id"} {
		if _, da := zustand[feld]; da {
			t.Errorf("/api/auth/me gesperrt liefert %q — hinter der Sperre steht nur die E-Mail-Adresse", feld)
		}
	}

	// Der Mailserver fällt aus. Ein falsches Passwort schließt nicht auf, das der Anmeldung schon.
	mailserverTot(t)
	if rec := sperreEntsperren(pool, a, cookie, "geraten"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Mailserver tot, falsches Passwort: Status %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
	if _, err := a.VerifyToken(cookie.Value); !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("nach falschem Passwort muss die Sperre stehen: %v", err)
	}
	rec = sperreEntsperren(pool, a, cookie, "geheim-eins")
	if rec.Code != http.StatusOK {
		t.Fatalf("Mailserver tot, richtiges Passwort: Status %d, erwartet 200 — ein Ausfall des Mailservers darf niemanden aussperren: %s", rec.Code, rec.Body.String())
	}
	var antwort LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || antwort.Email != email || len(antwort.Permissions) == 0 {
		t.Errorf("Entsperren antwortet wie die Anmeldung: %+v (Fehler: %v)", antwort, err)
	}
	if _, err := a.VerifyToken(cookie.Value); err != nil {
		t.Fatalf("nach dem Entsperren muss das Token wieder gelten: %v", err)
	}

	// Der Mailserver ist zurück und lehnt ab: Dann entscheidet er, nicht der Prüfwert.
	sperreSperren(t, a, cookie)
	mitMiniIMAP(t, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	if rec := sperreEntsperren(pool, a, cookie, "geheim-eins"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Mailserver lehnt ab: Status %d, erwartet 401 — der Prüfwert gilt nur bei einem Ausfall: %s", rec.Code, rec.Body.String())
	}

	// Der Mailserver nimmt ein neues Passwort an: Der Prüfwert zieht nach.
	mitMiniIMAP(t, "OK LOGIN completed")
	if rec := sperreEntsperren(pool, a, cookie, "geheim-zwei"); rec.Code != http.StatusOK {
		t.Fatalf("Mailserver nimmt das neue Passwort an: Status %d: %s", rec.Code, rec.Body.String())
	}
	sperreSperren(t, a, cookie)
	mailserverTot(t)
	if rec := sperreEntsperren(pool, a, cookie, "geheim-eins"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("altes Passwort nach dem Wechsel: Status %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
	if rec := sperreEntsperren(pool, a, cookie, "geheim-zwei"); rec.Code != http.StatusOK {
		t.Fatalf("neues Passwort bei totem Mailserver: Status %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
}

// Eine Erneuerung des Tokens hebt die Sperre nicht auf: Das neue Token trägt dieselbe Kennung.
func TestSperre_UeberdauertDieErneuerungDesTokens(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	_, email := sperreKonto(t, pool)

	// Ausgestellt mit einer Stunde, erneuert von einem Server mit zwölf: Die Restlaufzeit
	// liegt unter der Hälfte, der Handler stellt ein neues Token aus.
	kurz := sperreAuthenticator(t, pool, time.Hour)
	a := sperreAuthenticator(t, pool, 12*time.Hour)

	mitMiniIMAP(t, "OK LOGIN completed")
	cookie := sperreAnmelden(t, pool, kurz, email, "geheim")
	if !sperreSperren(t, a, cookie) {
		t.Fatal("Sperren meldet gesperrt=false")
	}

	rec := sperreAnfrage(RefreshTokenHandler(a, false), "/api/auth/refresh", "", cookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"refresh":"renewed"`) {
		t.Fatalf("Erneuerung einer gesperrten Anmeldung: Status %d: %s", rec.Code, rec.Body.String())
	}
	neu := sitzungsCookie(t, rec)
	if neu.Value == cookie.Value {
		t.Fatal("die Erneuerung hat kein neues Token ausgestellt")
	}
	if _, err := a.VerifyToken(neu.Value); !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("erneuertes Token einer gesperrten Anmeldung: VerifyToken = %v, erwartet ErrSitzungGesperrt", err)
	}
}

// Ein Token ohne Zeile lässt sich nicht sperren — und wird deshalb auch nicht erneuert.
func TestSperre_TokenOhneZeileWirdNichtGesperrtUndNichtErneuert(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	benutzerID, _ := sperreKonto(t, pool)
	kurz := sperreAuthenticator(t, pool, time.Hour)
	a := sperreAuthenticator(t, pool, 12*time.Hour)

	token, err := kurz.GenerateToken(benutzerID, "", RoleAdmin, "")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	cookie := &http.Cookie{Name: "session_token", Value: token}

	if sperreSperren(t, a, cookie) {
		t.Fatal("ein Token ohne Zeile meldet gesperrt=true — ohne Prüfwert ließe es sich bei einem Ausfall des Mailservers nicht aufschließen")
	}
	if _, err := a.VerifyToken(token); err != nil {
		t.Fatalf("ein Token ohne Zeile bleibt gültig: %v", err)
	}

	rec := sperreAnfrage(RefreshTokenHandler(a, false), "/api/auth/refresh", "", cookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"refresh":"skipped"`) {
		t.Fatalf("Erneuerung ohne Zeile: Status %d: %s — erwartet skipped", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("ein Token ohne Zeile darf kein neues Cookie bekommen")
	}

	// Ohne Prüfwert bleibt bei totem Mailserver die Auskunft des Mailservers.
	mailserverTot(t)
	if rec := sperreEntsperren(pool, a, cookie, "egal"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Entsperren ohne Zeile bei totem Mailserver: Status %d, erwartet 503: %s", rec.Code, rec.Body.String())
	}
}

// Die Zeile lebt nicht länger als die Anmeldung: Abmelden, Löschen des Kontos und der Ablauf
// nehmen sie mit.
func TestSperre_ZeileEndetMitDerAnmeldung(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	benutzerID, email := sperreKonto(t, pool)
	a := sperreAuthenticator(t, pool, 12*time.Hour)
	mitMiniIMAP(t, "OK LOGIN completed")

	zeilen := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM sitzungen WHERE benutzer_id = $1`, benutzerID).Scan(&n); err != nil {
			t.Fatalf("Zeilen zählen: %v", err)
		}
		return n
	}

	// Abmelden.
	cookie := sperreAnmelden(t, pool, a, email, "geheim")
	claims, err := a.VerifyToken(cookie.Value)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if err := a.Sitzungen.Beende(ctx, claims.SitzungID); err != nil {
		t.Fatalf("Beende: %v", err)
	}
	if n := zeilen(); n != 0 {
		t.Fatalf("nach Beende: %d Zeilen, erwartet 0", n)
	}
	// Das Token trägt noch die Kennung, die Zeile ist weg: Sperren meldet, dass nichts
	// gesperrt wurde — der Client darf sich nicht auf eine Sperre verlassen, die es nicht gibt.
	if sperreSperren(t, a, cookie) {
		t.Fatal("Sperren meldet gesperrt=true für eine Anmeldung ohne Zeile")
	}

	// Ablauf: Nur die abgelaufene Zeile geht.
	sperreAnmelden(t, pool, a, email, "geheim")
	sperreAnmelden(t, pool, a, email, "geheim")
	if _, err := pool.Exec(ctx, `
		UPDATE sitzungen SET laeuft_ab = NOW() - INTERVAL '1 minute'
		WHERE id = (SELECT id FROM sitzungen WHERE benutzer_id = $1 ORDER BY erstellt_am LIMIT 1)`,
		benutzerID); err != nil {
		t.Fatalf("Zeile altern: %v", err)
	}
	a.Sitzungen.raeumeAb()
	if n := zeilen(); n != 1 {
		t.Fatalf("nach dem Abräumen: %d Zeilen, erwartet 1 (die laufende)", n)
	}

	// Konto gelöscht.
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, benutzerID); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}
	if n := zeilen(); n != 0 {
		t.Fatalf("nach dem Löschen des Kontos: %d Zeilen, erwartet 0", n)
	}
}
