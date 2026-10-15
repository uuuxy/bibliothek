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

// Die Sperre nach Inaktivität am ganzen Weg: echte Tür der Anmeldung, echte Tabelle sitzungen,
// ein Mailserver, der annimmt, ablehnt oder nicht erreichbar ist. Zwei Zusagen: Eine gesperrte
// Anmeldung liefert nichts, bis das Passwort eingegeben ist, und ein Ausfall des Mailservers
// sperrt niemanden aus.

const sperreTestGeheimnis = "test-secret-mit-mindestens-32-zeichen!!"

// sperreKonto legt ein aktives Konto an und räumt es am Ende des Tests ab.
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

func sperreAnmeldung(t *testing.T, pool *pgxpool.Pool, dauer time.Duration) *Anmeldung {
	t.Helper()
	a, err := NewAnmeldung(sperreTestGeheimnis, pool, dauer)
	if err != nil {
		t.Fatalf("NewAnmeldung: %v", err)
	}
	t.Cleanup(a.Sperrliste.Stoppe)
	t.Cleanup(a.Sitzungen.Stoppe)
	return a
}

func sperreAnfrage(tuer http.HandlerFunc, pfad, rumpf string, cookie *http.Cookie) *httptest.ResponseRecorder {
	anfrage := httptest.NewRequest(http.MethodPost, pfad, strings.NewReader(rumpf))
	anfrage.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		anfrage.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	tuer(rec, anfrage)
	return rec
}

// sperreAnmelden meldet über die echte Tür an und liefert das Cookie der Sitzung.
func sperreAnmelden(t *testing.T, pool *pgxpool.Pool, a *Anmeldung, email, passwort string) *http.Cookie {
	t.Helper()
	rumpf := fmt.Sprintf(`{"email":%q,"password":%q}`, email, passwort)
	rec := sperreAnfrage(Anmelden(pool, a, false), "/login", rumpf, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Anmeldung: Status %d: %s", rec.Code, rec.Body.String())
	}
	return sitzungsCookie(t, rec)
}

func sperreEntsperren(pool *pgxpool.Pool, a *Anmeldung, cookie *http.Cookie, passwort string) *httptest.ResponseRecorder {
	return sperreAnfrage(Entsperren(pool, a), "/api/auth/entsperren",
		fmt.Sprintf(`{"password":%q}`, passwort), cookie)
}

