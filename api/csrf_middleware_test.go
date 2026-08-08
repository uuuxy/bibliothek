package api

// csrf_middleware_test.go — Belegt die CSRF-Prüfung am Verhalten der Middleware statt
// an der Klassifizierungsfunktion: Was zählt, ist, ob eine mutierende Anfrage ohne
// gültiges Token den dahinterliegenden Handler erreicht.
//
// Anlass ist eine Ausnahmeliste, die /api/admin und vier weitere Präfixe von der
// Prüfung ausnahm — begründet mit einem Inventur-CSRF-System, das es im Code nicht
// gab. Ein Test über die Klassifizierung hätte diese Liste bloß abgeschrieben; diese
// Tests nennen stattdessen die echten Routen, die geschützt sein müssen.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// durchlaufCSRF schickt eine Anfrage durch die Middleware und meldet, ob der
// dahinterliegende Handler erreicht wurde.
func durchlaufCSRF(t *testing.T, method, path string, mitToken bool) (erreicht bool, code int) {
	t.Helper()

	s := &Server{}
	mw := s.CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		erreicht = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	if mitToken {
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok-42"})
		req.Header.Set("X-CSRF-Token", "tok-42")
	}
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	return erreicht, rec.Code
}

// Die Routen in dieser Liste waren durch die /api/admin-Ausnahme ungeschützt. Sie
// stehen hier ausgeschrieben, damit ein wiederkehrender Präfix-Freibrief auffällt.
var mutierendeAdminRouten = map[string][2]string{
	"Rechte aller Rollen ändern":  {http.MethodPut, "/api/admin/permissions"},
	"SMTP-Server umkonfigurieren": {http.MethodPut, "/api/admin/settings/mail"},
	"Testmail verschicken":        {http.MethodPost, "/api/admin/settings/mail/test"},
	"Schüler sperren":             {http.MethodPatch, "/api/admin/students/7/lock"},
	"Fälligkeit verschieben":      {http.MethodPatch, "/api/admin/ausleihen/7/faelligkeit"},
	"Mahnungen drucken":           {http.MethodPost, "/api/admin/mahnungen/bulk-print"},
	"Cover-Sync starten":          {http.MethodPost, "/api/admin/sync-covers"},
	"Bestand importieren":         {http.MethodPost, "/api/admin/import-bestand"},
	"Buch anlegen":                {http.MethodPost, "/api/books"},
	"Fach anlegen":                {http.MethodPost, "/api/subjects"},
}

func TestCSRFMiddlewareSchuetztMutationenOhneToken(t *testing.T) {
	for name, route := range mutierendeAdminRouten {
		t.Run(name, func(t *testing.T) {
			erreicht, code := durchlaufCSRF(t, route[0], route[1], false)
			if erreicht {
				t.Fatalf("%s %s ohne Token hat den Handler erreicht", route[0], route[1])
			}
			if code != http.StatusForbidden {
				t.Errorf("Status = %d, erwartet 403", code)
			}
		})
	}
}

func TestCSRFMiddlewareLaesstMutationenMitTokenDurch(t *testing.T) {
	for name, route := range mutierendeAdminRouten {
		t.Run(name, func(t *testing.T) {
			erreicht, code := durchlaufCSRF(t, route[0], route[1], true)
			if !erreicht {
				t.Fatalf("%s %s mit gültigem Token wurde blockiert (Status %d)", route[0], route[1], code)
			}
		})
	}
}

// Ein Token, das nicht zum Cookie passt, ist kein Token: Sonst genügte ein beliebiger
// selbstgesetzter Header, und das Double-Submit-Verfahren wäre wertlos.
func TestCSRFMiddlewareWeistFremdesTokenAb(t *testing.T) {
	s := &Server{}
	erreicht := false
	mw := s.CSRFMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { erreicht = true }))

	req := httptest.NewRequest(http.MethodPut, "/api/admin/permissions", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok-42"})
	req.Header.Set("X-CSRF-Token", "tok-43")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if erreicht || rec.Code != http.StatusForbidden {
		t.Fatalf("abweichendes Token: erreicht=%v Status=%d, erwartet 403", erreicht, rec.Code)
	}
}

