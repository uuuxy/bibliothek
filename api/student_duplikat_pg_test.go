package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// TestCreateStudent_NamensvetternOhneGeburtsdatum sichert #5 (Zwillings-Blockade) ab:
// Zwei namensgleiche Schüler OHNE Geburtsdatum müssen beide anlegbar sein — ein fehlendes
// Geburtsdatum ist kein Duplikat-Kriterium. Vorher stülpte coalesce(...,'1900-01-01')
// beiden Seiten dasselbe Ersatzdatum über und machte den zweiten "Leon Müller"
// fälschlich zum Duplikat (409), sodass er gar nicht angelegt werden konnte.
func TestCreateStudent_NamensvetternOhneGeburtsdatum(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	if code, body := createStudent(t, srv, `{"vorname":"Leon","nachname":"Müller","klasse":"5a"}`); code != http.StatusCreated {
		t.Fatalf("erster Leon Müller: erwartet 201, war %d: %s", code, body)
	}
	if code, body := createStudent(t, srv, `{"vorname":"Leon","nachname":"Müller","klasse":"5b"}`); code != http.StatusCreated {
		t.Fatalf("zweiter Leon Müller (ohne Geburtsdatum) blockiert: erwartet 201, war %d: %s", code, body)
	}
}

// TestCreateStudent_EchtesDuplikatMitGeburtsdatum: Bei gleichem Namen UND gleichem,
// bekanntem Geburtsdatum bleibt die Duplikatssperre aktiv (409) — der Fix schwächt die
// echte Duplikaterkennung nicht ab.
func TestCreateStudent_EchtesDuplikatMitGeburtsdatum(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	if code, body := createStudent(t, srv, `{"vorname":"Anna","nachname":"Schmidt","klasse":"6a","geburtsdatum":"2012-03-04"}`); code != http.StatusCreated {
		t.Fatalf("erste Anna Schmidt: erwartet 201, war %d: %s", code, body)
	}
	if code, body := createStudent(t, srv, `{"vorname":"Anna","nachname":"Schmidt","klasse":"6b","geburtsdatum":"2012-03-04"}`); code != http.StatusConflict {
		t.Fatalf("echtes Duplikat (gleicher Name + Geburtsdatum): erwartet 409, war %d: %s", code, body)
	}
}

func createStudent(t *testing.T, srv *Server, jsonBody string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/schueler", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.CreateStudentHandler().ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}