func sperreSperren(t *testing.T, a *Anmeldung, cookie *http.Cookie) bool {
	t.Helper()
	rec := sperreAnfrage(Sperren(a), "/api/auth/sperren", "", cookie)
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
	kontoID, email := sperreKonto(t, pool)
	a := sperreAnmeldung(t, pool, 12*time.Hour)

	mitMiniIMAP(t, a, "OK LOGIN completed")
	cookie := sperreAnmelden(t, pool, a, email, "geheim-eins")

	// Die Anmeldung hat eine Zeile, und in ihr steht das Passwort nicht.
	var pruefwert string
	var gesperrt bool
	if err := pool.QueryRow(ctx, `
		SELECT passwort_pruefwert, gesperrt_seit IS NOT NULL FROM sitzungen WHERE benutzer_id = $1`,
		kontoID).Scan(&pruefwert, &gesperrt); err != nil {
		t.Fatalf("Zeile der Anmeldung lesen: %v", err)
	}
	if strings.Contains(pruefwert, "geheim-eins") || !strings.HasPrefix(pruefwert, "argon2id$") {
		t.Fatalf("Prüfwert = %q, erwartet ein Argon2id-Wert ohne das Passwort", pruefwert)
	}
	if gesperrt {
		t.Fatal("eine frische Anmeldung darf nicht gesperrt sein")
	}
	if _, err := a.PruefeToken(cookie.Value); err != nil {
		t.Fatalf("vor der Sperre muss das Token gelten: %v", err)
	}

	// Sperren: Ab jetzt gilt das Token für keine Anfrage mehr, die Daten liefert.
	if !sperreSperren(t, a, cookie) {
		t.Fatal("Sperren meldet gesperrt=false, obwohl die Anmeldung eine Zeile mit Prüfwert hat")
	}
	if _, err := a.PruefeToken(cookie.Value); !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("nach dem Sperren: PruefeToken = %v, erwartet ErrSitzungGesperrt", err)
	}

	// Der eigene Zustand ist lesbar, aber nur als „gesperrt" mit der E-Mail-Adresse.
	anfrage := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	anfrage.AddCookie(cookie)
	rec := httptest.NewRecorder()
	AngemeldetesKonto(pool, a)(rec, anfrage)
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
	if _, err := a.PruefeToken(cookie.Value); !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("nach falschem Passwort muss die Sperre stehen: %v", err)
	}
	rec = sperreEntsperren(pool, a, cookie, "geheim-eins")
	if rec.Code != http.StatusOK {
		t.Fatalf("Mailserver tot, richtiges Passwort: Status %d, erwartet 200 — ein Ausfall des Mailservers darf niemanden aussperren: %s", rec.Code, rec.Body.String())
	}
	var antwort anmeldeAntwort
	if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil || antwort.Email != email || len(antwort.Rechte) == 0 {
		t.Errorf("Entsperren antwortet wie die Anmeldung: %+v (Fehler: %v)", antwort, err)
	}
	if _, err := a.PruefeToken(cookie.Value); err != nil {
		t.Fatalf("nach dem Entsperren muss das Token wieder gelten: %v", err)
	}

	// Der Mailserver ist zurück und lehnt ab: Dann entscheidet er, nicht der Prüfwert.
	sperreSperren(t, a, cookie)
	mitMiniIMAP(t, a, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	if rec := sperreEntsperren(pool, a, cookie, "geheim-eins"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("Mailserver lehnt ab: Status %d, erwartet 401 — der Prüfwert gilt nur bei einem Ausfall: %s", rec.Code, rec.Body.String())
	}

	// Der Mailserver nimmt ein neues Passwort an: Der Prüfwert zieht nach.
	mitMiniIMAP(t, a, "OK LOGIN completed")
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

	// Ausgestellt mit einer Stunde, erneuert von einem Server mit zwölf: Es bleibt weniger als
	// die Hälfte, die Tür stellt ein neues Token aus.
	kurz := sperreAnmeldung(t, pool, time.Hour)
	a := sperreAnmeldung(t, pool, 12*time.Hour)

	mitMiniIMAP(t, kurz, "OK LOGIN completed")
	cookie := sperreAnmelden(t, pool, kurz, email, "geheim")
	if !sperreSperren(t, a, cookie) {
		t.Fatal("Sperren meldet gesperrt=false")
	}

	rec := sperreAnfrage(SitzungVerlaengern(a, false), "/api/auth/refresh", "", cookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"refresh":"renewed"`) {
		t.Fatalf("Erneuerung einer gesperrten Anmeldung: Status %d: %s", rec.Code, rec.Body.String())
	}
	neu := sitzungsCookie(t, rec)
	if neu.Value == cookie.Value {
		t.Fatal("die Erneuerung hat kein neues Token ausgestellt")
	}
	if _, err := a.PruefeToken(neu.Value); !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("erneuertes Token einer gesperrten Anmeldung: PruefeToken = %v, erwartet ErrSitzungGesperrt", err)
	}
}

// Ein Token ohne Zeile lässt sich nicht sperren und wird deshalb auch nicht erneuert.
func TestSperre_TokenOhneZeileWirdNichtGesperrtUndNichtErneuert(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	kontoID, _ := sperreKonto(t, pool)
	kurz := sperreAnmeldung(t, pool, time.Hour)
	a := sperreAnmeldung(t, pool, 12*time.Hour)

	token, err := kurz.ErzeugeToken(kontoID, "", RolleAdmin, "")
	if err != nil {
		t.Fatalf("ErzeugeToken: %v", err)
	}
	cookie := &http.Cookie{Name: "session_token", Value: token}

	if sperreSperren(t, a, cookie) {
		t.Fatal("ein Token ohne Zeile meldet gesperrt=true — ohne Prüfwert ließe es sich bei einem Ausfall des Mailservers nicht aufschließen")
	}
	if _, err := a.PruefeToken(token); err != nil {
		t.Fatalf("ein Token ohne Zeile bleibt gültig: %v", err)
	}

	rec := sperreAnfrage(SitzungVerlaengern(a, false), "/api/auth/refresh", "", cookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"refresh":"skipped"`) {
		t.Fatalf("Erneuerung ohne Zeile: Status %d: %s — erwartet skipped", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("ein Token ohne Zeile darf kein neues Cookie bekommen")
	}

	// Ohne Prüfwert bleibt bei totem Mailserver die Auskunft des Mailservers.
	mailserverTot(t)
	rec = sperreEntsperren(pool, a, cookie, "egal")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Entsperren ohne Zeile bei totem Mailserver: Status %d, erwartet 503: %s", rec.Code, rec.Body.String())
	}
	const erwartet = "Mailserver ist nicht erreichbar — Anmeldung derzeit nicht möglich, bitte später erneut versuchen"
	if got := fehlermeldungAus(t, rec); got != erwartet {
		t.Errorf("Entsperren: Meldung %q, erwartet %q", got, erwartet)
	}
}