// Die beiden Ausnahmen müssen offen bleiben, sonst kommt niemand mehr heraus: Ein
// abgelaufenes Token darf sich beim Abmelden bzw. Erneuern nicht am fehlenden
// CSRF-Cookie festhaken.
//
// Hier stand bis zum 08.08.2026 zusätzlich /login/barcode, und der Kommentar sprach von
// „drei Ausnahmen". istPruefungsAusnahme kennt aber nur zwei — der Pfad kam durch, weil
// er aus istAPIPfad herausfiel, nicht weil ihn jemand freigegeben hätte. Der Test hat
// eine LÜCKE als Entscheidung dokumentiert und hätte grün weitergelaufen, wenn jemand
// die Route tatsächlich angelegt hätte.
func TestCSRFMiddlewareAusnahmenBleibenOffen(t *testing.T) {
	for _, pfad := range []string{"/api/auth/logout", "/api/auth/refresh"} {
		t.Run(pfad, func(t *testing.T) {
			erreicht, code := durchlaufCSRF(t, http.MethodPost, pfad, false)
			if !erreicht {
				t.Fatalf("%s ohne Token wurde blockiert (Status %d)", pfad, code)
			}
		})
	}
}

// Und die Gegenrichtung, die vorher niemand geprüft hat: Ein Unterpfad von /login ist
// KEINE Ausnahme. Legt jemand /login/code für den Helfer-Zugang an, greift die
// CSRF-Prüfung ab der ersten Zeile — ohne dass er daran denken muss.
//
// Login-CSRF ist kein Randfall: Eine fremde Seite kann ein Formular mit
// enctype="text/plain" abschicken, dessen Rumpf als JSON durchgeht, und das Opfer damit
// in die Sitzung eines ANGREIFER-Kontos anmelden. Alles, was danach am Tresen gescannt
// wird, landet dort. SameSite=Strict verhindert das nicht — es regelt das SENDEN von
// Cookies, nicht das Setzen durch die Antwort.
func TestCSRFMiddlewarePrueftAuchLoginUnterpfade(t *testing.T) {
	for _, pfad := range []string{"/login", "/login/code", "/login/barcode"} {
		t.Run(pfad, func(t *testing.T) {
			erreicht, code := durchlaufCSRF(t, http.MethodPost, pfad, false)
			if erreicht {
				t.Fatalf("%s ohne CSRF-Token durchgelassen (Status %d) — Login-CSRF steht offen", pfad, code)
			}
			if code != http.StatusForbidden {
				t.Errorf("%s: Status %d, erwartet 403", pfad, code)
			}
		})
	}
}

// Lesende Zugriffe dürfen nie an der Prüfung scheitern — und holen sich dabei das
// Cookie, mit dem das Frontend die nächste Mutation unterschreibt.
func TestCSRFMiddlewareLaesstLesendeDurchUndSetztDasCookie(t *testing.T) {
	s := &Server{}
	mw := s.CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/permissions", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET wurde blockiert: Status %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), csrfCookieName) {
		t.Errorf("kein %s-Cookie in der Antwort: %q", csrfCookieName, rec.Header().Get("Set-Cookie"))
	}
}

func TestCSRFMiddlewareUeberspringtCSRFTokenEndpunkt(t *testing.T) {
	s := &Server{}
	mw := s.CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/csrf-token", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET wurde blockiert: Status %d", rec.Code)
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Errorf("Middleware darf beim Endpunkt /api/csrf-token kein Cookie setzen, gefunden: %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestCSRFMiddlewareLeeresToken(t *testing.T) {
	s := &Server{}
	mw := s.CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("Leerer Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/admin/permissions", strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok-42"})
		req.Header.Set("X-CSRF-Token", "   ") // leer nach Trim
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Erwartet 403 bei leerem Header, erhalten %d", rec.Code)
		}
	})

	t.Run("Leeres Cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/admin/permissions", strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "   "}) // leer nach Trim
		req.Header.Set("X-CSRF-Token", "tok-42")
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Erwartet 403 bei leerem Cookie, erhalten %d", rec.Code)
		}
	})
}
