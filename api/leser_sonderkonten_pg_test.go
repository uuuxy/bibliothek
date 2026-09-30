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

// Sonderkonten als Art des Lesers (Migration 153, Entscheidung vom 30.09.2026): Praktikum und
// Fachbereich brauchen keinen Zugang zu „Mein Portal" — ein Fachbereich ist ein Sammelkonto,
// das die Kollegen des Fachs benutzen, ein Praktikant leiht aus, meldet sich aber nicht an.
// Bis dahin verlangte jede Neuanlage im Kollegium eine Schul-E-Mail; ein neues
// Fachbereich-Konto oder ein Praktikant ohne Schuladresse ließ sich gar nicht anlegen.
// Sekretariat und U-plus bleiben wie eine Lehrkraft: mit Adresse und Konto.
//
// Geprüft über die echten Handler, Anlegen wie Akte.
func TestSonderkonten_AnlegenUndAkte(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	claims := &auth.Claims{UserID: "00000000-0000-0000-0000-0000000000aa", Rolle: auth.RoleAdmin}

	anlegen := func(t *testing.T, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/schueler", strings.NewReader(rumpf))
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		rec := httptest.NewRecorder()
		srv.CreateStudentHandler()(rec, req)
		return rec
	}
	patch := func(t *testing.T, id, rumpf string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(rumpf))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey, claims))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(repository.NewAuditRepository(pool))(rec, req)
		return rec
	}
	zeile := func(t *testing.T, nachname string) (id, art string, konten int) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			SELECT l.id::text, l.art, (SELECT count(*) FROM benutzer b WHERE b.leser_id = l.id)
			  FROM leser l WHERE l.nachname = $1`, nachname).Scan(&id, &art, &konten); err != nil {
			t.Fatalf("Leserzeile %s lesen: %v", nachname, err)
		}
		return id, art, konten
	}

	t.Run("Praktikum ohne Schul-E-Mail: Leserzeile ohne Konto", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"Paul","nachname":"Praktikant","art":"praktikum"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("Antwort %d (erwartet 201): %s", rec.Code, rec.Body.String())
		}
		if _, art, konten := zeile(t, "Praktikant"); art != "praktikum" || konten != 0 {
			t.Errorf("Art %q mit %d Konten, erwartet praktikum ohne Konto", art, konten)
		}
	})

	t.Run("Fachbereich mit Schul-E-Mail wird abgewiesen", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"Fachbereich","nachname":"Erdkunde","art":"fachbereich",
			"email":"erdkunde@schule.invalid"}`)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "keinen Zugang") {
			t.Fatalf("Antwort %d (erwartet 400 „keinen Zugang“): %s", rec.Code, rec.Body.String())
		}
		if n := zaehleLeser(t, pool, "Erdkunde"); n != 0 {
			t.Errorf("trotz Ablehnung steht eine Leserzeile da (%d)", n)
		}
	})

	t.Run("Sekretariat braucht die Schul-E-Mail wie eine Lehrkraft", func(t *testing.T) {
		if rec := anlegen(t, `{"vorname":"Sabine","nachname":"Sekretariat","art":"sekretariat"}`); rec.Code != http.StatusBadRequest {
			t.Fatalf("ohne Adresse: Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		rec := anlegen(t, `{"vorname":"Sabine","nachname":"Sekretariat","art":"sekretariat",
			"email":"sabine.sekretariat@schule.invalid"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("mit Adresse: Antwort %d (erwartet 201): %s", rec.Code, rec.Body.String())
		}
		if _, art, konten := zeile(t, "Sekretariat"); art != "sekretariat" || konten != 1 {
			t.Errorf("Art %q mit %d Konten, erwartet sekretariat mit einem Konto", art, konten)
		}
	})

	t.Run("Akte: Praktikum bekommt keine nachgetragene Schul-E-Mail", func(t *testing.T) {
		id, _, _ := zeile(t, "Praktikant")
		rec := patch(t, id, `{"vorname":"Paul","nachname":"Praktikant","art":"praktikum",
			"email":"paul.praktikant@schule.invalid"}`)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "keinen Zugang") {
			t.Fatalf("Antwort %d (erwartet 400 „keinen Zugang“): %s", rec.Code, rec.Body.String())
		}
		if _, _, konten := zeile(t, "Praktikant"); konten != 0 {
			t.Errorf("%d Konten am Praktikum, erwartet keins", konten)
		}
	})

	t.Run("Akte: aus Praktikum wird Lehrkraft, im selben Speichern mit Konto", func(t *testing.T) {
		id, _, _ := zeile(t, "Praktikant")
		rec := patch(t, id, `{"vorname":"Paul","nachname":"Praktikant","art":"lehrkraft",
			"email":"paul.praktikant@schule.invalid"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Antwort %d (erwartet 200): %s", rec.Code, rec.Body.String())
		}
		if _, art, konten := zeile(t, "Praktikant"); art != "lehrkraft" || konten != 1 {
			t.Errorf("Art %q mit %d Konten, erwartet lehrkraft mit einem Konto", art, konten)
		}
	})

	t.Run("Akte: aus einer Lehrkraft ohne Konto wird ein Fachbereich, ohne Konto", func(t *testing.T) {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO leser (vorname, nachname, art)
			VALUES ('Fachbereich', 'Chemie', 'lehrkraft') RETURNING id::text`).Scan(&id); err != nil {
			t.Fatalf("Altbestand anlegen: %v", err)
		}
		rec := patch(t, id, `{"vorname":"Fachbereich","nachname":"Chemie","art":"fachbereich",
			"email":"chemie@schule.invalid"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("mit Adresse: Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if rec := patch(t, id, `{"vorname":"Fachbereich","nachname":"Chemie","art":"fachbereich","email":""}`); rec.Code != http.StatusOK {
			t.Fatalf("ohne Adresse: Antwort %d (erwartet 200): %s", rec.Code, rec.Body.String())
		}
		if _, art, konten := zeile(t, "Chemie"); art != "fachbereich" || konten != 0 {
			t.Errorf("Art %q mit %d Konten, erwartet fachbereich ohne Konto", art, konten)
		}
	})

	t.Run("Unbekannte Art nennt alle sieben", func(t *testing.T) {
		rec := anlegen(t, `{"vorname":"X","nachname":"Y","art":"hausmeister"}`)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Praktikum, Sekretariat, U-plus, Fachbereich") {
			t.Fatalf("Antwort %d: %s", rec.Code, rec.Body.String())
		}
	})
}
