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

// Die Schul-E-Mail in der Akte NACHTRAGEN — die Reparatur für den Altbestand.
//
// Seit dem 16.09.2026 verlangt „Neuer Leser" bei Lehrkraft und LiV die Adresse, und damit
// entsteht das Konto sofort: Die spätere Selbstanmeldung findet es und legt nichts
// Zweites an (leser_kollegium_konto_pg_test.go). Für jeden, der VORHER eingetragen wurde,
// galt das nicht. Er steht ohne Adresse und ohne Konto in der Leserdatei, und die Akte
// kannte die Adresse nicht einmal als Feld — der Doppeleintrag war nur noch HINTERHER zu
// reparieren, durch Zusammenführen.
//
// Geprüft wird hier der ganze Weg über den echten Handler: Adresse nachtragen, Konto
// entsteht an DERSELBEN Leserzeile, und die Grenzen drumherum (Schüler, belegte Adresse,
// vorhandenes Konto, Freischaltungsrecht).
func TestSchulEmailNachtragen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	patch := func(t *testing.T, id, body string, rolle auth.Role) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: "00000000-0000-0000-0000-0000000000aa", Rolle: rolle}))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(repository.NewAuditRepository(pool))(rec, req)
		return rec
	}

	// Ein Kollege von vor dem 16.09.2026: Leserzeile ohne Adresse, ohne Konto. Er wird
	// DIREKT in die Tabelle geschrieben, weil der Anlegeweg genau das heute nicht mehr
	// zulässt — der Altbestand ist der Fall, den es zu reparieren gilt.
	altbestand := func(t *testing.T, vorname, nachname, art string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art) VALUES ($1, $2, $3) RETURNING id::text`,
			vorname, nachname, art).Scan(&id); err != nil {
			t.Fatalf("Altbestand anlegen: %v", err)
		}
		return id
	}

	kontoVon := func(t *testing.T, leserID string) (email, rolle string, aktiv, beantragt bool) {
		t.Helper()
		err := pool.QueryRow(ctx, `
			SELECT COALESCE(email, ''), rolle::text, aktiv, zugang_beantragt_am IS NOT NULL
			  FROM benutzer WHERE leser_id = $1`, leserID).Scan(&email, &rolle, &aktiv, &beantragt)
		if err != nil {
			return "", "", false, false
		}
		return email, rolle, aktiv, beantragt
	}

	t.Run("Mitarbeiter trägt nach: Konto entsteht, freigeschaltet wird es nicht", func(t *testing.T) {
		id := altbestand(t, "Agnes", "Altbestand", "lehrkraft")
		rec := patch(t, id, `{"vorname":"Agnes","nachname":"Altbestand","email":"agnes.altbestand@schule.invalid"}`,
			auth.RoleMitarbeiter)
		if rec.Code != http.StatusOK {
			t.Fatalf("Antwort %d: %s", rec.Code, rec.Body.String())
		}

		email, rolle, aktiv, beantragt := kontoVon(t, id)
		if email != "agnes.altbestand@schule.invalid" {
			t.Fatalf("kein Konto an der Leserzeile (Adresse %q)", email)
		}
		if rolle != "kollegium" {
			t.Errorf("Rolle %q, erwartet kollegium", rolle)
		}
		// Der Kern der Rechte-Paarung: Das Konto entsteht immer (daran hängt der Schutz
		// vor dem Doppel), freigeschaltet wird es nur von dem, der das auch sonst darf.
		if aktiv {
			t.Error("ein Mitarbeiter ohne manage_users darf kein AKTIVES Konto erzeugen")
		}
		if !beantragt {
			t.Error("ohne zugang_beantragt_am sieht der Admin den Antrag nicht in der Freischaltungs-Zeile")
		}

		// EINE Leserzeile: Der Wächter trg_benutzer_hat_leserzeile darf keine zweite
		// angelegt haben — genau der Doppeleintrag, den das hier verhindern soll.
		if n := zaehleLeser(t, pool, "Altbestand"); n != 1 {
			t.Fatalf("erwartet genau eine Leserzeile, gefunden: %d", n)
		}
	})

	t.Run("dieselbe Adresse noch einmal ist kein Fehler", func(t *testing.T) {
		id := altbestand(t, "Wieder", "Holung", "liv")
		if rec := patch(t, id, `{"nachname":"Holung","email":"wieder.holung@schule.invalid"}`, auth.RoleAdmin); rec.Code != http.StatusOK {
			t.Fatalf("erster Nachtrag: %d %s", rec.Code, rec.Body.String())
		}
		// Das Formular schickt die angezeigte Adresse bei JEDEM Speichern mit.
		if rec := patch(t, id, `{"nachname":"Holung","email":"wieder.holung@schule.invalid"}`, auth.RoleAdmin); rec.Code != http.StatusOK {
			t.Fatalf("zweites Speichern: %d %s", rec.Code, rec.Body.String())
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM benutzer WHERE leser_id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("Konten zählen: %v", err)
		}
		if n != 1 {
			t.Fatalf("erwartet genau ein Konto, gefunden: %d", n)
		}
	})

	t.Run("eine vorhandene Adresse wird hier nicht überschrieben", func(t *testing.T) {
		id := altbestand(t, "Fest", "Stehend", "lehrkraft")
		if rec := patch(t, id, `{"nachname":"Stehend","email":"fest.stehend@schule.invalid"}`, auth.RoleAdmin); rec.Code != http.StatusOK {
			t.Fatalf("Nachtrag: %d %s", rec.Code, rec.Body.String())
		}
		rec := patch(t, id, `{"nachname":"Stehend","email":"ganz.anders@schule.invalid"}`, auth.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Benutzerverwaltung") {
			t.Errorf("die Meldung soll die zuständige Tür nennen: %s", rec.Body.String())
		}
		if email, _, _, _ := kontoVon(t, id); email != "fest.stehend@schule.invalid" {
			t.Errorf("die Adresse am Konto wurde verändert: %q", email)
		}
	})

	t.Run("die Adresse lässt sich nicht entfernen", func(t *testing.T) {
		id := altbestand(t, "Nicht", "Weg", "lehrkraft")
		if rec := patch(t, id, `{"nachname":"Weg","email":"nicht.weg@schule.invalid"}`, auth.RoleAdmin); rec.Code != http.StatusOK {
			t.Fatalf("Nachtrag: %d %s", rec.Code, rec.Body.String())
		}
		rec := patch(t, id, `{"nachname":"Weg","email":""}`, auth.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if email, _, _, _ := kontoVon(t, id); email != "nicht.weg@schule.invalid" {
			t.Errorf("die Adresse am Konto ist weg: %q", email)
		}
	})

	t.Run("ein Schüler bekommt kein Konto", func(t *testing.T) {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, klasse, abgaenger_jahr, barcode_id, geburtsdatum)
			VALUES ('Kind', 'Kontolos', 'schueler', '07A', 2031, 'A-SCHULMAIL-1', '2012-05-04')
			RETURNING id::text`).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		rec := patch(t, id, `{"nachname":"Kontolos","email":"kind@schule.invalid"}`, auth.RoleAdmin)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if email, _, _, _ := kontoVon(t, id); email != "" {
			t.Errorf("der Schüler hat ein Konto bekommen: %q", email)
		}
	})

	t.Run("eine belegte Adresse ist eine Auskunft", func(t *testing.T) {
		erster := altbestand(t, "Erster", "Belegt", "lehrkraft")
		if rec := patch(t, erster, `{"nachname":"Belegt","email":"belegt@schule.invalid"}`, auth.RoleAdmin); rec.Code != http.StatusOK {
			t.Fatalf("Nachtrag: %d %s", rec.Code, rec.Body.String())
		}
		zweiter := altbestand(t, "Zweiter", "Belegt2", "lehrkraft")
		rec := patch(t, zweiter, `{"nachname":"Belegt2","email":"belegt@schule.invalid"}`, auth.RoleAdmin)
		if rec.Code != http.StatusConflict {
			t.Fatalf("Antwort %d (erwartet 409): %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "bereits ein Zugang") {
			t.Errorf("die Meldung soll auf den vorhandenen Eintrag zeigen: %s", rec.Body.String())
		}
	})

	t.Run("die Akte zeigt die Adresse", func(t *testing.T) {
		id := altbestand(t, "Sicht", "Bar", "lehrkraft")
		if rec := patch(t, id, `{"nachname":"Bar","email":"sicht.bar@schule.invalid"}`, auth.RoleAdmin); rec.Code != http.StatusOK {
			t.Fatalf("Nachtrag: %d %s", rec.Code, rec.Body.String())
		}

		// Ohne die Adresse im Profil kann das Formular sie nicht anzeigen — und was es
		// nicht anzeigt, schickt es beim Speichern auch nicht mit.
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+id, nil)
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.GetStudentProfileHandler(repository.NewStudentRepository(pool))(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Profil: %d %s", rec.Code, rec.Body.String())
		}
		var antwort map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("Profil lesen: %v", err)
		}
		if antwort["email"] != "sicht.bar@schule.invalid" {
			t.Errorf("die Akte zeigt %q statt der Schul-Adresse", antwort["email"])
		}
	})
}