// Die Zeile lebt nicht länger als die Anmeldung: Abmelden, Löschen des Kontos und der Ablauf
// nehmen sie mit.
func TestSperre_ZeileEndetMitDerAnmeldung(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	kontoID, email := sperreKonto(t, pool)
	a := sperreAnmeldung(t, pool, 12*time.Hour)
	mitMiniIMAP(t, a, "OK LOGIN completed")

	zeilen := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM sitzungen WHERE benutzer_id = $1`, kontoID).Scan(&n); err != nil {
			t.Fatalf("Zeilen zählen: %v", err)
		}
		return n
	}

	// Abmelden.
	cookie := sperreAnmelden(t, pool, a, email, "geheim")
	claims, err := a.PruefeToken(cookie.Value)
	if err != nil {
		t.Fatalf("PruefeToken: %v", err)
	}
	if err := a.Sitzungen.Beende(ctx, claims.SitzungID); err != nil {
		t.Fatalf("Beende: %v", err)
	}
	if n := zeilen(); n != 0 {
		t.Fatalf("nach Beende: %d Zeilen, erwartet 0", n)
	}
	// Das Token trägt noch die Kennung, die Zeile ist weg: Sperren meldet, dass nichts gesperrt
	// wurde. Die Oberfläche darf sich nicht auf eine Sperre verlassen, die es nicht gibt.
	if sperreSperren(t, a, cookie) {
		t.Fatal("Sperren meldet gesperrt=true für eine Anmeldung ohne Zeile")
	}

	// Ablauf: Nur die abgelaufene Zeile geht.
	sperreAnmelden(t, pool, a, email, "geheim")
	sperreAnmelden(t, pool, a, email, "geheim")
	if _, err := pool.Exec(ctx, `
		UPDATE sitzungen SET laeuft_ab = NOW() - INTERVAL '1 minute'
		WHERE id = (SELECT id FROM sitzungen WHERE benutzer_id = $1 ORDER BY erstellt_am LIMIT 1)`,
		kontoID); err != nil {
		t.Fatalf("Zeile altern: %v", err)
	}
	a.Sitzungen.raeumeAb()
	if n := zeilen(); n != 1 {
		t.Fatalf("nach dem Abräumen: %d Zeilen, erwartet 1 (die laufende)", n)
	}
	var laufende int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sitzungen WHERE benutzer_id = $1 AND laeuft_ab > NOW()`,
		kontoID).Scan(&laufende); err != nil {
		t.Fatalf("Zeilen zählen: %v", err)
	}
	if laufende != 1 {
		t.Fatalf("nach dem Abräumen steht die abgelaufene Zeile da und die laufende ist weg")
	}

	// Konto gelöscht.
	if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE id = $1`, kontoID); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}
	if n := zeilen(); n != 0 {
		t.Fatalf("nach dem Löschen des Kontos: %d Zeilen, erwartet 0", n)
	}
}
