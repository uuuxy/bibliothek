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

// Ein zweites Konto derselben Person darf keine zweite Leserzeile erzeugen (OFFEN.md 5.17).
//
// Der Wächter trg_benutzer_hat_leserzeile hängt jedem Konto ohne Leserzeile eine frische
// an. Wird das Konto eines Kollegen gelöscht (seine Leserzeile bleibt — die Ausleihen
// hängen daran), und legt die Benutzerverwaltung ihm später ein neues an, stand er bis zum
// 22.09.2026 zweimal in der Leserdatei: einmal mit Ausweis und Geschichte, einmal leer.
// Repariert wurde das mit „Zusammenführen"; entstehen soll es gar nicht. Der Weg zum
// Konto an einer vorhandenen Leserzeile ist die Schul-E-Mail in der Akte
// (LegeKollegiumskonto) — dorthin verweist die Antwort.
func TestBenutzerAnlegen_KeineZweiteLeserzeileFuerDieselbePerson(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	admin := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a517", Rolle: auth.RoleAdmin}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id IN
			(SELECT leser_id FROM benutzer WHERE email LIKE '%@zweitezeile.invalid')`); err != nil {
			t.Errorf("aufräumen (Leser der Konten): %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@zweitezeile.invalid'`); err != nil {
			t.Errorf("aufräumen (Konten): %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE nachname IN ('Zweitezeile', 'Zweitezeile-Kind')`); err != nil {
			t.Errorf("aufräumen (Leser): %v", err)
		}
	})
	anlegen := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/benutzer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, admin))
		rec := httptest.NewRecorder()
		srv.CreateUserHandler(userRepo).ServeHTTP(rec, req)
		return rec
	}
	zaehleLeser := func(t *testing.T, nachname string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM leser WHERE nachname = $1 AND deleted_at IS NULL`, nachname).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Die Leserzeile eines Kollegen OHNE Konto — so sieht sie nach dem Löschen des Kontos
	// aus, und so kommt sie aus der Littera-Übernahme.
	if _, err := pool.Exec(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art) VALUES ('A-517', 'Kim', 'Zweitezeile', 'lehrkraft')`); err != nil {
		t.Fatal(err)
	}

	rec := anlegen(`{"vorname":"Kim","nachname":"Zweitezeile","email":"kim@zweitezeile.invalid","rolle":"kollegium"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Konto für eine Person mit Leserzeile ohne Konto: Status %d, erwartet 409: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Akte") {
		t.Errorf("die Antwort nennt den Weg nicht (Schul-E-Mail in der Akte): %s", rec.Body.String())
	}
	if n := zaehleLeser(t, "Zweitezeile"); n != 1 {
		t.Errorf("%d Leserzeilen für Kim Zweitezeile, erwartet 1 — die zweite ist genau der Doppeleintrag", n)
	}

	// Gegenprobe 1: Ein Schüler gleichen Namens ist eine andere Person — das Konto entsteht.
	if _, err := pool.Exec(ctx, `INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('S-517', 'Lea', 'Zweitezeile-Kind', '7a', 2030)`); err != nil {
		t.Fatal(err)
	}
	if rec := anlegen(`{"vorname":"Lea","nachname":"Zweitezeile-Kind","email":"lea@zweitezeile.invalid","rolle":"kollegium"}`); rec.Code != http.StatusOK {
		t.Errorf("Namensvetter unter den Schülern darf nicht bremsen: Status %d: %s", rec.Code, rec.Body.String())
	}

	// Gegenprobe 2: Hat der Kollege gleichen Namens schon ein Konto, ist der Neue ein
	// Namensvetter mit eigener Adresse — kein Doppeleintrag, den die Akte auflösen könnte.
	if _, err := pool.Exec(ctx, `INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id)
		VALUES ('Kim', 'Zweitezeile', 'kim.alt@zweitezeile.invalid', 'kollegium', true,
		        (SELECT id FROM leser WHERE barcode_id = 'A-517'))`); err != nil {
		t.Fatal(err)
	}
	if rec := anlegen(`{"vorname":"Kim","nachname":"Zweitezeile","email":"kim.neu@zweitezeile.invalid","rolle":"kollegium"}`); rec.Code != http.StatusOK {
		t.Errorf("Namensvetter mit eigenem Konto darf nicht bremsen: Status %d: %s", rec.Code, rec.Body.String())
	}
}