// Die Ausweisnummer eines Kollegen lässt sich wieder entfernen — die des Schülers nicht.
//
// Peters Blick auf die fertige Maske am 16.09.2026 brachte den Fall ans Licht: Der
// Hinweis unter dem Feld sagt dem Kollegen „Leer lassen, solange kein Ausweis gedruckt
// ist" — leeren ließ es sich aber nicht mehr, sobald einmal etwas drinstand. Das Formular
// ließ das leere Feld weg und meldete Erfolg, der Server hätte es mit 400 abgewiesen. Ein
// Tippfehler in der Nummer war damit endgültig.
func TestAusweisnummerLeeren(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	patch := func(t *testing.T, id, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: "00000000-0000-0000-0000-0000000000aa", Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(repository.NewAuditRepository(pool))(rec, req)
		return rec
	}

	t.Run("beim Kollegen wird die Spalte NULL", func(t *testing.T) {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, barcode_id)
			VALUES ('Tipp', 'Fehler', 'lehrkraft', 'A-TIPPFEHLER')
			RETURNING id::text`).Scan(&id); err != nil {
			t.Fatalf("Kollegen anlegen: %v", err)
		}
		if rec := patch(t, id, `{"nachname":"Fehler","barcode_id":""}`); rec.Code != http.StatusOK {
			t.Fatalf("Antwort %d: %s", rec.Code, rec.Body.String())
		}
		// NULL und nicht der leere String: `uniq_schueler_barcode_active` ließe genau
		// einen zweiten Leser mit "" nicht zu.
		var nummer *string
		if err := pool.QueryRow(ctx, `SELECT barcode_id FROM leser WHERE id = $1`, id).Scan(&nummer); err != nil {
			t.Fatalf("Ausweisnummer lesen: %v", err)
		}
		if nummer != nil {
			t.Fatalf("die Nummer steht noch da: %q", *nummer)
		}
	})

	t.Run("zwei Kollegen dürfen zugleich ohne Nummer dastehen", func(t *testing.T) {
		for _, name := range []string{"OhneA", "OhneB"} {
			var id string
			if err := pool.QueryRow(ctx, `
				INSERT INTO leser (vorname, nachname, art, barcode_id)
				VALUES ('Leer', $1, 'lehrkraft', $2) RETURNING id::text`,
				name, "A-"+name).Scan(&id); err != nil {
				t.Fatalf("Kollegen anlegen: %v", err)
			}
			if rec := patch(t, id, `{"nachname":"`+name+`","barcode_id":""}`); rec.Code != http.StatusOK {
				t.Fatalf("%s: Antwort %d: %s", name, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("beim Schüler bleibt sie Pflicht, mit Begründung", func(t *testing.T) {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO leser (vorname, nachname, art, klasse, abgaenger_jahr, barcode_id, geburtsdatum)
			VALUES ('Kind', 'MitAusweis', 'schueler', '07A', 2031, 'A-KIND-1', '2012-05-04')
			RETURNING id::text`).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		rec := patch(t, id, `{"nachname":"MitAusweis","barcode_id":""}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Antwort %d (erwartet 400): %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "Theke") {
			t.Errorf("die Meldung soll den Grund nennen: %s", rec.Body.String())
		}
		var nummer string
		if err := pool.QueryRow(ctx, `SELECT COALESCE(barcode_id, '') FROM leser WHERE id = $1`, id).Scan(&nummer); err != nil {
			t.Fatalf("Ausweisnummer lesen: %v", err)
		}
		if nummer != "A-KIND-1" {
			t.Errorf("die Nummer des Schülers wurde verändert: %q", nummer)
		}
	})
}
