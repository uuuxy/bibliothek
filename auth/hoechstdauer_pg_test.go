package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Zehn Stunden nach der letzten Passworteingabe ist eine Anmeldung gesperrt, auch wenn sie in der
// Zeit benutzt und ihr Token erneuert wurde. Das Passwort schließt sie auf und beginnt die zehn
// Stunden neu, bei totem Mailserver über den Prüfwert: Die Grenze sperrt niemanden aus.
func TestSperre_HoechstdauerNachDerPassworteingabe(t *testing.T) {
	pool := pgPoolFuerSelbstanmeldung(t)
	ctx := context.Background()
	kontoID, email := sperreKonto(t, pool)

	// Ausgestellt mit einer Stunde, erneuert von einem Server mit zwölf: Die Erneuerung stellt ein
	// neues Token aus (wie in TestSperre_UeberdauertDieErneuerungDesTokens).
	kurz := sperreAnmeldung(t, pool, time.Hour)
	a := sperreAnmeldung(t, pool, 12*time.Hour)
	mitMiniIMAP(t, kurz, "OK LOGIN completed")
	mitMiniIMAP(t, a, "OK LOGIN completed")
	cookie := sperreAnmelden(t, pool, kurz, email, "geheim-eins")

	altere := func(her time.Duration) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			UPDATE sitzungen SET passwort_geprueft_am = NOW() - make_interval(secs => $2)
			WHERE benutzer_id = $1`, kontoID, her.Seconds()); err != nil {
			t.Fatalf("Passworteingabe zurückdatieren: %v", err)
		}
	}
	passwortVor := func() time.Duration {
		t.Helper()
		var her float64
		if err := pool.QueryRow(ctx, `
			SELECT EXTRACT(EPOCH FROM NOW() - passwort_geprueft_am) FROM sitzungen WHERE benutzer_id = $1`,
			kontoID).Scan(&her); err != nil {
			t.Fatalf("Passworteingabe lesen: %v", err)
		}
		return time.Duration(her * float64(time.Second))
	}

	// Eine Minute vor der Grenze gilt das Token.
	altere(hoechstdauerNachPasswort - time.Minute)
	if _, err := a.PruefeToken(ctx, cookie.Value); err != nil {
		t.Fatalf("eine Minute vor der Höchstdauer: PruefeToken = %v, erwartet gültig", err)
	}

	// Danach ist die Anmeldung gesperrt, für jede Tür mit 423 (ErrSitzungGesperrt) und mit dem
	// eigenen Grund.
	altere(hoechstdauerNachPasswort + time.Minute)
	_, err := a.PruefeToken(ctx, cookie.Value)
	if !errors.Is(err, ErrHoechstdauerErreicht) || !errors.Is(err, ErrSitzungGesperrt) {
		t.Fatalf("nach der Höchstdauer: PruefeToken = %v, erwartet ErrHoechstdauerErreicht als ErrSitzungGesperrt", err)
	}

	// Der Sperrbildschirm erfährt den Grund und die E-Mail-Adresse, sonst nichts.
	anfrage := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	anfrage.AddCookie(cookie)
	rec := httptest.NewRecorder()
	AngemeldetesKonto(pool, a)(rec, anfrage)
	if rec.Code != http.StatusLocked {
		t.Fatalf("/api/auth/me nach der Höchstdauer: Status %d, erwartet 423: %s", rec.Code, rec.Body.String())
	}
	var zustand map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &zustand); err != nil {
		t.Fatalf("/api/auth/me: Antwort unlesbar: %v", err)
	}
	if zustand["grund"] != "hoechstdauer" || zustand["email"] != email || zustand["error"] != MeldungHoechstdauerErreicht {
		t.Errorf("/api/auth/me nach der Höchstdauer: %v, erwartet grund hoechstdauer, %q und %q",
			zustand, email, MeldungHoechstdauerErreicht)
	}
	if _, da := zustand["permissions"]; da {
		t.Error("/api/auth/me gesperrt liefert die Rechte — hinter der Sperre steht nur die E-Mail-Adresse")
	}

	// Die Erneuerung des Tokens verlängert die Anmeldung, nicht die Zeit seit dem Passwort.
	rec = sperreAnfrage(SitzungVerlaengern(a, false), "/api/auth/refresh", "", cookie)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"refresh":"renewed"`) {
		t.Fatalf("Erneuerung nach der Höchstdauer: Status %d: %s", rec.Code, rec.Body.String())
	}
	erneuert := sitzungsCookie(t, rec)
	if _, err := a.PruefeToken(ctx, erneuert.Value); !errors.Is(err, ErrHoechstdauerErreicht) {
		t.Fatalf("erneuertes Token nach der Höchstdauer: PruefeToken = %v, erwartet ErrHoechstdauerErreicht", err)
	}
	if her := passwortVor(); her < hoechstdauerNachPasswort {
		t.Fatalf("die Erneuerung hat die Passworteingabe vorgestellt: vor %v, erwartet über %v", her, hoechstdauerNachPasswort)
	}

	// Ein falsches Passwort schließt nicht auf.
	mitMiniIMAP(t, a, "NO [AUTHENTICATIONFAILED] Authentication failed.")
	if rec := sperreEntsperren(pool, a, erneuert, "geraten"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("falsches Passwort: Status %d, erwartet 401: %s", rec.Code, rec.Body.String())
	}
	if _, err := a.PruefeToken(ctx, erneuert.Value); !errors.Is(err, ErrHoechstdauerErreicht) {
		t.Fatalf("nach falschem Passwort muss die Sperre stehen: %v", err)
	}

	// Der Mailserver ist ausgefallen: Der Prüfwert schließt auf, und die zehn Stunden beginnen neu.
	mailserverTot(t)
	if rec := sperreEntsperren(pool, a, erneuert, "geheim-eins"); rec.Code != http.StatusOK {
		t.Fatalf("Mailserver tot, richtiges Passwort: Status %d, erwartet 200 — die Höchstdauer darf niemanden aussperren: %s",
			rec.Code, rec.Body.String())
	}
	if _, err := a.PruefeToken(ctx, erneuert.Value); err != nil {
		t.Fatalf("nach dem Aufschließen muss das Token gelten: %v", err)
	}
	if her := passwortVor(); her > time.Minute {
		t.Fatalf("das Aufschließen hat die Passworteingabe nicht vermerkt: vor %v", her)
	}
}
