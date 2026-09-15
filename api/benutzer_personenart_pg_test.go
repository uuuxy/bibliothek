package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Die Personenart sagt, WER jemand im Kollegium ist — Lehrkraft oder LiV (Lehrkraft im
// Vorbereitungsdienst). Die Rolle sagt, was jemand in der Software DARF; beides ist getrennt
// (Peter, 15.09.2026). Sichtbar ist die Personenart nur in der Benutzerverwaltung.
func TestBenutzerPersonenart(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	admin := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a119", Rolle: auth.RoleAdmin}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@personenart.invalid'`); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})

	fahre := func(h http.Handler, methode, id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(methode, "/api/benutzer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if id != "" {
			req.SetPathValue("id", id)
		}
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, admin))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	lies := func(t *testing.T, email string) (id string, art *string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT id, personenart FROM benutzer WHERE email = $1`, email).
			Scan(&id, &art); err != nil {
			t.Fatalf("Konto %s lesen: %v", email, err)
		}
		return id, art
	}
	erwarte := func(t *testing.T, art *string, wert string) {
		t.Helper()
		switch {
		case wert == "" && art != nil:
			t.Errorf("Personenart %q, erwartet leer", *art)
		case wert != "" && (art == nil || *art != wert):
			t.Errorf("Personenart %v, erwartet %q", art, wert)
		}
	}
	anlegen := func(t *testing.T, body string) {
		t.Helper()
		if rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "", body); rec.Code != http.StatusOK {
			t.Fatalf("Anlegen: Status %d: %s", rec.Code, rec.Body.String())
		}
	}

	t.Run("Kollegium ohne Angabe wird Lehrkraft", func(t *testing.T) {
		anlegen(t, `{"vorname":"Kai","nachname":"Kollege","email":"a@personenart.invalid","rolle":"kollegium"}`)
		_, art := lies(t, "a@personenart.invalid")
		erwarte(t, art, "lehrkraft")
	})

	t.Run("LiV wird gespeichert", func(t *testing.T) {
		anlegen(t, `{"vorname":"Lia","nachname":"Vorbereitung","email":"b@personenart.invalid","rolle":"kollegium","personenart":"liv"}`)
		_, art := lies(t, "b@personenart.invalid")
		erwarte(t, art, "liv")
	})

	t.Run("Mitarbeiter ohne Angabe bleibt leer", func(t *testing.T) {
		anlegen(t, `{"vorname":"Mia","nachname":"Mitarbeit","email":"c@personenart.invalid","rolle":"mitarbeiter"}`)
		_, art := lies(t, "c@personenart.invalid")
		erwarte(t, art, "")
	})

	t.Run("unbekannte Personenart wird abgewiesen", func(t *testing.T) {
		rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "",
			`{"vorname":"Udo","nachname":"Unbekannt","email":"d@personenart.invalid","rolle":"kollegium","personenart":"schueler"}`)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Personenart") {
			t.Fatalf("erwartet 400 mit „Personenart“, war %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Ändern ohne das Feld lässt die Personenart stehen, leer leert sie", func(t *testing.T) {
		anlegen(t, `{"vorname":"Eva","nachname":"Aendern","email":"e@personenart.invalid","rolle":"kollegium","personenart":"liv"}`)
		id, _ := lies(t, "e@personenart.invalid")

		rec := fahre(srv.UpdateUserHandler(userRepo), http.MethodPut, id,
			`{"vorname":"Eva","nachname":"Neu","email":"e@personenart.invalid","rolle":"kollegium","aktiv":true}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Ändern ohne Feld: Status %d: %s", rec.Code, rec.Body.String())
		}
		_, art := lies(t, "e@personenart.invalid")
		erwarte(t, art, "liv")

		rec = fahre(srv.UpdateUserHandler(userRepo), http.MethodPut, id,
			`{"vorname":"Eva","nachname":"Neu","email":"e@personenart.invalid","rolle":"kollegium","aktiv":true,"personenart":""}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Leeren: Status %d: %s", rec.Code, rec.Body.String())
		}
		_, art = lies(t, "e@personenart.invalid")
		erwarte(t, art, "")
	})

	t.Run("die Liste der Benutzerverwaltung liefert die Personenart", func(t *testing.T) {
		anlegen(t, `{"vorname":"Lena","nachname":"Liste","email":"f@personenart.invalid","rolle":"kollegium","personenart":"liv"}`)
		rec := fahre(srv.ListUsersHandler(userRepo), http.MethodGet, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("Liste: Status %d: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"personenart":"liv"`) {
			t.Errorf("die Liste nennt die Personenart nicht: %s", rec.Body.String())
		}
	})
}
