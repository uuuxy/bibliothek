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

// Ein geleertes Stammdatenfeld muss auch in der Datenbank leer werden.
//
// Das Formular (useStudentEditForm.svelte.js) baut seine Nutzlast als
// `strasse: formData.strasse || null` — ein geräumtes Feld geht also als JSON-null
// raus. In patchStudentRequest ist die Spalte ein *string; JSON-null landet dort als
// nil, und addStr überspringt nil ("nicht mitgeschickt"). Ergebnis: Der alte Wert
// bleibt stehen, die Oberfläche meldet "Änderungen gespeichert", und beim nächsten
// Öffnen der Akte steht die gelöschte Adresse wieder da.
//
// Das betrifft ausgerechnet die Felder, deren Löschung jemand verlangen kann:
// eltern_email und die Postanschrift.
func TestSchuelerFeldLeeren_WirktInDerDatenbank(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr,
		                      strasse, hausnummer, plz, ort, eltern_email)
		VALUES ('Leer','Test','07a','LEER-1', 2030,
		        'Hauptstr','12','60311','Frankfurt','eltern@example.org')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	fahre := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(repository.NewAuditRepository(pool))(rec, req)
		return rec
	}

	// GENAU der Rumpf, den das Formular schickt, wenn jemand Adresse und
	// Eltern-Mail räumt und speichert.
	rec := fahre(`{"vorname":"Leer","nachname":"Test","geburtsdatum":null,"lusd_id":"",
	          "klasse":"07a","barcode_id":"LEER-1","abgaenger_jahr":2030,
	          "strasse":"","hausnummer":"","plz":"","ort":"","eltern_email":""}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
	}

	var strasse, hausnummer, plz, ort, elternMail string
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(strasse,''), coalesce(hausnummer,''), coalesce(plz,''),
		       coalesce(ort,''), coalesce(eltern_email,'')
		FROM schueler WHERE id = $1`, id).
		Scan(&strasse, &hausnummer, &plz, &ort, &elternMail); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}

	for _, f := range []struct{ name, wert string }{
		{"strasse", strasse}, {"hausnummer", hausnummer},
		{"plz", plz}, {"ort", ort}, {"eltern_email", elternMail},
	} {
		if f.wert != "" {
			t.Errorf("%s steht nach dem Leeren immer noch auf %q — die Oberfläche hat "+
				"\"gespeichert\" gemeldet, geändert hat sich nichts.", f.name, f.wert)
		}
	}
}

// Die Gegenrichtung derselben Regel: Pflichtfelder lassen sich NICHT wegräumen.
//
// Ohne diese Hälfte wäre der Fix oben ein Tausch von einem Schaden gegen einen
// grösseren: Sobald das Formular geräumte Felder als leeren String schickt, würde ein
// leerer Vorname genauso ankommen — und der PATCH schrieb ihn bis zum 23.08.2026 mit
// 200 durch. Bei der Klasse kam ein zweiter Schaden dazu: calculateAbgaengerJahr
// leitete aus dem leeren Namen noch ein Abgängerjahr ab.
func TestSchuelerPflichtfeldLeeren_WirdAbgelehnt(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		VALUES ('Echt','Name','07a','PFLICHT-1', 2030) RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}

	for _, rumpf := range []string{
		`{"vorname":""}`, `{"nachname":""}`, `{"barcode_id":""}`, `{"klasse":""}`,
		`{"vorname":"   "}`,
	} {
		req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(rumpf))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
			&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
		rec := httptest.NewRecorder()
		srv.PatchStudentHandler(repository.NewAuditRepository(pool))(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s gab %d statt 400 — ein Pflichtfeld liess sich leeren", rumpf, rec.Code)
		}
	}

	var v, n, b, k string
	if err := pool.QueryRow(ctx, `
		SELECT coalesce(vorname,''), coalesce(nachname,''), coalesce(barcode_id,''), coalesce(klasse,'')
		FROM schueler WHERE id = $1`, id).Scan(&v, &n, &b, &k); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	if v != "Echt" || n != "Name" || b != "PFLICHT-1" {
		t.Errorf("Stammdaten wurden trotz Ablehnung verändert: %q %q %q", v, n, b)
	}
	// Die Klasse wird nur auf "nicht leer" geprüft, nicht auf "07a": Der Versetzungslauf
	// (POST /api/students/promote) schreibt in seinem eigenen Test JEDE Schülerzeile und
	// zieht diese hier mit. Der Anspruch des Tests ist ohnehin "das Pflichtfeld wurde
	// nicht geräumt" — welchen Jahrgang die Zeile zwischenzeitlich trägt, gehört nicht
	// dazu. Als exakter Vergleich war der Test allein grün und in der vollen Suite rot.
	if k == "" {
		t.Error("die Klasse wurde trotz Ablehnung geleert")
	}
}
