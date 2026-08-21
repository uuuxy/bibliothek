package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bibliothek/db"
)

// TestCreateStudent_OhneGeburtsdatumAbgewiesen: Das Geburtsdatum ist bei der Handanlage
// Pflicht — es ist der einzige Schlüssel, über den der LUSD-Import (ohne Schüler-ID im
// Export der Schule) einen von Hand angelegten Schüler wiedererkennt. Ohne Datum
// entstünde beim nächsten Import ein Duplikat. Die Meldung muss den Grund nennen,
// sonst wird die Pflicht als Schikane gelesen.
func TestCreateStudent_OhneGeburtsdatumAbgewiesen(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	for _, body := range []string{
		`{"vorname":"Leon","nachname":"Müller","klasse":"5a"}`,
		`{"vorname":"Leon","nachname":"Müller","klasse":"5a","geburtsdatum":""}`,
		`{"vorname":"Leon","nachname":"Müller","klasse":"5a","geburtsdatum":"   "}`,
	} {
		code, resp := createStudent(t, srv, body)
		if code != http.StatusBadRequest {
			t.Fatalf("ohne Geburtsdatum: erwartet 400, war %d: %s (Body %s)", code, resp, body)
		}
		if !strings.Contains(resp, "LUSD-Import") {
			t.Errorf("Meldung muss den Grund (LUSD-Import) nennen: %s", resp)
		}
	}
	var anzahl int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM schueler WHERE nachname='Müller'`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	if anzahl != 0 {
		t.Fatalf("trotz 400 wurden %d Zeilen angelegt", anzahl)
	}
}

// TestCreateStudent_NamensvetternMitVerschiedenemGeburtsdatum sichert die Zwillings-
// Regel (#5): Namensgleiche Schüler mit VERSCHIEDENEM Geburtsdatum sind verschiedene
// Personen und müssen beide anlegbar sein.
func TestCreateStudent_NamensvetternMitVerschiedenemGeburtsdatum(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	srv := &Server{DB: &db.Database{Pool: pool}}

	if code, body := createStudent(t, srv, `{"vorname":"Leon","nachname":"Müller","klasse":"5a","geburtsdatum":"2012-01-02"}`); code != http.StatusCreated {
		t.Fatalf("erster Leon Müller: erwartet 201, war %d: %s", code, body)
	}
	if code, body := createStudent(t, srv, `{"vorname":"Leon","nachname":"Müller","klasse":"5b","geburtsdatum":"2013-03-04"}`); code != http.StatusCreated {
		t.Fatalf("zweiter Leon Müller (anderes Geburtsdatum) blockiert: erwartet 201, war %d: %s", code, body)
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
