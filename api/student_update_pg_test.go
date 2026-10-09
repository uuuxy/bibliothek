package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"
)

// aendereLeser schickt einen PATCH an die Akte eines Lesers, mit den Rechten der Verwaltung.
func aendereLeser(t *testing.T, srv *Server, id, rumpf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/schueler/"+id, strings.NewReader(rumpf))
	req.SetPathValue("id", id)
	req = req.WithContext(context.WithValue(req.Context(), auth.ClaimsContextKey,
		&auth.Claims{UserID: id, Rolle: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	srv.PatchStudentHandler(repository.NewAuditRepository(srv.DB.Pool))(rec, req)
	return rec
}

// Jedes Feld der Änderung landet in seiner eigenen Spalte. Die Werte sind je Feld verschieden:
// Zwei vertauschte Spalten gleichen Typs (Ort und Straße, Vor- und Nachname) fielen mit
// gleichen Werten nicht auf.
func TestLeserAendern_JedesFeldLandetInSeinerSpalte(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, aktualisiert_am)
		VALUES ('Alt','Stand','05a','FELD-ALT', 2031, '2020-01-01')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}

	rec := aendereLeser(t, srv, id, `{"vorname":"Vorn","nachname":"Nachn","klasse":"07b",
		"barcode_id":"FELD-NEU","abgaenger_jahr":2033,"geburtsdatum":"2012-03-04",
		"strasse":"Strassenweg","hausnummer":"17c","plz":"60311","ort":"Ortsname",
		"eltern_email":"eltern@example.org","lusd_id":"LUSD-4711"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
	}

	var ist struct {
		vorname, nachname, klasse, barcode, geburtsdatum string
		strasse, hausnummer, plz, ort, elternEmail, lusd string
		abgaengerJahr                                    int
		aktualisiert                                     time.Time
	}
	if err := pool.QueryRow(ctx, `
		SELECT vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum::text,
		       strasse, hausnummer, plz, ort, eltern_email, lusd_id, aktualisiert_am
		FROM leser WHERE id = $1`, id).Scan(
		&ist.vorname, &ist.nachname, &ist.klasse, &ist.barcode, &ist.abgaengerJahr, &ist.geburtsdatum,
		&ist.strasse, &ist.hausnummer, &ist.plz, &ist.ort, &ist.elternEmail, &ist.lusd, &ist.aktualisiert,
	); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}

	for _, f := range []struct{ spalte, ist, soll string }{
		{"vorname", ist.vorname, "Vorn"},
		{"nachname", ist.nachname, "Nachn"},
		{"klasse", ist.klasse, "07B"},
		{"barcode_id", ist.barcode, "FELD-NEU"},
		{"geburtsdatum", ist.geburtsdatum, "2012-03-04"},
		{"strasse", ist.strasse, "Strassenweg"},
		{"hausnummer", ist.hausnummer, "17c"},
		{"plz", ist.plz, "60311"},
		{"ort", ist.ort, "Ortsname"},
		{"eltern_email", ist.elternEmail, "eltern@example.org"},
		{"lusd_id", ist.lusd, "LUSD-4711"},
	} {
		if f.ist != f.soll {
			t.Errorf("%s steht auf %q, geschickt war %q", f.spalte, f.ist, f.soll)
		}
	}
	if ist.abgaengerJahr != 2033 {
		t.Errorf("abgaenger_jahr steht auf %d, geschickt war 2033", ist.abgaengerJahr)
	}
	if ist.aktualisiert.Year() == 2020 {
		t.Errorf("aktualisiert_am steht noch auf %s, die Änderung hat den Zeitpunkt nicht gesetzt", ist.aktualisiert)
	}
}

// Geleerte Anschrift und geleerter Elternkontakt stehen als NULL in der Zeile, nicht als leerer
// Text: Der DSGVO-Lauf und die LUSD-Ausleitung schreiben „gelöscht" ebenso, und eine Abfrage
// auf IS NULL fände einen leeren Text nicht.
func TestLeserAendern_LeererWertWirdNull(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()
	srv := &Server{DB: &db.Database{Pool: pool}}

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr,
		                      strasse, hausnummer, plz, ort, eltern_email)
		VALUES ('Null','Probe','07a','NULL-1', 2030,
		        'Hauptstr','12','60311','Frankfurt','eltern@example.org')
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Leser anlegen: %v", err)
	}

	rec := aendereLeser(t, srv, id, `{"strasse":"","hausnummer":"  ","plz":"","ort":" ","eltern_email":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH gab %d: %s", rec.Code, rec.Body.String())
	}

	var strasse, hausnummer, plz, ort, elternEmail bool
	if err := pool.QueryRow(ctx, `
		SELECT strasse IS NULL, hausnummer IS NULL, plz IS NULL, ort IS NULL, eltern_email IS NULL
		FROM leser WHERE id = $1`, id).Scan(&strasse, &hausnummer, &plz, &ort, &elternEmail); err != nil {
		t.Fatalf("zurücklesen: %v", err)
	}
	for _, f := range []struct {
		spalte  string
		istNull bool
	}{
		{"strasse", strasse}, {"hausnummer", hausnummer}, {"plz", plz}, {"ort", ort},
		{"eltern_email", elternEmail},
	} {
		if !f.istNull {
			t.Errorf("%s ist nach dem Leeren nicht NULL", f.spalte)
		}
	}
}

// Eine Kennung, die es nicht gibt, ist „nicht gefunden" und kein Erfolg: Die Anweisung trifft
// keine Zeile, und ohne diese Auskunft meldete die Akte „gespeichert".
func TestLeserAendern_UnbekannteKennungIstNichtGefunden(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	rec := aendereLeser(t, srv, "00000000-0000-4000-8000-000000000000", `{"vorname":"Niemand"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PATCH auf eine unbekannte Kennung gab %d statt 404: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "leser nicht gefunden") {
		t.Errorf("die Meldung nennt den Grund nicht: %s", rec.Body.String())
	}
}
