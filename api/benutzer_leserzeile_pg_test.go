package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// Wer ein Konto anlegt, legt einen LESER an (Migration 125): Der Mensch bekommt seinen
// Platz in der Lesertabelle, das Konto bleibt Anmeldung und Rechte. Ohne diese Zeile fände
// die Theke die Person nicht — genau der Fehler, den der Umbau abschafft.
//
// Geprüft am Live-Pfad, also über die Handler der Benutzerverwaltung und nicht über rohes
// SQL: Der Trigger allein beweist nur, dass die Datenbank es täte; dieser Test beweist,
// dass die Verwaltung auch dort ankommt, wo die Theke nachsieht.
func TestBenutzerBekommtLeserzeile(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}
	userRepo := repository.NewUserRepository(pool)
	admin := &auth.Claims{UserID: "00000000-0000-0000-0000-00000000a125", Rolle: auth.RoleAdmin}
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM leser WHERE id IN
			(SELECT leser_id FROM benutzer WHERE email LIKE '%@leserzeile.invalid')`); err != nil {
			t.Errorf("aufräumen (Leser): %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM benutzer WHERE email LIKE '%@leserzeile.invalid'`); err != nil {
			t.Errorf("aufräumen (Konten): %v", err)
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
	anlegen := func(t *testing.T, body string) {
		t.Helper()
		if rec := fahre(srv.CreateUserHandler(userRepo), http.MethodPost, "", body); rec.Code != http.StatusOK {
			t.Fatalf("Anlegen (%s): Status %d, %s", body, rec.Code, rec.Body.String())
		}
	}
	// liesLeser liefert Art, Ausweis und Namen der LESERZEILE hinter einem Konto.
	liesLeser := func(t *testing.T, email string) (kontoID, art, ausweis, name string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
			SELECT b.id, coalesce(l.art, ''), coalesce(l.barcode_id, ''),
			       coalesce(l.vorname || ' ' || l.nachname, '')
			FROM benutzer b LEFT JOIN leser l ON l.id = b.leser_id
			WHERE b.email = $1`, email).Scan(&kontoID, &art, &ausweis, &name); err != nil {
			t.Fatalf("Konto %s lesen: %v", email, err)
		}
		return
	}

	t.Run("jede Rolle bekommt eine Leserzeile, keine davon als Schüler", func(t *testing.T) {
		for _, rolle := range []string{"kollegium", "mitarbeiter", "helfer", "leitung", "admin"} {
			email := rolle + "@leserzeile.invalid"
			anlegen(t, `{"vorname":"Kai","nachname":"Konto","email":"`+email+`","rolle":"`+rolle+`"}`)
			_, art, _, _ := liesLeser(t, email)
			// Nicht 'schueler': Diese Zeile darf der LUSD-Abgleich nie als Abgänger
			// markieren und der Löschjob nie anfassen.
			if art != "lehrkraft" {
				t.Errorf("%s: Art der Leserzeile %q, erwartet lehrkraft", rolle, art)
			}
		}
	})

	t.Run("die Ausweisnummer landet an der Leserzeile, nicht am Konto", func(t *testing.T) {
		email := "ausweis@leserzeile.invalid"
		anlegen(t, `{"vorname":"Ada","nachname":"Ausweis","email":"`+email+`","rolle":"mitarbeiter","barcode_id":"LZ-1"}`)
		_, _, ausweis, _ := liesLeser(t, email)
		if ausweis != "LZ-1" {
			t.Errorf("Ausweis an der Leserzeile = %q, erwartet LZ-1", ausweis)
		}
		// Und die Theke findet ihn dort: dieselbe Abfrage, die der Scan stellt.
		leser, err := repository.NewStudentRepository(pool).GetLeserByBarcode(ctx, "LZ-1")
		if err != nil || leser == nil {
			t.Fatalf("die Theke findet den neuen Ausweis nicht (err=%v)", err)
		}
		if leser.Nachname != "Ausweis" {
			t.Errorf("falsche Person am Ausweis: %+v", leser)
		}
	})

	t.Run("eine Namensänderung wandert in die Leserzeile mit", func(t *testing.T) {
		email := "wechsel@leserzeile.invalid"
		anlegen(t, `{"vorname":"Alt","nachname":"Name","email":"`+email+`","rolle":"mitarbeiter"}`)
		kontoID, _, _, _ := liesLeser(t, email)

		rec := fahre(srv.UpdateUserHandler(userRepo), http.MethodPut, kontoID,
			`{"vorname":"Neu","nachname":"Name","email":"`+email+`","rolle":"mitarbeiter","aktiv":true,"barcode_id":"LZ-2"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Ändern: Status %d, %s", rec.Code, rec.Body.String())
		}
		_, _, ausweis, name := liesLeser(t, email)
		// Liefen die beiden auseinander, hieße dieselbe Person an der Theke anders als in
		// der Benutzerverwaltung — und der eingetragene Ausweis bliebe wirkungslos.
		if name != "Neu Name" || ausweis != "LZ-2" {
			t.Errorf("Leserzeile nach der Änderung: %q / %q, erwartet Neu Name / LZ-2", name, ausweis)
		}
	})

	t.Run("die Benutzerliste nennt keine Personenart mehr", func(t *testing.T) {
		rec := fahre(srv.ListUsersHandler(userRepo), http.MethodGet, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("Liste: Status %d", rec.Code)
		}
		var liste []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &liste); err != nil {
			t.Fatalf("Liste lesen: %v", err)
		}
		if len(liste) == 0 {
			t.Fatal("leere Liste — das Gate wäre still grün")
		}
		for _, eintrag := range liste {
			if _, da := eintrag["personenart"]; da {
				t.Fatalf("die Liste trägt noch eine Personenart: %v", eintrag)
			}
		}
	})
}
