package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Das Foto-GET wurde nie ausgeliefert: Drei Stellen im Code bauen die Bild-URL als
// /api/schueler/<barcode>/photo (photo_service.go, student_profile_queries.go,
// resolveFotoURL), aber die Route hieß {id} — und ValidateUUIDParamsMiddleware prüft
// genau diesen Namen gegen das UUID-Format. Jeder echte Barcode ("S-abg1-ms3p73…")
// lief damit in 400, bevor der Handler überhaupt lief.
//
// Der Test hält beide Hälften fest: dass der Platzhalter der Produktionsroute NICHT
// wieder "id" heißt, und warum das den Unterschied macht.
func TestFotoGetRouteNutztBarcodePlatzhalter(t *testing.T) {
	inhalt, err := os.ReadFile("routes_students.go")
	if err != nil {
		t.Fatalf("routes_students.go lesen: %v", err)
	}

	route := regexp.MustCompile(`mux\.Handle\("GET /api/schueler/\{([a-z_]+)\}/photo"`)
	m := route.FindStringSubmatch(string(inhalt))
	if m == nil {
		t.Fatal("GET-Foto-Route nicht gefunden — Registrierungs-Syntax geändert? Dann diesen Test mitziehen.")
	}
	if m[1] == "id" {
		t.Errorf(`Foto-GET nutzt wieder den Platzhalter {id}. ValidateUUIDParamsMiddleware prüft `+
			`ausschließlich "id" gegen das UUID-Format und weist damit jeden Barcode mit 400 ab — `+
			`genau die URL, die resolveFotoURL erzeugt. Gefunden: {%s}`, m[1])
	}
}

// Gegenprobe am Verhalten: Unter dem Namen "id" lehnt die Prüfung einen Barcode ab,
// unter jedem anderen Namen reicht sie ihn durch. Ohne diesen Nachweis wüsste man
// nicht, ob die Umbenennung oben überhaupt etwas bewirkt.
func TestUUIDPruefungGreiftNurBeimParameterID(t *testing.T) {
	const barcode = "S-abg1-ms3p7309e19de436"

	baue := func(muster string) *http.ServeMux {
		mux := http.NewServeMux()
		mux.Handle(muster, ValidateUUIDParamsMiddleware(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })))
		return mux
	}

	rufe := func(mux *http.ServeMux) int {
		req := httptest.NewRequest(http.MethodGet, "/api/schueler/"+barcode+"/photo", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := rufe(baue("GET /api/schueler/{id}/photo")); got != http.StatusBadRequest {
		t.Errorf("mit {id}: Status %d; want 400 — die Vorbedingung des Bugs stimmt nicht mehr", got)
	}
	if got := rufe(baue("GET /api/schueler/{barcode_id}/photo")); got != http.StatusOK {
		t.Errorf("mit {barcode_id}: Status %d; want 200 (Barcode muss durchkommen)", got)
	}
}

// Die Prüfung deckt jetzt auch {schueler_id} und {ausleihe_id} ab (beide Spalten sind
// laut schema.sql UUID). Entscheidend ist die Gegenprobe: Parameter, die KEINE UUID
// tragen, dürfen nicht mitgeprüft werden — sonst wiederholt sich der Foto-Fehler an
// anderer Stelle. {klasse} ist ein Klassenname, {barcode_id} ein Barcode.
func TestUUIDPruefungDecktNurEchteUUIDParameterAb(t *testing.T) {
	faelle := []struct {
		name      string
		muster    string
		pfad      string
		willBlock bool
	}{
		{"schueler_id mit Unsinn wird geprüft", "GET /api/print/rechnung/{schueler_id}", "/api/print/rechnung/keine-uuid", true},
		{"schueler_id mit UUID kommt durch", "GET /api/print/rechnung/{schueler_id}", "/api/print/rechnung/11111111-2222-3333-4444-555555555555", false},
		{"ausleihe_id mit Unsinn wird geprüft", "POST /api/ausleihen/{ausleihe_id}/x", "/api/ausleihen/keine-uuid/x", true},
		{"klasse bleibt unangetastet", "DELETE /api/klassen-mapping/{klasse}", "/api/klassen-mapping/5b", false},
		{"barcode_id bleibt unangetastet", "GET /api/schueler/{barcode_id}/photo", "/api/schueler/S-abg1-ms3p7309e19de436/photo", false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.Handle(f.muster, ValidateUUIDParamsMiddleware(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })))

			methode := strings.Fields(f.muster)[0]
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(methode, f.pfad, nil))

			blockiert := rec.Code == http.StatusBadRequest
			if blockiert != f.willBlock {
				t.Errorf("%s → Status %d (blockiert=%v); want blockiert=%v", f.pfad, rec.Code, blockiert, f.willBlock)
			}
		})
	}
}

// Upload und Auslieferung liegen auf demselben Pfad, tragen aber verschiedene
// Platzhalternamen. Go's ServeMux paniked bei kollidierenden Mustern — dieser Test
// belegt, dass die Kombination zulässig ist und beide Methoden ihr Ziel finden.
func TestFotoRoutenKollidierenNicht(t *testing.T) {
	mux := http.NewServeMux()
	ziel := ""
	mux.HandleFunc("POST /api/schueler/{id}/photo", func(w http.ResponseWriter, r *http.Request) {
		ziel = "upload:" + r.PathValue("id")
	})
	mux.HandleFunc("GET /api/schueler/{barcode_id}/photo", func(w http.ResponseWriter, r *http.Request) {
		ziel = "serve:" + r.PathValue("barcode_id")
	})

	const uuid = "11111111-2222-3333-4444-555555555555"
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/schueler/"+uuid+"/photo", nil))
	if ziel != "upload:"+uuid {
		t.Errorf("POST landete bei %q; want upload mit UUID", ziel)
	}

	const barcode = "S-abg1-ms3p7309e19de436"
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/schueler/"+barcode+"/photo", nil))
	if !strings.HasPrefix(ziel, "serve:") || ziel != "serve:"+barcode {
		t.Errorf("GET landete bei %q; want serve mit Barcode", ziel)
	}
}
